package dank16

import (
	"encoding/json"
	"fmt"
	"math"
)

type RGB struct {
	R, G, B float64
}

type HSV struct {
	H, S, V float64
}

func HexToRGB(hex string) RGB {
	if hex[0] == '#' {
		hex = hex[1:]
	}
	var r, g, b uint8
	fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b)
	return RGB{
		R: float64(r) / 255.0,
		G: float64(g) / 255.0,
		B: float64(b) / 255.0,
	}
}

func RGBToHex(rgb RGB) string {
	r := math.Max(0, math.Min(1, rgb.R))
	g := math.Max(0, math.Min(1, rgb.G))
	b := math.Max(0, math.Min(1, rgb.B))
	return fmt.Sprintf("#%02x%02x%02x", int(r*255), int(g*255), int(b*255))
}

func RGBToHSV(rgb RGB) HSV {
	max := math.Max(math.Max(rgb.R, rgb.G), rgb.B)
	min := math.Min(math.Min(rgb.R, rgb.G), rgb.B)
	delta := max - min

	var h float64
	if delta == 0 {
		h = 0
	} else if max == rgb.R {
		h = math.Mod((rgb.G-rgb.B)/delta, 6.0) / 6.0
	} else if max == rgb.G {
		h = ((rgb.B-rgb.R)/delta + 2.0) / 6.0
	} else {
		h = ((rgb.R-rgb.G)/delta + 4.0) / 6.0
	}

	if h < 0 {
		h += 1.0
	}

	var s float64
	if max == 0 {
		s = 0
	} else {
		s = delta / max
	}

	return HSV{H: h, S: s, V: max}
}

func HSVToRGB(hsv HSV) RGB {
	h := hsv.H * 6.0
	c := hsv.V * hsv.S
	x := c * (1.0 - math.Abs(math.Mod(h, 2.0)-1.0))
	m := hsv.V - c

	var r, g, b float64
	switch int(h) {
	case 0:
		r, g, b = c, x, 0
	case 1:
		r, g, b = x, c, 0
	case 2:
		r, g, b = 0, c, x
	case 3:
		r, g, b = 0, x, c
	case 4:
		r, g, b = x, 0, c
	case 5:
		r, g, b = c, 0, x
	default:
		r, g, b = c, 0, x
	}

	return RGB{R: r + m, G: g + m, B: b + m}
}

func sRGBToLinear(c float64) float64 {
	if c <= 0.03928 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}

func Luminance(hex string) float64 {
	rgb := HexToRGB(hex)
	return 0.2126*sRGBToLinear(rgb.R) + 0.7152*sRGBToLinear(rgb.G) + 0.0722*sRGBToLinear(rgb.B)
}

func ContrastRatio(hexFg, hexBg string) float64 {
	lumFg := Luminance(hexFg)
	lumBg := Luminance(hexBg)
	lighter := math.Max(lumFg, lumBg)
	darker := math.Min(lumFg, lumBg)
	return (lighter + 0.05) / (darker + 0.05)
}

func EnsureContrast(hexColor, hexBg string, minRatio float64, isLightMode bool) string {
	currentRatio := ContrastRatio(hexColor, hexBg)
	if currentRatio >= minRatio {
		return hexColor
	}

	rgb := HexToRGB(hexColor)
	hsv := RGBToHSV(rgb)

	for step := 1; step < 30; step++ {
		delta := float64(step) * 0.02

		if isLightMode {
			newV := math.Max(0, hsv.V-delta)
			candidate := RGBToHex(HSVToRGB(HSV{H: hsv.H, S: hsv.S, V: newV}))
			if ContrastRatio(candidate, hexBg) >= minRatio {
				return candidate
			}

			newV = math.Min(1, hsv.V+delta)
			candidate = RGBToHex(HSVToRGB(HSV{H: hsv.H, S: hsv.S, V: newV}))
			if ContrastRatio(candidate, hexBg) >= minRatio {
				return candidate
			}
		} else {
			newV := math.Min(1, hsv.V+delta)
			candidate := RGBToHex(HSVToRGB(HSV{H: hsv.H, S: hsv.S, V: newV}))
			if ContrastRatio(candidate, hexBg) >= minRatio {
				return candidate
			}

			newV = math.Max(0, hsv.V-delta)
			candidate = RGBToHex(HSVToRGB(HSV{H: hsv.H, S: hsv.S, V: newV}))
			if ContrastRatio(candidate, hexBg) >= minRatio {
				return candidate
			}
		}
	}

	return hexColor
}

