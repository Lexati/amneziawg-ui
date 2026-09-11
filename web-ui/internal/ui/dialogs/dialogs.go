// Package dialogs is the one way a dialog is opened on this page.
//
// fyne builds the dismiss/confirm row out of plain widget.Buttons, which report
// the default arrow cursor: right for a desktop window, wrong for a page where
// everything clickable shows a hand. dialog.CustomDialog.SetButtons is the
// supported way to replace that row, so every dialog in this UI is opened
// through one of the helpers below and none of them calls dialog.Show*
// directly.
//
// A dialog built this way carries no callback of its own, so dismissing it with
// Escape reports nothing rather than a "cancelled" - which is what every caller
// here does with a cancel anyway.
package dialogs

import (
	"unicode"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"amneziawg-web-ui/web-ui/internal/ui/browser"
	"amneziawg-web-ui/web-ui/internal/ui/env"
	"amneziawg-web-ui/web-ui/internal/ui/widgets"
)

// Show presents content with a single dismiss button.
func Show(win fyne.Window, title, dismiss string, content fyne.CanvasObject, size fyne.Size) {
	view := dialog.NewCustomWithoutButtons(title, content, win)
	view.SetButtons([]fyne.CanvasObject{widgets.NewButton(dismiss, nil, view.Hide)})
	view.Resize(size)
	view.Show()
}

// ShowForm presents content with the cancel/confirm pair a form needs.
//
// onConfirm reports whether the dialog is finished: returning false leaves it
// open, which is what a form does when what the user typed does not validate -
// closing it first would throw the input away along with the mistake.
func ShowForm(win fyne.Window, title, confirm, dismiss string, content fyne.CanvasObject, size fyne.Size, onConfirm func() bool) {
	view := dialog.NewCustomWithoutButtons(title, content, win)

	cancel := widgets.NewButton(dismiss, theme.CancelIcon(), view.Hide)
	ok := widgets.NewButton(confirm, theme.ConfirmIcon(), func() {
		if !onConfirm() {
			return
		}
		view.Hide()
	})
	ok.Importance = widget.HighImportance

	view.SetButtons([]fyne.CanvasObject{cancel, ok})
	view.Resize(size)
	view.Show()
}

// Confirm asks a yes/no question, the way dialog.ShowConfirm would.
func Confirm(win fyne.Window, title, message string, onConfirm func()) {
	view := dialog.NewCustomWithoutButtons(title, wrappedMessage(message), win)
	view.SetIcon(theme.QuestionIcon())

	no := widgets.NewButton("No", theme.CancelIcon(), view.Hide)
	yes := widgets.NewButton("Yes", theme.ConfirmIcon(), func() {
		view.Hide()
		onConfirm()
	})
	yes.Importance = widget.HighImportance

	view.SetButtons([]fyne.CanvasObject{no, yes})
	view.Resize(messageSize(win, message, view))
	view.Show()
}

// ShowError reports a failed call. Must run on the Fyne thread; components
// reach it through env.Notifier.Fail, which takes care of that.
func ShowError(win fyne.Window, err error) {
	message := err.Error()
	if r, size := utf8.DecodeRuneInString(message); r != utf8.RuneError {
		message = string(unicode.ToUpper(r)) + message[size:]
	}

	view := dialog.NewCustomWithoutButtons("Error", wrappedMessage(message), win)
	view.SetIcon(theme.ErrorIcon())
	view.SetButtons([]fyne.CanvasObject{widgets.NewButton("OK", nil, view.Hide)})
	view.Resize(messageSize(win, message, view))
	view.Show()
}

func wrappedMessage(message string) fyne.CanvasObject {
	return &widget.Label{Text: message, Alignment: fyne.TextAlignCenter, Wrapping: fyne.TextWrapWord}
}

// messageSize is the size fyne would have given a text dialog: wide enough for
// the message on one line, capped at 600px and at most 90% of the window. A
// dialog built with NewCustomWithoutButtons does not run fyne's own sizing
// hook, and a word-wrapped label on its own asks for the width of its longest
// word - a two-line-per-sentence sliver.
func messageSize(win fyne.Window, message string, view *dialog.CustomDialog) fyne.Size {
	// The 32 is fyne's own dialogLayout padding around the content.
	unwrapped := widget.NewLabel(message).MinSize().Width + 32 + theme.Padding()*2
	width := min(unwrapped, 600, win.Canvas().Size().Width*0.9)
	return fyne.NewSize(width, view.MinSize().Height)
}

// Scrolled wraps dialog content so a long form never grows past the viewport.
func Scrolled(content fyne.CanvasObject) fyne.CanvasObject {
	scroll := container.NewVScroll(content)
	scroll.SetMinSize(fyne.NewSize(520, 320))
	return scroll
}

// Size clamps a preferred size to the browser window.
func Size(win fyne.Window, width, height float32) fyne.Size {
	canvasSize := win.Canvas().Size()
	if canvasSize.Width > 0 && width > canvasSize.Width-40 {
		width = canvasSize.Width - 40
	}
	if canvasSize.Height > 0 && height > canvasSize.Height-40 {
		height = canvasSize.Height - 40
	}
	return fyne.NewSize(width, height)
}

// Copy puts text on the clipboard and reports the outcome instead of assuming
// one: the browser is entitled to refuse a clipboard write, and the text stays
// selectable in the dialog for exactly that case.
func Copy(notify env.Notifier, text string) {
	browser.CopyText(text, func(ok bool) {
		if ok {
			notify.OK("Copied to clipboard")
			return
		}
		notify.Warn("Could not copy - select the text and press Ctrl+C")
	})
}
