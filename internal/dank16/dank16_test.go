package dank16

import (
	"encoding/json"
	"math"
	"testing"
)

func TestHexToRGB(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected RGB
	}{
		{
			name:     "black with hash",
			input:    "#000000",
			expected: RGB{R: 0.0, G: 0.0, B: 0.0},
		},
		{
			name:     "white with hash",
			input:    "#ffffff",
			expected: RGB{R: 1.0, G: 1.0, B: 1.0},
		},
		{
			name:     "red without hash",
			input:    "ff0000",
			expected: RGB{R: 1.0, G: 0.0, B: 0.0},
		},
		{
			name:     "purple",
			input:    "#625690",
			expected: RGB{R: 0.3843137254901961, G: 0.33725490196078434, B: 0.5647058823529412},
		},
		{
			name:     "mid gray",
			input:    "#808080",
			expected: RGB{R: 0.5019607843137255, G: 0.5019607843137255, B: 0.5019607843137255},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HexToRGB(tt.input)
			if !floatEqual(result.R, tt.expected.R) || !floatEqual(result.G, tt.expected.G) || !floatEqual(result.B, tt.expected.B) {
				t.Errorf("HexToRGB(%s) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestRGBToHex(t *testing.T) {
	tests := []struct {
		name     string
		input    RGB
		expected string
	}{
		{
			name:     "black",
			input:    RGB{R: 0.0, G: 0.0, B: 0.0},
			expected: "#000000",
		},
		{
			name:     "white",
			input:    RGB{R: 1.0, G: 1.0, B: 1.0},
			expected: "#ffffff",
		},
		{
			name:     "red",
			input:    RGB{R: 1.0, G: 0.0, B: 0.0},
			expected: "#ff0000",
		},
		{
			name:     "clamping above 1.0",
			input:    RGB{R: 1.5, G: 0.5, B: 0.5},
			expected: "#ff7f7f",
		},
		{
			name:     "clamping below 0.0",
			input:    RGB{R: -0.5, G: 0.5, B: 0.5},
			expected: "#007f7f",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RGBToHex(tt.input)
			if result != tt.expected {
				t.Errorf("RGBToHex(%v) = %s, expected %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestRGBToHSV(t *testing.T) {
	tests := []struct {
		name     string
		input    RGB
		expected HSV
	}{
		{
			name:     "black",
			input:    RGB{R: 0.0, G: 0.0, B: 0.0},
			expected: HSV{H: 0.0, S: 0.0, V: 0.0},
		},
		{
			name:     "white",
			input:    RGB{R: 1.0, G: 1.0, B: 1.0},
			expected: HSV{H: 0.0, S: 0.0, V: 1.0},
		},
		{
			name:     "red",
			input:    RGB{R: 1.0, G: 0.0, B: 0.0},
			expected: HSV{H: 0.0, S: 1.0, V: 1.0},
		},
		{
			name:     "green",
			input:    RGB{R: 0.0, G: 1.0, B: 0.0},
			expected: HSV{H: 0.3333333333333333, S: 1.0, V: 1.0},
		},
		{
			name:     "blue",
			input:    RGB{R: 0.0, G: 0.0, B: 1.0},
			expected: HSV{H: 0.6666666666666666, S: 1.0, V: 1.0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RGBToHSV(tt.input)
			if !floatEqual(result.H, tt.expected.H) || !floatEqual(result.S, tt.expected.S) || !floatEqual(result.V, tt.expected.V) {
				t.Errorf("RGBToHSV(%v) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestHSVToRGB(t *testing.T) {
	tests := []struct {
		name     string
		input    HSV
		expected RGB
	}{
		{
			name:     "black",
			input:    HSV{H: 0.0, S: 0.0, V: 0.0},
			expected: RGB{R: 0.0, G: 0.0, B: 0.0},
		},
		{
			name:     "white",
			input:    HSV{H: 0.0, S: 0.0, V: 1.0},
			expected: RGB{R: 1.0, G: 1.0, B: 1.0},
		},
		{
			name:     "red",
			input:    HSV{H: 0.0, S: 1.0, V: 1.0},
			expected: RGB{R: 1.0, G: 0.0, B: 0.0},
		},
		{
			name:     "green",
			input:    HSV{H: 0.3333333333333333, S: 1.0, V: 1.0},
			expected: RGB{R: 0.0, G: 1.0, B: 0.0},
		},
		{
			name:     "blue",
			input:    HSV{H: 0.6666666666666666, S: 1.0, V: 1.0},
			expected: RGB{R: 0.0, G: 0.0, B: 1.0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HSVToRGB(tt.input)
			if !floatEqual(result.R, tt.expected.R) || !floatEqual(result.G, tt.expected.G) || !floatEqual(result.B, tt.expected.B) {
				t.Errorf("HSVToRGB(%v) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestLuminance(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected float64
	}{
		{
			name:     "black",
			input:    "#000000",
			expected: 0.0,
		},
		{
			name:     "white",
			input:    "#ffffff",
			expected: 1.0,
		},
		{
			name:     "red",
			input:    "#ff0000",
			expected: 0.2126,
		},
		{
			name:     "green",
			input:    "#00ff00",
			expected: 0.7152,
		},
		{
			name:     "blue",
			input:    "#0000ff",
			expected: 0.0722,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Luminance(tt.input)
			if !floatEqual(result, tt.expected) {
				t.Errorf("Luminance(%s) = %f, expected %f", tt.input, result, tt.expected)
			}
		})
	}
}

func TestContrastRatio(t *testing.T) {
	tests := []struct {
		name     string
		fg       string
		bg       string
		expected float64
	}{
		{
			name:     "black on white",
			fg:       "#000000",
			bg:       "#ffffff",
			expected: 21.0,
		},
		{
			name:     "white on black",
			fg:       "#ffffff",
			bg:       "#000000",
			expected: 21.0,
		},
		{
			name:     "same color",
			fg:       "#808080",
			bg:       "#808080",
			expected: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ContrastRatio(tt.fg, tt.bg)
			if !floatEqual(result, tt.expected) {
				t.Errorf("ContrastRatio(%s, %s) = %f, expected %f", tt.fg, tt.bg, result, tt.expected)
			}
		})
	}
}

func TestEnsureContrast(t *testing.T) {
	tests := []struct {
		name        string
		color       string
		bg          string
		minRatio    float64
		isLightMode bool
	}{
		{
			name:        "already sufficient contrast dark mode",
			color:       "#ffffff",
			bg:          "#000000",
			minRatio:    4.5,
			isLightMode: false,
		},
		{
			name:        "already sufficient contrast light mode",
			color:       "#000000",
			bg:          "#ffffff",
			minRatio:    4.5,
			isLightMode: true,
		},
		{
			name:        "needs adjustment dark mode",
			color:       "#404040",
			bg:          "#1a1a1a",
			minRatio:    4.5,
			isLightMode: false,
		},
		{
			name:        "needs adjustment light mode",
			color:       "#c0c0c0",
			bg:          "#f8f8f8",
			minRatio:    4.5,
			isLightMode: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EnsureContrast(tt.color, tt.bg, tt.minRatio, tt.isLightMode)
			actualRatio := ContrastRatio(result, tt.bg)
			if actualRatio < tt.minRatio {
				t.Errorf("EnsureContrast(%s, %s, %f, %t) = %s with ratio %f, expected ratio >= %f",
					tt.color, tt.bg, tt.minRatio, tt.isLightMode, result, actualRatio, tt.minRatio)
			}
		})
	}
}

func TestGeneratePalette(t *testing.T) {
	tests := []struct {
		name string
		base string
		opts PaletteOptions
	}{
		{
			name: "dark theme default",
			base: "#625690",
			opts: PaletteOptions{IsLight: false},
		},
		{
			name: "light theme default",
			base: "#625690",
			opts: PaletteOptions{IsLight: true},
		},
		{
			name: "dark theme with honor primary",
			base: "#625690",
			opts: PaletteOptions{
				IsLight:      false,
				HonorPrimary: "#ff6600",
			},
		},
		{
			name: "light theme with custom background",
			base: "#625690",
			opts: PaletteOptions{
				IsLight:    true,
				Background: "#fafafa",
			},
		},
		{
			name: "dark theme with all options",
			base: "#625690",
			opts: PaletteOptions{
				IsLight:      false,
				HonorPrimary: "#ff6600",
				Background:   "#0a0a0a",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GeneratePalette(tt.base, tt.opts)

			if len(result) != 16 {
				t.Errorf("GeneratePalette returned %d colors, expected 16", len(result))
			}

			for i, color := range result {
				if len(color) != 7 || color[0] != '#' {
					t.Errorf("Color at index %d (%s) is not a valid hex color", i, color)
				}
			}

			if tt.opts.Background != "" && result[0] != tt.opts.Background {
				t.Errorf("Background color = %s, expected %s", result[0], tt.opts.Background)
			} else if !tt.opts.IsLight && tt.opts.Background == "" && result[0] != "#1a1a1a" {
				t.Errorf("Dark mode background = %s, expected #1a1a1a", result[0])
			} else if tt.opts.IsLight && tt.opts.Background == "" && result[0] != "#f8f8f8" {
				t.Errorf("Light mode background = %s, expected #f8f8f8", result[0])
			}

			if tt.opts.IsLight && result[15] != "#1a1a1a" {
				t.Errorf("Light mode foreground = %s, expected #1a1a1a", result[15])
			} else if !tt.opts.IsLight && result[15] != "#ffffff" {
				t.Errorf("Dark mode foreground = %s, expected #ffffff", result[15])
			}
		})
	}
}

func TestGenerateVSCodeTheme(t *testing.T) {
	colors := GeneratePalette("#625690", PaletteOptions{IsLight: false})

	tests := []struct {
		name    string
		colors  []string
		isLight bool
	}{
		{
			name:    "dark theme",
			colors:  colors,
			isLight: false,
		},
		{
			name:    "light theme",
			colors:  GeneratePalette("#625690", PaletteOptions{IsLight: true}),
			isLight: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateVSCodeTheme(tt.colors, tt.isLight)

			if result.Schema != "vscode://schemas/color-theme" {
				t.Errorf("Schema = %s, expected vscode://schemas/color-theme", result.Schema)
			}

			if result.Name != "Dynamic Base16 DankShell" {
				t.Errorf("Name = %s, expected Dynamic Base16 DankShell", result.Name)
			}

			expectedType := "dark"
			if tt.isLight {
				expectedType = "light"
			}
			if result.Type != expectedType {
				t.Errorf("Type = %s, expected %s", result.Type, expectedType)
			}

			if !result.SemanticHighlighting {
				t.Error("SemanticHighlighting should be true")
			}

			if len(result.Colors) == 0 {
				t.Error("Colors map is empty")
			}

			if len(result.TokenColors) == 0 {
				t.Error("TokenColors array is empty")
			}

			if len(result.SemanticTokenColors) == 0 {
				t.Error("SemanticTokenColors map is empty")
			}

			requiredColors := []string{
				"editor.background",
				"editor.foreground",
				"terminal.ansiBlack",
				"terminal.ansiRed",
				"terminal.ansiBrightWhite",
			}
			for _, key := range requiredColors {
				if _, ok := result.Colors[key]; !ok {
					t.Errorf("Missing required color key: %s", key)
				}
			}
		})
	}
}

func TestEnrichVSCodeTheme(t *testing.T) {
	colors := GeneratePalette("#625690", PaletteOptions{IsLight: false})

	baseTheme := map[string]interface{}{
		"name": "Test Theme",
		"type": "dark",
		"colors": map[string]interface{}{
			"editor.background": "#000000",
		},
	}

	themeJSON, err := json.Marshal(baseTheme)
	if err != nil {
		t.Fatalf("Failed to marshal base theme: %v", err)
	}

	result, err := EnrichVSCodeTheme(themeJSON, colors)
	if err != nil {
		t.Fatalf("EnrichVSCodeTheme failed: %v", err)
	}

	var enriched map[string]interface{}
	if err := json.Unmarshal(result, &enriched); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	colorsMap, ok := enriched["colors"].(map[string]interface{})
	if !ok {
		t.Fatal("colors is not a map")
	}

	terminalColors := []string{
		"terminal.ansiBlack",
		"terminal.ansiRed",
		"terminal.ansiGreen",
		"terminal.ansiYellow",
		"terminal.ansiBlue",
		"terminal.ansiMagenta",
		"terminal.ansiCyan",
		"terminal.ansiWhite",
		"terminal.ansiBrightBlack",
		"terminal.ansiBrightRed",
		"terminal.ansiBrightGreen",
		"terminal.ansiBrightYellow",
		"terminal.ansiBrightBlue",
		"terminal.ansiBrightMagenta",
		"terminal.ansiBrightCyan",
		"terminal.ansiBrightWhite",
	}

	for i, key := range terminalColors {
		if val, ok := colorsMap[key]; !ok {
			t.Errorf("Missing terminal color: %s", key)
		} else if val != colors[i] {
			t.Errorf("%s = %s, expected %s", key, val, colors[i])
		}
	}

	if colorsMap["editor.background"] != "#000000" {
		t.Error("Original theme colors should be preserved")
	}
}

func TestEnrichVSCodeThemeInvalidJSON(t *testing.T) {
	colors := GeneratePalette("#625690", PaletteOptions{IsLight: false})
	invalidJSON := []byte("{invalid json")

	_, err := EnrichVSCodeTheme(invalidJSON, colors)
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestRoundTripConversion(t *testing.T) {
	testColors := []string{"#000000", "#ffffff", "#ff0000", "#00ff00", "#0000ff", "#625690", "#808080"}

	for _, hex := range testColors {
		t.Run(hex, func(t *testing.T) {
			rgb := HexToRGB(hex)
			result := RGBToHex(rgb)
			if result != hex {
				t.Errorf("Round trip %s -> RGB -> %s failed", hex, result)
			}
		})
	}
}

func TestRGBHSVRoundTrip(t *testing.T) {
	testCases := []RGB{
		{R: 0.0, G: 0.0, B: 0.0},
		{R: 1.0, G: 1.0, B: 1.0},
		{R: 1.0, G: 0.0, B: 0.0},
		{R: 0.0, G: 1.0, B: 0.0},
		{R: 0.0, G: 0.0, B: 1.0},
		{R: 0.5, G: 0.5, B: 0.5},
		{R: 0.3843137254901961, G: 0.33725490196078434, B: 0.5647058823529412},
	}

	for _, rgb := range testCases {
		t.Run("", func(t *testing.T) {
			hsv := RGBToHSV(rgb)
			result := HSVToRGB(hsv)
			if !floatEqual(result.R, rgb.R) || !floatEqual(result.G, rgb.G) || !floatEqual(result.B, rgb.B) {
				t.Errorf("Round trip RGB->HSV->RGB failed: %v -> %v -> %v", rgb, hsv, result)
			}
		})
	}
}

func floatEqual(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}