type PaletteOptions struct {
	IsLight      bool
	HonorPrimary string
	Background   string
}

func GeneratePalette(baseColor string, opts PaletteOptions) []string {
	rgb := HexToRGB(baseColor)
	hsv := RGBToHSV(rgb)

	palette := make([]string, 0, 16)

	var bgColor string
	if opts.Background != "" {
		bgColor = opts.Background
	} else if opts.IsLight {
		bgColor = "#f8f8f8"
	} else {
		bgColor = "#1a1a1a"
	}
	palette = append(palette, bgColor)

	redH := 0.0
	var redColor string
	if opts.IsLight {
		redColor = RGBToHex(HSVToRGB(HSV{H: redH, S: 0.75, V: 0.85}))
		palette = append(palette, EnsureContrast(redColor, bgColor, 4.5, opts.IsLight))
	} else {
		redColor = RGBToHex(HSVToRGB(HSV{H: redH, S: 0.6, V: 0.8}))
		palette = append(palette, EnsureContrast(redColor, bgColor, 4.5, opts.IsLight))
	}

	greenH := 0.33
	var greenColor string
	if opts.IsLight {
		greenColor = RGBToHex(HSVToRGB(HSV{H: greenH, S: math.Max(hsv.S*0.9, 0.75), V: hsv.V * 0.6}))
		palette = append(palette, EnsureContrast(greenColor, bgColor, 4.5, opts.IsLight))
	} else {
		greenColor = RGBToHex(HSVToRGB(HSV{H: greenH, S: math.Max(hsv.S*0.65, 0.5), V: hsv.V * 0.9}))
		palette = append(palette, EnsureContrast(greenColor, bgColor, 4.5, opts.IsLight))
	}

	yellowH := 0.08
	var yellowColor string
	if opts.IsLight {
		yellowColor = RGBToHex(HSVToRGB(HSV{H: yellowH, S: math.Max(hsv.S*0.85, 0.7), V: hsv.V * 0.7}))
		palette = append(palette, EnsureContrast(yellowColor, bgColor, 4.5, opts.IsLight))
	} else {
		yellowColor = RGBToHex(HSVToRGB(HSV{H: yellowH, S: math.Max(hsv.S*0.5, 0.45), V: hsv.V * 1.4}))
		palette = append(palette, EnsureContrast(yellowColor, bgColor, 4.5, opts.IsLight))
	}

	var blueColor string
	if opts.IsLight {
		blueColor = RGBToHex(HSVToRGB(HSV{H: hsv.H, S: math.Max(hsv.S*0.9, 0.7), V: hsv.V * 1.1}))
		palette = append(palette, EnsureContrast(blueColor, bgColor, 4.5, opts.IsLight))
	} else {
		blueColor = RGBToHex(HSVToRGB(HSV{H: hsv.H, S: math.Max(hsv.S*0.8, 0.6), V: math.Min(hsv.V*1.6, 1.0)}))
		palette = append(palette, EnsureContrast(blueColor, bgColor, 4.5, opts.IsLight))
	}

	magH := hsv.H - 0.03
	if magH < 0 {
		magH += 1.0
	}
	var magColor string
	if opts.HonorPrimary != "" {
		hr := HexToRGB(opts.HonorPrimary)
		hh := RGBToHSV(hr)
		if opts.IsLight {
			magColor = RGBToHex(HSVToRGB(HSV{H: hh.H, S: math.Max(hh.S*0.9, 0.7), V: hh.V * 0.85}))
			palette = append(palette, EnsureContrast(magColor, bgColor, 4.5, opts.IsLight))
		} else {
			magColor = RGBToHex(HSVToRGB(HSV{H: hh.H, S: hh.S * 0.8, V: hh.V * 0.75}))
			palette = append(palette, EnsureContrast(magColor, bgColor, 4.5, opts.IsLight))
		}
	} else if opts.IsLight {
		magColor = RGBToHex(HSVToRGB(HSV{H: magH, S: math.Max(hsv.S*0.75, 0.6), V: hsv.V * 0.9}))
		palette = append(palette, EnsureContrast(magColor, bgColor, 4.5, opts.IsLight))
	} else {
		magColor = RGBToHex(HSVToRGB(HSV{H: magH, S: math.Max(hsv.S*0.7, 0.6), V: hsv.V * 0.85}))
		palette = append(palette, EnsureContrast(magColor, bgColor, 4.5, opts.IsLight))
	}

	cyanH := hsv.H + 0.08
	if cyanH > 1.0 {
		cyanH -= 1.0
	}
	if opts.HonorPrimary != "" {
		palette = append(palette, EnsureContrast(opts.HonorPrimary, bgColor, 4.5, opts.IsLight))
	} else if opts.IsLight {
		cyanColor := RGBToHex(HSVToRGB(HSV{H: cyanH, S: math.Max(hsv.S*0.8, 0.65), V: hsv.V * 1.05}))
		palette = append(palette, EnsureContrast(cyanColor, bgColor, 4.5, opts.IsLight))
	} else {
		cyanColor := RGBToHex(HSVToRGB(HSV{H: cyanH, S: math.Max(hsv.S*0.6, 0.5), V: math.Min(hsv.V*1.25, 0.85)}))
		palette = append(palette, EnsureContrast(cyanColor, bgColor, 4.5, opts.IsLight))
	}

	if opts.IsLight {
		palette = append(palette, "#2e2e2e")
		palette = append(palette, "#4a4a4a")
	} else {
		palette = append(palette, "#abb2bf")
		palette = append(palette, "#5c6370")
	}

	if opts.IsLight {
		brightRed := RGBToHex(HSVToRGB(HSV{H: redH, S: 0.6, V: 0.9}))
		palette = append(palette, EnsureContrast(brightRed, bgColor, 3.0, opts.IsLight))
		brightGreen := RGBToHex(HSVToRGB(HSV{H: greenH, S: math.Max(hsv.S*0.8, 0.7), V: hsv.V * 0.65}))
		palette = append(palette, EnsureContrast(brightGreen, bgColor, 3.0, opts.IsLight))
		brightYellow := RGBToHex(HSVToRGB(HSV{H: yellowH, S: math.Max(hsv.S*0.75, 0.65), V: hsv.V * 0.75}))
		palette = append(palette, EnsureContrast(brightYellow, bgColor, 3.0, opts.IsLight))
		if opts.HonorPrimary != "" {
			hr := HexToRGB(opts.HonorPrimary)
			hh := RGBToHSV(hr)
			brightBlue := RGBToHex(HSVToRGB(HSV{H: hh.H, S: math.Min(hh.S*1.1, 1.0), V: math.Min(hh.V*1.2, 1.0)}))
			palette = append(palette, EnsureContrast(brightBlue, bgColor, 3.0, opts.IsLight))
		} else {
			brightBlue := RGBToHex(HSVToRGB(HSV{H: hsv.H, S: math.Max(hsv.S*0.8, 0.7), V: math.Min(hsv.V*1.3, 1.0)}))
			palette = append(palette, EnsureContrast(brightBlue, bgColor, 3.0, opts.IsLight))
		}
		brightMag := RGBToHex(HSVToRGB(HSV{H: magH, S: math.Max(hsv.S*0.9, 0.75), V: math.Min(hsv.V*1.25, 1.0)}))
		palette = append(palette, EnsureContrast(brightMag, bgColor, 3.0, opts.IsLight))
		brightCyan := RGBToHex(HSVToRGB(HSV{H: cyanH, S: math.Max(hsv.S*0.75, 0.65), V: math.Min(hsv.V*1.25, 1.0)}))
		palette = append(palette, EnsureContrast(brightCyan, bgColor, 3.0, opts.IsLight))
	} else {
		brightRed := RGBToHex(HSVToRGB(HSV{H: redH, S: 0.45, V: math.Min(1.0, 0.9)}))
		palette = append(palette, EnsureContrast(brightRed, bgColor, 3.0, opts.IsLight))
		brightGreen := RGBToHex(HSVToRGB(HSV{H: greenH, S: math.Max(hsv.S*0.5, 0.4), V: math.Min(hsv.V*1.5, 0.9)}))
		palette = append(palette, EnsureContrast(brightGreen, bgColor, 3.0, opts.IsLight))
		brightYellow := RGBToHex(HSVToRGB(HSV{H: yellowH, S: math.Max(hsv.S*0.4, 0.35), V: math.Min(hsv.V*1.6, 0.95)}))
		palette = append(palette, EnsureContrast(brightYellow, bgColor, 3.0, opts.IsLight))
		if opts.HonorPrimary != "" {
			hr := HexToRGB(opts.HonorPrimary)
			hh := RGBToHSV(hr)
			brightBlue := RGBToHex(HSVToRGB(HSV{H: hh.H, S: math.Min(hh.S*1.2, 1.0), V: math.Min(hh.V*1.1, 1.0)}))
			palette = append(palette, EnsureContrast(brightBlue, bgColor, 3.0, opts.IsLight))
		} else {
			brightBlue := RGBToHex(HSVToRGB(HSV{H: hsv.H, S: math.Max(hsv.S*0.6, 0.5), V: math.Min(hsv.V*1.5, 0.9)}))
			palette = append(palette, EnsureContrast(brightBlue, bgColor, 3.0, opts.IsLight))
		}
		brightMag := RGBToHex(HSVToRGB(HSV{H: magH, S: math.Max(hsv.S*0.7, 0.6), V: math.Min(hsv.V*1.3, 0.9)}))
		palette = append(palette, EnsureContrast(brightMag, bgColor, 3.0, opts.IsLight))
		brightCyanH := hsv.H + 0.02
		if brightCyanH > 1.0 {
			brightCyanH -= 1.0
		}
		brightCyan := RGBToHex(HSVToRGB(HSV{H: brightCyanH, S: math.Max(hsv.S*0.6, 0.5), V: math.Min(hsv.V*1.2, 0.85)}))
		palette = append(palette, EnsureContrast(brightCyan, bgColor, 3.0, opts.IsLight))
	}

	if opts.IsLight {
		palette = append(palette, "#1a1a1a")
	} else {
		palette = append(palette, "#ffffff")
	}

	return palette
}

