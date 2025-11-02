package dank16

import (
	"encoding/json"
	"fmt"
	"math"

	"github.com/lucasb-eyer/go-colorful"
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
	if c <= 0.04045 {
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

func getLstar(hex string) float64 {
	rgb := HexToRGB(hex)
	col := colorful.Color{R: rgb.R, G: rgb.G, B: rgb.B}
	L, _, _ := col.Lab()
	return L * 100.0 // go-colorful uses 0-1, we need 0-100 for DPS
}

// Lab to hex, clamping if needed
func labToHex(L, a, b float64) string {
	c := colorful.Lab(L/100.0, a, b) // back to 0-1 for go-colorful
	r, g, b2 := c.Clamped().RGB255()
	return fmt.Sprintf("#%02x%02x%02x", r, g, b2)
}

// Adjust brightness while keeping the same hue
func retoneToL(hex string, Ltarget float64) string {
	rgb := HexToRGB(hex)
	col := colorful.Color{R: rgb.R, G: rgb.G, B: rgb.B}
	L, a, b := col.Lab()
	L100 := L * 100.0

	scale := 1.0
	if L100 != 0 {
		scale = Ltarget / L100
	}

	a2, b2 := a*scale, b*scale

	// Don't let it get too saturated
	maxChroma := 0.4
	if math.Hypot(a2, b2) > maxChroma {
		k := maxChroma / math.Hypot(a2, b2)
		a2 *= k
		b2 *= k
	}

	return labToHex(Ltarget, a2, b2)
}

func DeltaPhiStar(hexFg, hexBg string, negativePolarity bool) float64 {
	Lf := getLstar(hexFg)
	Lb := getLstar(hexBg)

	phi := 1.618
	inv := 0.618
	lc := math.Pow(math.Abs(math.Pow(Lb, phi)-math.Pow(Lf, phi)), inv)*1.414 - 40

	if negativePolarity {
		lc += 5
	}

	return lc
}

func DeltaPhiStarContrast(hexFg, hexBg string, isLightMode bool) float64 {
	negativePolarity := !isLightMode
	return DeltaPhiStar(hexFg, hexBg, negativePolarity)
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

func EnsureContrastDPS(hexColor, hexBg string, minLc float64, isLightMode bool) string {
	currentLc := DeltaPhiStarContrast(hexColor, hexBg, isLightMode)
	if currentLc >= minLc {
		return hexColor
	}

	rgb := HexToRGB(hexColor)
	hsv := RGBToHSV(rgb)

	for step := 1; step < 50; step++ {
		delta := float64(step) * 0.015

		if isLightMode {
			newV := math.Max(0, hsv.V-delta)
			candidate := RGBToHex(HSVToRGB(HSV{H: hsv.H, S: hsv.S, V: newV}))
			if DeltaPhiStarContrast(candidate, hexBg, isLightMode) >= minLc {
				return candidate
			}

			newV = math.Min(1, hsv.V+delta)
			candidate = RGBToHex(HSVToRGB(HSV{H: hsv.H, S: hsv.S, V: newV}))
			if DeltaPhiStarContrast(candidate, hexBg, isLightMode) >= minLc {
				return candidate
			}
		} else {
			newV := math.Min(1, hsv.V+delta)
			candidate := RGBToHex(HSVToRGB(HSV{H: hsv.H, S: hsv.S, V: newV}))
			if DeltaPhiStarContrast(candidate, hexBg, isLightMode) >= minLc {
				return candidate
			}

			newV = math.Max(0, hsv.V-delta)
			candidate = RGBToHex(HSVToRGB(HSV{H: hsv.H, S: hsv.S, V: newV}))
			if DeltaPhiStarContrast(candidate, hexBg, isLightMode) >= minLc {
				return candidate
			}
		}
	}

	return hexColor
}

// Nudge L* until contrast is good enough. Keeps hue intact unlike HSV fiddling.
func EnsureContrastDPSLstar(hexColor, hexBg string, minLc float64, isLightMode bool) string {
	current := DeltaPhiStarContrast(hexColor, hexBg, isLightMode)
	if current >= minLc {
		return hexColor
	}

	fg := HexToRGB(hexColor)
	cf := colorful.Color{R: fg.R, G: fg.G, B: fg.B}
	Lf, af, bf := cf.Lab()

	dir := 1.0
	if isLightMode {
		dir = -1.0 // light mode = darker text
	}

	step := 0.5
	for i := 0; i < 120; i++ {
		Lf = math.Max(0, math.Min(100, Lf+dir*step))
		cand := labToHex(Lf, af, bf)
		if DeltaPhiStarContrast(cand, hexBg, isLightMode) >= minLc {
			return cand
		}
	}

	return hexColor
}

type PaletteOptions struct {
	IsLight      bool
	HonorPrimary string
	Background   string
	UseDPS       bool
}

func ensureContrastAuto(hexColor, hexBg string, target float64, opts PaletteOptions) string {
	if opts.UseDPS {
		return EnsureContrastDPSLstar(hexColor, hexBg, target, opts.IsLight)
	}
	return EnsureContrast(hexColor, hexBg, target, opts.IsLight)
}

func GeneratePalette(baseColor string, opts PaletteOptions) []string {
	rgb := HexToRGB(baseColor)
	hsv := RGBToHSV(rgb)

	palette := make([]string, 0, 16)

	// Contrast targets: DPS is tuned to keep colors vibrant
	var normalTextTarget, secondaryTarget float64
	if opts.UseDPS {
		normalTextTarget = 40.0
		secondaryTarget = 35.0
	} else {
		normalTextTarget = 4.5 // WCAG AA
		secondaryTarget = 3.0
	}

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
		palette = append(palette, ensureContrastAuto(redColor, bgColor, normalTextTarget, opts))
	} else {
		redColor = RGBToHex(HSVToRGB(HSV{H: redH, S: 0.6, V: 0.8}))
		palette = append(palette, ensureContrastAuto(redColor, bgColor, normalTextTarget, opts))
	}

	greenH := 0.33
	var greenColor string
	if opts.IsLight {
		greenColor = RGBToHex(HSVToRGB(HSV{H: greenH, S: math.Max(hsv.S*0.9, 0.75), V: hsv.V * 0.6}))
		palette = append(palette, ensureContrastAuto(greenColor, bgColor, normalTextTarget, opts))
	} else {
		greenColor = RGBToHex(HSVToRGB(HSV{H: greenH, S: 0.35, V: 0.85})) // pastel and bright
		palette = append(palette, ensureContrastAuto(greenColor, bgColor, normalTextTarget, opts))
	}

	yellowH := 0.15 // actual yellow, not orange/brown
	var yellowColor string
	if opts.IsLight {
		yellowColor = RGBToHex(HSVToRGB(HSV{H: yellowH, S: 0.65, V: 0.7}))
		palette = append(palette, ensureContrastAuto(yellowColor, bgColor, normalTextTarget, opts))
	} else {
		yellowColor = RGBToHex(HSVToRGB(HSV{H: yellowH, S: 0.30, V: 0.88})) // pastel so it doesn't look like piss
		palette = append(palette, ensureContrastAuto(yellowColor, bgColor, normalTextTarget, opts))
	}

	var blueColor string
	if opts.IsLight {
		blueColor = RGBToHex(HSVToRGB(HSV{H: hsv.H, S: math.Max(hsv.S*0.9, 0.7), V: hsv.V * 1.1}))
		palette = append(palette, ensureContrastAuto(blueColor, bgColor, normalTextTarget, opts))
	} else {
		blueColor = RGBToHex(HSVToRGB(HSV{H: hsv.H, S: math.Max(hsv.S*0.8, 0.6), V: math.Min(hsv.V*1.6, 1.0)}))
		palette = append(palette, ensureContrastAuto(blueColor, bgColor, normalTextTarget, opts))
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
			palette = append(palette, ensureContrastAuto(magColor, bgColor, normalTextTarget, opts))
		} else {
			magColor = RGBToHex(HSVToRGB(HSV{H: hh.H, S: hh.S * 0.8, V: hh.V * 0.75}))
			palette = append(palette, ensureContrastAuto(magColor, bgColor, normalTextTarget, opts))
		}
	} else if opts.IsLight {
		magColor = RGBToHex(HSVToRGB(HSV{H: magH, S: math.Max(hsv.S*0.75, 0.6), V: hsv.V * 0.9}))
		palette = append(palette, ensureContrastAuto(magColor, bgColor, normalTextTarget, opts))
	} else {
		magColor = RGBToHex(HSVToRGB(HSV{H: magH, S: math.Max(hsv.S*0.7, 0.6), V: hsv.V * 0.85}))
		palette = append(palette, ensureContrastAuto(magColor, bgColor, normalTextTarget, opts))
	}

	cyanH := hsv.H + 0.08
	if cyanH > 1.0 {
		cyanH -= 1.0
	}
	if opts.HonorPrimary != "" {
		palette = append(palette, ensureContrastAuto(opts.HonorPrimary, bgColor, normalTextTarget, opts))
	} else if opts.IsLight {
		cyanColor := RGBToHex(HSVToRGB(HSV{H: cyanH, S: math.Max(hsv.S*0.8, 0.65), V: hsv.V * 1.05}))
		palette = append(palette, ensureContrastAuto(cyanColor, bgColor, normalTextTarget, opts))
	} else {
		cyanColor := RGBToHex(HSVToRGB(HSV{H: cyanH, S: math.Max(hsv.S*0.6, 0.5), V: math.Min(hsv.V*1.25, 0.85)}))
		palette = append(palette, ensureContrastAuto(cyanColor, bgColor, normalTextTarget, opts))
	}

	if opts.IsLight {
		palette = append(palette, "#1a1a1a")
		palette = append(palette, "#2e2e2e")
	} else {
		palette = append(palette, "#abb2bf")
		palette = append(palette, "#5c6370")
	}

	if opts.IsLight {
		brightRed := RGBToHex(HSVToRGB(HSV{H: redH, S: 0.6, V: 0.9}))
		palette = append(palette, ensureContrastAuto(brightRed, bgColor, secondaryTarget, opts))
		brightGreen := RGBToHex(HSVToRGB(HSV{H: greenH, S: math.Max(hsv.S*0.8, 0.7), V: hsv.V * 0.65}))
		palette = append(palette, ensureContrastAuto(brightGreen, bgColor, secondaryTarget, opts))
		// Bright yellow with lower saturation to stay yellow (not orange)
		brightYellow := RGBToHex(HSVToRGB(HSV{H: yellowH, S: 0.55, V: 0.85}))
		palette = append(palette, ensureContrastAuto(brightYellow, bgColor, secondaryTarget, opts))
		if opts.HonorPrimary != "" {
			hr := HexToRGB(opts.HonorPrimary)
			hh := RGBToHSV(hr)
			brightBlue := RGBToHex(HSVToRGB(HSV{H: hh.H, S: math.Min(hh.S*1.1, 1.0), V: math.Min(hh.V*1.2, 1.0)}))
			palette = append(palette, ensureContrastAuto(brightBlue, bgColor, secondaryTarget, opts))
		} else {
			brightBlue := RGBToHex(HSVToRGB(HSV{H: hsv.H, S: math.Max(hsv.S*0.8, 0.7), V: math.Min(hsv.V*1.3, 1.0)}))
			palette = append(palette, ensureContrastAuto(brightBlue, bgColor, secondaryTarget, opts))
		}
		brightMag := RGBToHex(HSVToRGB(HSV{H: magH, S: math.Max(hsv.S*0.9, 0.75), V: math.Min(hsv.V*1.25, 1.0)}))
		palette = append(palette, ensureContrastAuto(brightMag, bgColor, secondaryTarget, opts))
		brightCyan := RGBToHex(HSVToRGB(HSV{H: cyanH, S: math.Max(hsv.S*0.75, 0.65), V: math.Min(hsv.V*1.25, 1.0)}))
		palette = append(palette, ensureContrastAuto(brightCyan, bgColor, secondaryTarget, opts))
	} else {
		brightRed := RGBToHex(HSVToRGB(HSV{H: redH, S: 0.45, V: math.Min(1.0, 0.9)}))
		palette = append(palette, ensureContrastAuto(brightRed, bgColor, secondaryTarget, opts))
		brightGreen := RGBToHex(HSVToRGB(HSV{H: greenH, S: 0.30, V: 0.90})) // pastel bright green
		palette = append(palette, ensureContrastAuto(brightGreen, bgColor, secondaryTarget, opts))
		brightYellow := RGBToHex(HSVToRGB(HSV{H: yellowH, S: 0.25, V: 0.94}))
		palette = append(palette, ensureContrastAuto(brightYellow, bgColor, secondaryTarget, opts))
		if opts.HonorPrimary != "" {
			// Make it way brighter for type names in dark mode
			brightBlue := retoneToL(opts.HonorPrimary, 85.0)
			palette = append(palette, brightBlue)
		} else {
			brightBlue := RGBToHex(HSVToRGB(HSV{H: hsv.H, S: math.Max(hsv.S*0.6, 0.5), V: math.Min(hsv.V*1.5, 0.9)}))
			palette = append(palette, ensureContrastAuto(brightBlue, bgColor, secondaryTarget, opts))
		}
		brightMag := RGBToHex(HSVToRGB(HSV{H: magH, S: math.Max(hsv.S*0.7, 0.6), V: math.Min(hsv.V*1.3, 0.9)}))
		palette = append(palette, ensureContrastAuto(brightMag, bgColor, secondaryTarget, opts))
		brightCyanH := hsv.H + 0.02
		if brightCyanH > 1.0 {
			brightCyanH -= 1.0
		}
		brightCyan := RGBToHex(HSVToRGB(HSV{H: brightCyanH, S: math.Max(hsv.S*0.6, 0.5), V: math.Min(hsv.V*1.2, 0.85)}))
		palette = append(palette, ensureContrastAuto(brightCyan, bgColor, secondaryTarget, opts))
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
			"gitDecoration.modifiedResourceForeground":    colors[6],
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
			"editorGutter.modifiedBackground":             colors[6],
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
			{Scope: []string{"keyword", "storage.type", "storage.modifier"}, Settings: VSCodeTokenSetting{Foreground: colors[5]}},
			{Scope: []string{"variable", "meta.object-literal.key"}, Settings: VSCodeTokenSetting{Foreground: colors[7]}},
			{Scope: []string{"string", "constant.other.symbol"}, Settings: VSCodeTokenSetting{Foreground: colors[2]}},
			{Scope: []string{"constant.numeric", "constant.language", "constant.character"}, Settings: VSCodeTokenSetting{Foreground: colors[3]}},
			{Scope: []string{"entity.name.type", "support.type", "entity.name.class"}, Settings: VSCodeTokenSetting{Foreground: colors[3]}},
			{Scope: []string{"entity.name.function", "support.function"}, Settings: VSCodeTokenSetting{Foreground: colors[4]}},
			{Scope: []string{"support.class", "support.variable", "variable.language"}, Settings: VSCodeTokenSetting{Foreground: colors[6]}},
			{Scope: []string{"invalid"}, Settings: VSCodeTokenSetting{Foreground: colors[9]}},
			{Scope: []string{"invalid.deprecated"}, Settings: VSCodeTokenSetting{Foreground: colors[8]}},
			{Scope: []string{"markup.heading"}, Settings: VSCodeTokenSetting{Foreground: colors[4], FontStyle: "bold"}},
			{Scope: []string{"markup.bold"}, Settings: VSCodeTokenSetting{Foreground: colors[3], FontStyle: "bold"}},
			{Scope: []string{"markup.italic"}, Settings: VSCodeTokenSetting{Foreground: colors[5], FontStyle: "italic"}},
			{Scope: []string{"markup.underline"}, Settings: VSCodeTokenSetting{FontStyle: "underline"}},
			{Scope: []string{"markup.quote"}, Settings: VSCodeTokenSetting{Foreground: colors[6]}},
			{Scope: []string{"markup.list"}, Settings: VSCodeTokenSetting{Foreground: colors[7]}},
			{Scope: []string{"markup.raw", "markup.inline.raw"}, Settings: VSCodeTokenSetting{Foreground: colors[2]}},
		},
		SemanticHighlighting: true,
		SemanticTokenColors: map[string]VSCodeTokenSetting{
			"variable.readonly": {Foreground: colors[3]},
			"property":          {Foreground: colors[7]},
			"function":          {Foreground: colors[4]},
			"method":            {Foreground: colors[4]},
			"type":              {Foreground: colors[3]},
			"class":             {Foreground: colors[3]},
			"enumMember":        {Foreground: colors[3]},
			"string":            {Foreground: colors[2]},
			"number":            {Foreground: colors[3]},
			"comment":           {Foreground: colors[3], FontStyle: "italic"},
			"keyword":           {Foreground: colors[5]},
			"operator":          {Foreground: colors[7]},
			"parameter":         {Foreground: colors[7]},
			"namespace":         {Foreground: colors[6]},
		},
	}

	return theme
}

