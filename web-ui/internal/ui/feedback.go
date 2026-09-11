package ui

import (
	"fmt"
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"

	"amneziawg-web-ui/web-ui/internal/ui/dialogs"
	"amneziawg-web-ui/web-ui/internal/ui/style"
)

// feedback is the page's env.Notifier: outcomes go to the footer as a
// transient note, failures open an error dialog. Every method is safe from
// any goroutine.
type feedback struct {
	win   fyne.Window
	toast *canvas.Text
	seq   int
}

// notify shows a transient message in the footer. Successive messages replace
// each other, and each one clears itself after a few seconds.
func (f *feedback) notify(message string, c color.Color) {
	fyne.Do(func() {
		f.seq++
		seq := f.seq
		f.toast.Text = message
		f.toast.Color = c
		f.toast.Refresh()

		time.AfterFunc(4*time.Second, func() {
			fyne.Do(func() {
				if f.seq != seq {
					return
				}
				f.toast.Text = ""
				f.toast.Refresh()
			})
		})
	})
}

func (f *feedback) OK(format string, args ...any) {
	f.notify(fmt.Sprintf(format, args...), style.Success)
}

func (f *feedback) Warn(format string, args ...any) {
	f.notify(fmt.Sprintf(format, args...), style.Warning)
}

func (f *feedback) Fail(err error) {
	fyne.Do(func() { dialogs.ShowError(f.win, err) })
}