type VSCodeTheme struct {
	Schema               string                        `json:"$schema"`
	Name                 string                        `json:"name"`
	Type                 string                        `json:"type"`
	Colors               map[string]string             `json:"colors"`
	TokenColors          []VSCodeTokenColor            `json:"tokenColors"`
	SemanticHighlighting bool                          `json:"semanticHighlighting"`
	SemanticTokenColors  map[string]VSCodeTokenSetting `json:"semanticTokenColors"`
}

type VSCodeTokenColor struct {
	Scope    interface{}        `json:"scope"`
	Settings VSCodeTokenSetting `json:"settings"`
}

type VSCodeTokenSetting struct {
	Foreground string `json:"foreground,omitempty"`
	FontStyle  string `json:"fontStyle,omitempty"`
}

func GenerateVSCodeTheme(colors []string, isLight bool) VSCodeTheme {
	themeType := "dark"
	if isLight {
		themeType = "light"
	}

	theme := VSCodeTheme{
		Schema: "vscode://schemas/color-theme",
		Name:   "Dynamic Base16 DankShell",
		Type:   themeType,
		Colors: map[string]string{
			"editor.background":                           colors[0],
			"editor.foreground":                           colors[7],
			"editorLineNumber.foreground":                 colors[3],
			"editorLineNumber.activeForeground":           colors[7],
			"editorCursor.foreground":                     colors[4],
			"editor.selectionBackground":                  colors[2],
			"editor.inactiveSelectionBackground":          colors[1],
			"editor.lineHighlightBackground":              colors[1],
			"editorIndentGuide.background":                colors[1],
			"editorIndentGuide.activeBackground":          colors[3],
			"editorWhitespace.foreground":                 colors[2],
			"editorBracketMatch.background":               colors[2],
			"editorBracketMatch.border":                   colors[4],
			"activityBar.background":                      colors[0],
			"activityBar.foreground":                      colors[7],
			"activityBar.activeBorder":                    colors[4],
			"activityBar.activeBackground":                colors[1],
			"activityBarBadge.background":                 colors[4],
			"activityBarBadge.foreground":                 colors[0],
			"sideBar.background":                          colors[0],
			"sideBar.foreground":                          colors[7],
			"sideBar.border":                              colors[1],
			"sideBarTitle.foreground":                     colors[7],
			"sideBarSectionHeader.background":             colors[1],
			"sideBarSectionHeader.foreground":             colors[7],
			"list.activeSelectionBackground":              colors[2],
			"list.activeSelectionForeground":              colors[7],
			"list.inactiveSelectionBackground":            colors[1],
			"list.hoverBackground":                        colors[1],
			"list.focusBackground":                        colors[2],
			"list.highlightForeground":                    colors[4],
			"statusBar.background":                        colors[0],
			"statusBar.foreground":                        colors[7],
			"statusBar.border":                            colors[1],
			"statusBar.noFolderBackground":                colors[0],
			"statusBar.debuggingBackground":               colors[1],
			"statusBar.debuggingForeground":               colors[0],
			"tab.activeBackground":                        colors[0],
			"tab.inactiveBackground":                      colors[1],
			"tab.activeForeground":                        colors[7],
			"tab.inactiveForeground":                      colors[3],
			"tab.border":                                  colors[0],
			"tab.activeBorder":                            colors[4],
			"tab.unfocusedActiveBorder":                   colors[3],
			"editorGroupHeader.tabsBackground":            colors[1],
			"editorGroupHeader.noTabsBackground":          colors[0],
			"titleBar.activeBackground":                   colors[0],
			"titleBar.activeForeground":                   colors[7],
			"titleBar.inactiveBackground":                 colors[0],
			"titleBar.inactiveForeground":                 colors[3],
			"titleBar.border":                             colors[1],
			"input.background":                            colors[1],
			"input.foreground":                            colors[7],
			"input.border":                                colors[2],
			"input.placeholderForeground":                 colors[3],
			"inputOption.activeBorder":                    colors[4],
			"inputValidation.errorBackground":             colors[1],
			"inputValidation.errorBorder":                 colors[1],
			"dropdown.background":                         colors[1],
			"dropdown.foreground":                         colors[7],
			"dropdown.border":                             colors[2],
			"button.background":                           colors[4],
			"button.foreground":                           colors[0],
			"button.hoverBackground":                      colors[12],
			"focusBorder":                                 colors[4],
			"badge.background":                            colors[6],
			"badge.foreground":                            colors[0],
			"panel.background":                            colors[0],
			"panel.border":                                colors[1],
			"panelTitle.activeBorder":                     colors[4],
			"panelTitle.activeForeground":                 colors[7],
			"panelTitle.inactiveForeground":               colors[3],
			"terminal.background":                         colors[0],
			"terminal.foreground":                         colors[7],
			"terminal.ansiBlack":                          colors[0],
			"terminal.ansiRed":                            colors[1],
			"terminal.ansiGreen":                          colors[2],
			"terminal.ansiYellow":                         colors[3],
			"terminal.ansiBlue":                           colors[4],
			"terminal.ansiMagenta":                        colors[5],
			"terminal.ansiCyan":                           colors[6],
			"terminal.ansiWhite":                          colors[7],
			"terminal.ansiBrightBlack":                    colors[8],
			"terminal.ansiBrightRed":                      colors[9],
			"terminal.ansiBrightGreen":                    colors[10],
			"terminal.ansiBrightYellow":                   colors[11],
			"terminal.ansiBrightBlue":                     colors[12],
			"terminal.ansiBrightMagenta":                  colors[13],
			"terminal.ansiBrightCyan":                     colors[14],
			"terminal.ansiBrightWhite":                    colors[15],
			"gitDecoration.modifiedResourceForeground":    colors[11],
			"gitDecoration.deletedResourceForeground":     colors[1],
			"gitDecoration.untrackedResourceForeground":   colors[10],
			"gitDecoration.ignoredResourceForeground":     colors[3],
			"gitDecoration.conflictingResourceForeground": colors[5],
			"editorWidget.background":                     colors[1],
			"editorWidget.border":                         colors[2],
			"editorSuggestWidget.background":              colors[1],
			"editorSuggestWidget.border":                  colors[2],
			"editorSuggestWidget.selectedBackground":      colors[2],
			"editorSuggestWidget.highlightForeground":     colors[4],
			"peekView.border":                             colors[4],
			"peekViewEditor.background":                   colors[1],
			"peekViewResult.background":                   colors[1],
			"peekViewTitle.background":                    colors[1],
			"notificationCenter.border":                   colors[2],
			"notifications.background":                    colors[1],
			"notifications.border":                        colors[2],
			"breadcrumb.foreground":                       colors[3],
			"breadcrumb.focusForeground":                  colors[7],
			"breadcrumb.activeSelectionForeground":        colors[4],
			"scrollbarSlider.background":                  colors[2] + "40",
			"scrollbarSlider.hoverBackground":             colors[3] + "60",
			"scrollbarSlider.activeBackground":            colors[3] + "80",
			"editorError.foreground":                      colors[1],
			"editorWarning.foreground":                    colors[11],
			"editorInfo.foreground":                       colors[4],
			"editorGutter.addedBackground":                colors[10],
			"editorGutter.modifiedBackground":             colors[11],
			"editorGutter.deletedBackground":              colors[1],
			"diffEditor.insertedTextBackground":           colors[10] + "20",
			"diffEditor.removedTextBackground":            colors[1] + "20",
			"merge.currentHeaderBackground":               colors[4] + "40",
			"merge.incomingHeaderBackground":              colors[10] + "40",
			"menubar.selectionBackground":                 colors[2],
			"menu.background":                             colors[1],
			"menu.foreground":                             colors[7],
			"menu.selectionBackground":                    colors[2],
			"menu.selectionForeground":                    colors[7],
			"debugToolBar.background":                     colors[1],
			"debugExceptionWidget.background":             colors[1],
			"debugExceptionWidget.border":                 colors[1],
		},
		TokenColors: []VSCodeTokenColor{
			{Scope: []string{"comment", "punctuation.definition.comment"}, Settings: VSCodeTokenSetting{Foreground: colors[3], FontStyle: "italic"}},
			{Scope: []string{"keyword", "storage.type", "storage.modifier"}, Settings: VSCodeTokenSetting{Foreground: colors[13]}},
			{Scope: []string{"variable", "meta.object-literal.key"}, Settings: VSCodeTokenSetting{Foreground: colors[1]}},
			{Scope: []string{"string", "constant.other.symbol"}, Settings: VSCodeTokenSetting{Foreground: colors[10]}},
			{Scope: []string{"constant.numeric", "constant.language", "constant.character"}, Settings: VSCodeTokenSetting{Foreground: colors[9]}},
			{Scope: []string{"entity.name.type", "support.type", "entity.name.class"}, Settings: VSCodeTokenSetting{Foreground: colors[11]}},
			{Scope: []string{"entity.name.function", "support.function"}, Settings: VSCodeTokenSetting{Foreground: colors[12]}},
			{Scope: []string{"support.class", "support.variable", "variable.language"}, Settings: VSCodeTokenSetting{Foreground: colors[14]}},
			{Scope: []string{"invalid"}, Settings: VSCodeTokenSetting{Foreground: colors[1]}},
			{Scope: []string{"invalid.deprecated"}, Settings: VSCodeTokenSetting{Foreground: colors[5]}},
			{Scope: []string{"markup.heading"}, Settings: VSCodeTokenSetting{Foreground: colors[12], FontStyle: "bold"}},
			{Scope: []string{"markup.bold"}, Settings: VSCodeTokenSetting{Foreground: colors[11], FontStyle: "bold"}},
			{Scope: []string{"markup.italic"}, Settings: VSCodeTokenSetting{Foreground: colors[13], FontStyle: "italic"}},
			{Scope: []string{"markup.underline"}, Settings: VSCodeTokenSetting{FontStyle: "underline"}},
			{Scope: []string{"markup.quote"}, Settings: VSCodeTokenSetting{Foreground: colors[14]}},
			{Scope: []string{"markup.list"}, Settings: VSCodeTokenSetting{Foreground: colors[1]}},
			{Scope: []string{"markup.raw", "markup.inline.raw"}, Settings: VSCodeTokenSetting{Foreground: colors[10]}},
		},
		SemanticHighlighting: true,
		SemanticTokenColors: map[string]VSCodeTokenSetting{
			"variable.readonly": {Foreground: colors[11]},
			"property":          {Foreground: colors[1]},
			"function":          {Foreground: colors[12]},
			"method":            {Foreground: colors[12]},
			"type":              {Foreground: colors[11]},
			"class":             {Foreground: colors[11]},
			"enumMember":        {Foreground: colors[9]},
			"string":            {Foreground: colors[10]},
			"number":            {Foreground: colors[9]},
			"comment":           {Foreground: colors[3], FontStyle: "italic"},
			"keyword":           {Foreground: colors[13]},
			"operator":          {Foreground: colors[7]},
			"parameter":         {Foreground: colors[1]},
			"namespace":         {Foreground: colors[14]},
		},
	}

	return theme
}

