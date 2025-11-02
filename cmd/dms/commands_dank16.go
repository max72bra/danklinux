package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/AvengeMedia/danklinux/internal/dank16"
	"github.com/AvengeMedia/danklinux/internal/log"
	"github.com/spf13/cobra"
)

var dank16Cmd = &cobra.Command{
	Use:   "dank16 <hex_color>",
	Short: "Generate Base16 color palettes",
	Long:  "Generate Base16 color palettes from a base color with support for various output formats",
	Args:  cobra.ExactArgs(1),
	Run:   runDank16,
}

func init() {
	dank16Cmd.Flags().Bool("light", false, "Generate light theme variant")
	dank16Cmd.Flags().Bool("kitty", false, "Output in Kitty terminal format")
	dank16Cmd.Flags().Bool("foot", false, "Output in Foot terminal format")
	dank16Cmd.Flags().Bool("alacritty", false, "Output in Alacritty terminal format")
	dank16Cmd.Flags().Bool("vscode", false, "Output as VSCode theme JSON")
	dank16Cmd.Flags().String("vscode-enrich", "", "Enrich existing VSCode theme file with terminal colors")
	dank16Cmd.Flags().String("honor-primary", "", "Honor primary color for specific palette positions")
	dank16Cmd.Flags().String("background", "", "Custom background color")
	dank16Cmd.Flags().String("contrast", "dps", "Contrast algorithm: dps (Delta Phi Star, default) or wcag")
}

func runDank16(cmd *cobra.Command, args []string) {
	baseColor := args[0]
	if !strings.HasPrefix(baseColor, "#") {
		baseColor = "#" + baseColor
	}

	isLight, _ := cmd.Flags().GetBool("light")
	isKitty, _ := cmd.Flags().GetBool("kitty")
	isFoot, _ := cmd.Flags().GetBool("foot")
	isAlacritty, _ := cmd.Flags().GetBool("alacritty")
	isVSCode, _ := cmd.Flags().GetBool("vscode")
	vscodeEnrich, _ := cmd.Flags().GetString("vscode-enrich")
	honorPrimary, _ := cmd.Flags().GetString("honor-primary")
	background, _ := cmd.Flags().GetString("background")
	contrastAlgo, _ := cmd.Flags().GetString("contrast")

	if honorPrimary != "" && !strings.HasPrefix(honorPrimary, "#") {
		honorPrimary = "#" + honorPrimary
	}

	if background != "" && !strings.HasPrefix(background, "#") {
		background = "#" + background
	}

	contrastAlgo = strings.ToLower(contrastAlgo)
	if contrastAlgo != "dps" && contrastAlgo != "wcag" {
		log.Fatalf("Invalid contrast algorithm: %s (must be 'dps' or 'wcag')", contrastAlgo)
	}

	opts := dank16.PaletteOptions{
		IsLight:      isLight,
		HonorPrimary: honorPrimary,
		Background:   background,
		UseDPS:       contrastAlgo == "dps",
	}

	colors := dank16.GeneratePalette(baseColor, opts)

	if isVSCode {
		theme := dank16.GenerateVSCodeTheme(colors, isLight)
		output, err := json.MarshalIndent(theme, "", "  ")
		if err != nil {
			log.Fatalf("Error generating VSCode theme: %v", err)
		}
		fmt.Println(string(output))
	} else if vscodeEnrich != "" {
		data, err := os.ReadFile(vscodeEnrich)
		if err != nil {
			log.Fatalf("Error reading file: %v", err)
		}

		enriched, err := dank16.EnrichVSCodeTheme(data, colors)
		if err != nil {
			log.Fatalf("Error enriching theme: %v", err)
		}
		fmt.Println(string(enriched))
	} else if isKitty {
		kittyColors := []struct {
			name  string
			color string
		}{
			{"color0", colors[0]},
			{"color1", colors[1]},
			{"color2", colors[2]},
			{"color3", colors[3]},
			{"color4", colors[4]},
			{"color5", colors[5]},
			{"color6", colors[6]},
			{"color7", colors[7]},
			{"color8", colors[8]},
			{"color9", colors[9]},
			{"color10", colors[10]},
			{"color11", colors[11]},
			{"color12", colors[12]},
			{"color13", colors[13]},
			{"color14", colors[14]},
			{"color15", colors[15]},
		}

		for _, kc := range kittyColors {
			fmt.Printf("%s   %s\n", kc.name, kc.color)
		}
	} else if isFoot {
		footColors := []struct {
			name  string
			index int
		}{
			{"regular0", 0},
			{"regular1", 1},
			{"regular2", 2},
			{"regular3", 3},
			{"regular4", 4},
			{"regular5", 5},
			{"regular6", 6},
			{"regular7", 7},
			{"bright0", 8},
			{"bright1", 9},
			{"bright2", 10},
			{"bright3", 11},
			{"bright4", 12},
			{"bright5", 13},
			{"bright6", 14},
			{"bright7", 15},
		}

		for _, fc := range footColors {
			fmt.Printf("%s=%s\n", fc.name, strings.TrimPrefix(colors[fc.index], "#"))
		}
	} else if isAlacritty {
		fmt.Print(dank16.GenerateAlacrittyTheme(colors))
	} else {
		for i, color := range colors {
			fmt.Printf("palette = %d=%s\n", i, color)
		}
	}
}
