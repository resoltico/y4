package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type OtsuTheme struct{}

func NewOtsuTheme() fyne.Theme {
	return &OtsuTheme{}
}

func (t *OtsuTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		if variant == theme.VariantDark {
			return color.RGBA{R: 30, G: 30, B: 30, A: 255}
		}
		return color.RGBA{R: 250, G: 249, B: 245, A: 255}

	case theme.ColorNameButton:
		if variant == theme.VariantDark {
			return color.RGBA{R: 60, G: 60, B: 60, A: 255}
		}
		return color.RGBA{R: 240, G: 240, B: 240, A: 255}

	case theme.ColorNameForeground:
		if variant == theme.VariantDark {
			return color.RGBA{R: 255, G: 255, B: 255, A: 255}
		}
		return color.RGBA{R: 0, G: 0, B: 0, A: 255}

	case theme.ColorNamePrimary:
		if variant == theme.VariantDark {
			return color.RGBA{R: 100, G: 150, B: 255, A: 255}
		}
		return color.RGBA{R: 33, G: 150, B: 243, A: 255}

	case theme.ColorNameHover:
		if variant == theme.VariantDark {
			return color.RGBA{R: 255, G: 255, B: 255, A: 25}
		}
		return color.RGBA{R: 0, G: 0, B: 0, A: 25}

	case theme.ColorNameFocus:
		return t.Color(theme.ColorNamePrimary, variant)

	default:
		return theme.DefaultTheme().Color(name, variant)
	}
}

func (t *OtsuTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (t *OtsuTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t *OtsuTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}
