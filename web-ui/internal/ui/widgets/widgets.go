// Package widgets holds the small building blocks every part of the page is
// assembled from: the button with the right cursor, the text sizes, badges,
// panels and rules. Nothing in here knows about servers or clients.
package widgets

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"amneziawg-web-ui/web-ui/internal/ui/style"
)

// Button is widget.Button with the hand cursor a browser user expects over
// anything clickable. Fyne's button deliberately reports the default arrow -
// correct for a desktop window, wrong for a page - and the wasm driver maps
// whatever a widget asks for onto the CSS cursor, so this is all it takes.
type Button struct {
	widget.Button
}

// NewButton is the constructor every button in this UI goes through, so none
// of them can be built without the cursor.
func NewButton(label string, icon fyne.Resource, tapped func()) *Button {
	b := &Button{}
	b.ExtendBaseWidget(b)
	b.Text = label
	b.Icon = icon
	b.OnTapped = tapped
	return b
}

func (b *Button) Cursor() desktop.Cursor {
	if b.Disabled() {
		return desktop.DefaultCursor
	}
	return desktop.PointerCursor
}

// NewEntry is the single-line entry this page uses everywhere a value is
// short enough to fit its field.
//
// A stock widget.NewEntry() keeps an internal container.Scroll around its
// text, which makes it the innermost fyne.Scrollable under the pointer: the
// driver hands it every wheel event and it silently drops the ones it cannot
// use, so the page stops scrolling wherever the cursor happens to rest on an
// input. Turning that inner scroller off (which needs both fields - see
// entryRenderer.Layout in fyne) leaves no Scrollable in the way and the wheel
// reaches the page scroll again.
//
// The cost is that such an entry reports its whole text as its minimum width,
// so this is deliberately not used for the I1-I5 fields, whose values are
// megabyte-scale blobs.
func NewEntry() *widget.Entry {
	e := &widget.Entry{Wrapping: fyne.TextWrapOff, Scroll: fyne.ScrollNone}
	e.ExtendBaseWidget(e)
	return e
}

// EntryWithText is a prefilled single-line entry.
func EntryWithText(text string) *widget.Entry {
	entry := NewEntry()
	entry.SetText(text)
	return entry
}

// EntryWithPlaceholder is an empty single-line entry with a hint.
func EntryWithPlaceholder(placeholder string) *widget.Entry {
	entry := NewEntry()
	entry.SetPlaceHolder(placeholder)
	return entry
}

// NumberEntry is a prefilled field that still shows its accepted range once
// the value is cleared, which is exactly when the range is wanted.
func NumberEntry(text, limits string) *widget.Entry {
	entry := NewEntry()
	entry.SetPlaceHolder(limits)
	entry.SetText(text)
	return entry
}

// Labeled stacks a small caption above a field, for the dense parameter grids
// where a full widget.Form row would waste horizontal space.
func Labeled(caption string, field fyne.CanvasObject) fyne.CanvasObject {
	label := canvas.NewText(caption, style.Muted)
	label.TextSize = 11
	return container.NewVBox(label, field)
}

// SectionTitle is the bold heading a block of a form or a dialog opens with.
func SectionTitle(text string) fyne.CanvasObject {
	title := canvas.NewText(text, style.Text)
	title.TextSize = 14
	title.TextStyle = fyne.TextStyle{Bold: true}
	return title
}

// SmallText is the 12pt canvas text used for captions and counters.
func SmallText(text string, c color.Color) *canvas.Text {
	t := canvas.NewText(text, c)
	t.TextSize = 12
	return t
}

// SmallIcon is a 14px glyph in the muted colour, sized to sit next to
// SmallText on the same line.
func SmallIcon(icon fyne.Resource) fyne.CanvasObject {
	glyph := widget.NewIcon(theme.NewColoredResource(icon, theme.ColorNamePlaceHolder))
	return container.NewGridWrap(fyne.NewSize(14, 14), glyph)
}

// Badge is a small rounded tag in the given colour.
func Badge(text string, c color.NRGBA) fyne.CanvasObject {
	bg := canvas.NewRectangle(style.WithAlpha(c, 0x2e))
	bg.CornerRadius = 9
	bg.StrokeColor = style.WithAlpha(c, 0x99)
	bg.StrokeWidth = 1

	label := canvas.NewText(text, c)
	label.TextSize = 11
	label.TextStyle = fyne.TextStyle{Bold: true}

	padded := container.New(&insetLayout{h: 8, v: 3}, label)
	return container.NewStack(bg, padded)
}

