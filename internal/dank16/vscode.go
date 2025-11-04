package dank16

import (
	"encoding/json"
	"fmt"
)

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

func updateTokenColor(tc interface{}, scopeToColor map[string]string) {
	tcMap, ok := tc.(map[string]interface{})
	if !ok {
		return
	}

	scopes, ok := tcMap["scope"].([]interface{})
	if !ok {
		return
	}

	settings, ok := tcMap["settings"].(map[string]interface{})
	if !ok {
		return
	}

	for _, scope := range scopes {
		if applyColorToScope(settings, scope, scopeToColor) {
			break
		}
	}
}

func applyColorToScope(settings map[string]interface{}, scope interface{}, scopeToColor map[string]string) bool {
	scopeStr, ok := scope.(string)
	if !ok {
		return false
	}

	newColor, exists := scopeToColor[scopeStr]
	if !exists {
		return false
	}

	settings["foreground"] = newColor
	return true
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

	bg := colors[0]
	isLight := false
	if len(bg) == 7 && bg[0] == '#' {
		r, g, b := 0, 0, 0
		fmt.Sscanf(bg[1:], "%02x%02x%02x", &r, &g, &b)
		luminance := (0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)) / 255.0
		isLight = luminance > 0.5
	}

	if isLight {
		theme["type"] = "light"
	} else {
		theme["type"] = "dark"
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
			updateTokenColor(tc, scopeToColor)
			tokenColors[i] = tc
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