func GenerateAlacrittyTheme(colors []string) string {
	var result string
	result += "[colors.normal]\n"
	result += fmt.Sprintf("black   = '%s'\n", colors[0])
	result += fmt.Sprintf("red     = '%s'\n", colors[1])
	result += fmt.Sprintf("green   = '%s'\n", colors[2])
	result += fmt.Sprintf("yellow  = '%s'\n", colors[3])
	result += fmt.Sprintf("blue    = '%s'\n", colors[4])
	result += fmt.Sprintf("magenta = '%s'\n", colors[5])
	result += fmt.Sprintf("cyan    = '%s'\n", colors[6])
	result += fmt.Sprintf("white   = '%s'\n", colors[7])
	result += "\n"
	result += "[colors.bright]\n"
	result += fmt.Sprintf("black   = '%s'\n", colors[8])
	result += fmt.Sprintf("red     = '%s'\n", colors[9])
	result += fmt.Sprintf("green   = '%s'\n", colors[10])
	result += fmt.Sprintf("yellow  = '%s'\n", colors[11])
	result += fmt.Sprintf("blue    = '%s'\n", colors[12])
	result += fmt.Sprintf("magenta = '%s'\n", colors[13])
	result += fmt.Sprintf("cyan    = '%s'\n", colors[14])
	result += fmt.Sprintf("white   = '%s'\n", colors[15])
	return result
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

	tokenColors, ok := theme["tokenColors"].([]interface{})
	if ok {
		scopeToColor := map[string]string{
			"comment":                        colors[8],
			"punctuation.definition.comment": colors[8],
			"keyword":                        colors[5],
			"storage.type":                   colors[13], // uint8, etc
			"storage.modifier":               colors[5],
			"variable":                       colors[15],
			"meta.object-literal.key":        colors[15],
			"string":                         colors[3],
			"constant.other.symbol":          colors[3],
			"constant.numeric":               colors[3],
			"constant.language":              colors[11], // true/false/nil
			"constant.character":             colors[3],
			"entity.name.type":               colors[12], // type ABC
			"support.type":                   colors[13],
			"entity.name.class":              colors[12],
			"entity.name.function":           colors[2],
			"support.function":               colors[2],
			"support.class":                  colors[15],
			"support.variable":               colors[15],
			"variable.language":              colors[11], // this/self/super
		}

		for i, tc := range tokenColors {
			if tcMap, ok := tc.(map[string]interface{}); ok {
				if scopes, ok := tcMap["scope"].([]interface{}); ok {
					if settings, ok := tcMap["settings"].(map[string]interface{}); ok {
						for _, scope := range scopes {
							if scopeStr, ok := scope.(string); ok {
								if newColor, exists := scopeToColor[scopeStr]; exists {
									settings["foreground"] = newColor
									tokenColors[i] = tcMap
									break
								}
							}
						}
					}
				}
			}
		}
		theme["tokenColors"] = tokenColors
	}

	if semanticTokenColors, ok := theme["semanticTokenColors"].(map[string]interface{}); ok {
		updates := map[string]string{
			"variable":          colors[15], // white - most common element
			"variable.readonly": colors[11],
			"property":          colors[15], // white
			"function":          colors[2],
			"method":            colors[2],
			"type":              colors[12], // type ABC
			"class":             colors[12],
			"typeParameter":     colors[13],
			"enumMember":        colors[11],
			"string":            colors[3],
			"number":            colors[3],
			"comment":           colors[8],
			"keyword":           colors[5],
			"operator":          colors[15],
			"parameter":         colors[14],
			"namespace":         colors[15], // white - package names stand out
		}

		for key, color := range updates {
			if existing, ok := semanticTokenColors[key].(map[string]interface{}); ok {
				existing["foreground"] = color
			} else {
				semanticTokenColors[key] = map[string]interface{}{
					"foreground": color,
				}
			}
		}
	} else {
		semanticTokenColors := make(map[string]interface{})
		updates := map[string]string{
			"variable":          colors[7], // neutral gray - most common, stay subtle
			"variable.readonly": colors[11],
			"property":          colors[7], // neutral gray
			"function":          colors[2],
			"method":            colors[2],
			"type":              colors[12], // type ABC
			"class":             colors[12],
			"typeParameter":     colors[13],
			"enumMember":        colors[11],
			"string":            colors[3],
			"number":            colors[3],
			"comment":           colors[8],
			"keyword":           colors[5],
			"operator":          colors[15],
			"parameter":         colors[14],
			"namespace":         colors[15], // white - package names stand out
		}

		for key, color := range updates {
			semanticTokenColors[key] = map[string]interface{}{
				"foreground": color,
			}
		}
		theme["semanticTokenColors"] = semanticTokenColors
	}

	return json.MarshalIndent(theme, "", "  ")
}