// Card wraps content in the rounded surface panel used throughout the page.
func Card(content fyne.CanvasObject) fyne.CanvasObject {
	bg := canvas.NewRectangle(style.Surface)
	bg.CornerRadius = 10
	bg.StrokeColor = style.Border
	bg.StrokeWidth = 1
	return container.NewStack(bg, container.NewPadded(content))
}

// Separator is the thin horizontal rule between blocks.
func Separator() fyne.CanvasObject {
	line := canvas.NewRectangle(style.Border)
	line.SetMinSize(fyne.NewSize(0, 1))
	return line
}

// VerticalRule is the thin divider used inside single-line header rows.
func VerticalRule() fyne.CanvasObject {
	line := canvas.NewRectangle(style.Border)
	line.SetMinSize(fyne.NewSize(1, 16))
	return container.NewCenter(line)
}

// MutedNote is an italic, word-wrapped explanation under a field or a grid.
func MutedNote(text string) fyne.CanvasObject {
	note := widget.NewLabel(text)
	note.Wrapping = fyne.TextWrapWord
	note.TextStyle = fyne.TextStyle{Italic: true}
	return note
}

// InfoGrid lays key/value pairs out in two columns, values in monospace.
func InfoGrid(rows [][2]string) fyne.CanvasObject {
	grid := container.NewGridWithColumns(2)
	for _, row := range rows {
		key := SmallText(row[0], style.Muted)
		value := SmallText(row[1], style.Text)
		value.TextStyle = fyne.TextStyle{Monospace: true}
		grid.Add(container.NewHBox(key, value))
	}
	return grid
}

// MonospaceView is a scrollable text box for configs, plus the setter that
// replaces its contents. Entry has no read-only mode that still allows
// selecting and copying, so edits are reverted - which means the widget needs
// to know which text is the legitimate one at any moment, and callers must go
// through the setter rather than SetText.
func MonospaceView(text string) (*widget.Entry, func(string)) {
	view := widget.NewMultiLineEntry()
	view.TextStyle = fyne.TextStyle{Monospace: true}
	view.Wrapping = fyne.TextWrapBreak

	shown := text
	set := func(next string) {
		shown = next
		view.SetText(next)
	}

	view.OnChanged = func(typed string) {
		if typed != shown {
			view.SetText(shown)
		}
	}

	set(text)
	return view, set
}

// Truncate cuts a value down to limit bytes with an ellipsis, for the places
// where a megabyte-scale blob has to fit on one line.
func Truncate(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit] + "…"
}

// Uptime renders a duration in seconds the way an operator reads it: the
// two largest units that matter - "3d 4h", "4h 12m", "12m" - and "< 1m"
// while it is still counting its first minute.
func Uptime(seconds float64) string {
	total := int(seconds)
	days, hours, minutes := total/86400, total%86400/3600, total%3600/60
	switch {
	case days > 0:
		return lang.L("{{.Count}}d", map[string]any{"Count": days}) + " " +
			lang.L("{{.Count}}h", map[string]any{"Count": hours})
	case hours > 0:
		return lang.L("{{.Count}}h", map[string]any{"Count": hours}) + " " +
			lang.L("{{.Count}}m", map[string]any{"Count": minutes})
	case minutes > 0:
		return lang.L("{{.Count}}m", map[string]any{"Count": minutes})
	default:
		return "< " + lang.L("{{.Count}}m", map[string]any{"Count": 1})
	}
}

// insetLayout pads its children by an explicit number of pixels, which the
// theme padding cannot express per-side (used by the small badges).
type insetLayout struct {
	h, v float32
}

func (i *insetLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	for _, o := range objects {
		o.Move(fyne.NewPos(i.h, i.v))
		o.Resize(size.Subtract(fyne.NewSize(i.h*2, i.v*2)))
	}
}

func (i *insetLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	var min fyne.Size
	for _, o := range objects {
		min = min.Max(o.MinSize())
	}
	return min.Add(fyne.NewSize(i.h*2, i.v*2))
}
