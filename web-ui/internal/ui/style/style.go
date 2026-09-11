// Package style is the single dark palette the page is drawn in, and the
// fyne theme that applies it to the stock widgets. Every component colours
// its own canvas objects from the exported values, so a shade is changed in
// one place.
package style

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// The UI ships with a single dark palette, so every colour below is returned
// regardless of the variant the toolkit asks for.
var (
	Background = color.NRGBA{R: 0x0f, G: 0x13, B: 0x1a, A: 0xff}
	Surface    = color.NRGBA{R: 0x17, G: 0x1d, B: 0x27, A: 0xff}
	SurfaceAlt = color.NRGBA{R: 0x1e, G: 0x26, B: 0x33, A: 0xff}
	Border     = color.NRGBA{R: 0x2a, G: 0x34, B: 0x44, A: 0xff}
	Text       = color.NRGBA{R: 0xe6, G: 0xeb, B: 0xf2, A: 0xff}
	Muted      = color.NRGBA{R: 0x8b, G: 0x97, B: 0xa8, A: 0xff}
	Primary    = color.NRGBA{R: 0x3b, G: 0x82, B: 0xf6, A: 0xff}
	Success    = color.NRGBA{R: 0x22, G: 0xc5, B: 0x5e, A: 0xff}
	Warning    = color.NRGBA{R: 0xf5, G: 0x9e, B: 0x0b, A: 0xff}
	Error      = color.NRGBA{R: 0xef, G: 0x44, B: 0x44, A: 0xff}
	Accent     = color.NRGBA{R: 0xa7, G: 0x8b, B: 0xfa, A: 0xff}

	hover       = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0x12}
	pressed     = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0x1e}
	selection   = color.NRGBA{R: 0x3b, G: 0x82, B: 0xf6, A: 0x55}
	scrollBar   = color.NRGBA{R: 0x3a, G: 0x46, B: 0x5a, A: 0xcc}
	shadow      = color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 0x66}
	disabled    = color.NRGBA{R: 0x5c, G: 0x67, B: 0x77, A: 0xff}
	disabledBtn = color.NRGBA{R: 0x1a, G: 0x21, B: 0x2c, A: 0xff}
)

// WithAlpha returns the colour with its opacity replaced.
func WithAlpha(c color.NRGBA, alpha uint8) color.NRGBA {
	c.A = alpha
	return c
}

// darkTheme wraps the default theme and forces the dark variant everywhere,
// so the app looks the same no matter what the browser reports for
// prefers-color-scheme.
type darkTheme struct {
	fyne.Theme
}

// NewTheme returns the page's theme.
func NewTheme() fyne.Theme {
	return &darkTheme{Theme: theme.DefaultTheme()}
}

func (t *darkTheme) Color(name fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return Background
	case theme.ColorNameButton:
		return SurfaceAlt
	case theme.ColorNameDisabledButton:
		return disabledBtn
	case theme.ColorNameDisabled:
		return disabled
	case theme.ColorNameForeground, theme.ColorNameForegroundOnPrimary,
		theme.ColorNameForegroundOnError, theme.ColorNameForegroundOnSuccess,
		theme.ColorNameForegroundOnWarning:
		return Text
	case theme.ColorNameHeaderBackground:
		return Surface
	case theme.ColorNameHover:
		return hover
	case theme.ColorNameHyperlink:
		return Primary
	case theme.ColorNameInputBackground:
		return SurfaceAlt
	case theme.ColorNameInputBorder:
		return Border
	case theme.ColorNameMenuBackground, theme.ColorNameOverlayBackground:
		return Surface
	case theme.ColorNamePlaceHolder:
		return Muted
	case theme.ColorNamePressed:
		return pressed
	case theme.ColorNamePrimary:
		return Primary
	case theme.ColorNameScrollBar:
		return scrollBar
	case theme.ColorNameScrollBarBackground:
		return Background
	case theme.ColorNameSelection:
		return selection
	case theme.ColorNameSeparator:
		return Border
	case theme.ColorNameShadow:
		return shadow
	case theme.ColorNameSuccess:
		return Success
	case theme.ColorNameWarning:
		return Warning
	case theme.ColorNameError:
		return Error
	}

	return t.Theme.Color(name, theme.VariantDark)
}

func (t *darkTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNameInputRadius, theme.SizeNameSelectionRadius:
		return 6
	}

	return t.Theme.Size(name)
}