func EnrichVSCodeTheme(themeData []byte, colors []string) ([]byte, error) {
	var theme map[string]interface{}
	if err := json.Unmarshal(themeData, &theme); err != nil {
		return nil, err
	}

	colorsMap, ok := theme["colors"].(map[string]interface{})
	if !ok {
		colorsMap = make(map[string]interface{})
		theme["colors"] = colorsMap
	}

	colorsMap["terminal.ansiBlack"] = colors[0]
	colorsMap["terminal.ansiRed"] = colors[1]
	colorsMap["terminal.ansiGreen"] = colors[2]
	colorsMap["terminal.ansiYellow"] = colors[3]
	colorsMap["terminal.ansiBlue"] = colors[4]
	colorsMap["terminal.ansiMagenta"] = colors[5]
	colorsMap["terminal.ansiCyan"] = colors[6]
	colorsMap["terminal.ansiWhite"] = colors[7]
	colorsMap["terminal.ansiBrightBlack"] = colors[8]
	colorsMap["terminal.ansiBrightRed"] = colors[9]
	colorsMap["terminal.ansiBrightGreen"] = colors[10]
	colorsMap["terminal.ansiBrightYellow"] = colors[11]
	colorsMap["terminal.ansiBrightBlue"] = colors[12]
	colorsMap["terminal.ansiBrightMagenta"] = colors[13]
	colorsMap["terminal.ansiBrightCyan"] = colors[14]
	colorsMap["terminal.ansiBrightWhite"] = colors[15]

	return json.MarshalIndent(theme, "", "  ")
}
