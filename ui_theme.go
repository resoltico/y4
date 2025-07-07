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
		return color.RGBA{R: 255, G: 255, B: 255, A: 255}

	case theme.ColorNameHeaderBackground:
		if variant == theme.VariantDark {
			return color.RGBA{R: 45, G: 45, B: 45, A: 255}
		}
		return color.RGBA{R: 250, G: 249, B: 245, A: 255}

	case theme.ColorNameForeground:
		if variant == theme.VariantDark {
			return color.RGBA{R: 255, G: 255, B: 255, A: 255}
		}
		return color.RGBA{R: 0, G: 0, B: 0, A: 255}

	case theme.ColorNameButton:
		if variant == theme.VariantDark {
			return color.RGBA{R: 60, G: 60, B: 60, A: 255}
		}
		return color.RGBA{R: 240, G: 240, B: 240, A: 255}

	case theme.ColorNameDisabledButton:
		if variant == theme.VariantDark {
			return color.RGBA{R: 40, G: 40, B: 40, A: 255}
		}
		return color.RGBA{R: 220, G: 220, B: 220, A: 255}

	case theme.ColorNameDisabled:
		if variant == theme.VariantDark {
			return color.RGBA{R: 100, G: 100, B: 100, A: 255}
		}
		return color.RGBA{R: 150, G: 150, B: 150, A: 255}

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

	case theme.ColorNamePressed:
		if variant == theme.VariantDark {
			return color.RGBA{R: 255, G: 255, B: 255, A: 50}
		}
		return color.RGBA{R: 0, G: 0, B: 0, A: 50}

	case theme.ColorNameFocus:
		return t.Color(theme.ColorNamePrimary, variant)

	case theme.ColorNameInputBackground:
		if variant == theme.VariantDark {
			return color.RGBA{R: 50, G: 50, B: 50, A: 255}
		}
		return color.RGBA{R: 248, G: 248, B: 248, A: 255}

	case theme.ColorNameInputBorder:
		if variant == theme.VariantDark {
			return color.RGBA{R: 80, G: 80, B: 80, A: 255}
		}
		return color.RGBA{R: 200, G: 200, B: 200, A: 255}

	case theme.ColorNameMenuBackground:
		if variant == theme.VariantDark {
			return color.RGBA{R: 40, G: 40, B: 40, A: 255}
		}
		return color.RGBA{R: 252, G: 252, B: 252, A: 255}

	case theme.ColorNameOverlayBackground:
		if variant == theme.VariantDark {
			return color.RGBA{R: 35, G: 35, B: 35, A: 255}
		}
		return color.RGBA{R: 250, G: 250, B: 250, A: 255}

	case theme.ColorNamePlaceHolder:
		if variant == theme.VariantDark {
			return color.RGBA{R: 120, G: 120, B: 120, A: 255}
		}
		return color.RGBA{R: 140, G: 140, B: 140, A: 255}

	case theme.ColorNameSelection:
		if variant == theme.VariantDark {
			return color.RGBA{R: 70, G: 120, B: 200, A: 150}
		}
		return color.RGBA{R: 100, G: 150, B: 255, A: 150}

	case theme.ColorNameSeparator:
		if variant == theme.VariantDark {
			return color.RGBA{R: 70, G: 70, B: 70, A: 255}
		}
		return color.RGBA{R: 220, G: 220, B: 220, A: 255}

	case theme.ColorNameScrollBar:
		if variant == theme.VariantDark {
			return color.RGBA{R: 80, G: 80, B: 80, A: 255}
		}
		return color.RGBA{R: 180, G: 180, B: 180, A: 255}

	case theme.ColorNameShadow:
		if variant == theme.VariantDark {
			return color.RGBA{R: 0, G: 0, B: 0, A: 80}
		}
		return color.RGBA{R: 0, G: 0, B: 0, A: 40}

	case theme.ColorNameError:
		if variant == theme.VariantDark {
			return color.RGBA{R: 255, G: 100, B: 100, A: 255}
		}
		return color.RGBA{R: 244, G: 67, B: 54, A: 255}

	case theme.ColorNameSuccess:
		if variant == theme.VariantDark {
			return color.RGBA{R: 100, G: 255, B: 100, A: 255}
		}
		return color.RGBA{R: 76, G: 175, B: 80, A: 255}

	case theme.ColorNameWarning:
		if variant == theme.VariantDark {
			return color.RGBA{R: 255, G: 200, B: 100, A: 255}
		}
		return color.RGBA{R: 255, G: 152, B: 0, A: 255}

	case theme.ColorNameHyperlink:
		if variant == theme.VariantDark {
			return color.RGBA{R: 150, G: 200, B: 255, A: 255}
		}
		return color.RGBA{R: 33, G: 150, B: 243, A: 255}

	case theme.ColorNameForegroundOnPrimary:
		return color.RGBA{R: 255, G: 255, B: 255, A: 255}

	case theme.ColorNameForegroundOnError:
		return color.RGBA{R: 255, G: 255, B: 255, A: 255}

	case theme.ColorNameForegroundOnSuccess:
		return color.RGBA{R: 255, G: 255, B: 255, A: 255}

	case theme.ColorNameForegroundOnWarning:
		return color.RGBA{R: 0, G: 0, B: 0, A: 255}

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
