package main

import (
	"fmt"
	"time"

	"github.com/AvengeMedia/danklinux/internal/log"
	"github.com/AvengeMedia/danklinux/internal/server/brightness"
	"github.com/spf13/cobra"
)

var brightnessCmd = &cobra.Command{
	Use:   "brightness",
	Short: "Control device brightness",
	Long:  "Control brightness for backlight and LED devices (use --ddc to include DDC/I2C monitors)",
}

var brightnessListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all brightness devices",
	Long:  "List all available brightness devices with their current values",
	Run:   runBrightnessList,
}

var brightnessSetCmd = &cobra.Command{
	Use:   "set <device_id> <percent>",
	Short: "Set brightness for a device",
	Long:  "Set brightness percentage (0-100) for a specific device",
	Args:  cobra.ExactArgs(2),
	Run:   runBrightnessSet,
}

var brightnessGetCmd = &cobra.Command{
	Use:   "get <device_id>",
	Short: "Get brightness for a device",
	Long:  "Get current brightness percentage for a specific device",
	Args:  cobra.ExactArgs(1),
	Run:   runBrightnessGet,
}

func init() {
	brightnessListCmd.Flags().Bool("ddc", false, "Include DDC/I2C monitors (slower)")
	brightnessSetCmd.Flags().Bool("ddc", false, "Include DDC/I2C monitors (slower)")
	brightnessGetCmd.Flags().Bool("ddc", false, "Include DDC/I2C monitors (slower)")

	brightnessCmd.SetHelpTemplate(`{{.Long}}

Usage:
  {{.UseLine}}{{if .HasAvailableSubCommands}}

Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .Name .NamePadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`)

	brightnessListCmd.SetHelpTemplate(`{{.Long}}

Usage:
  {{.UseLine}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}
`)

	brightnessSetCmd.SetHelpTemplate(`{{.Long}}

Usage:
  {{.UseLine}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}
`)

	brightnessGetCmd.SetHelpTemplate(`{{.Long}}

Usage:
  {{.UseLine}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}
`)

	brightnessCmd.AddCommand(brightnessListCmd, brightnessSetCmd, brightnessGetCmd)
}

func runBrightnessList(cmd *cobra.Command, args []string) {
	includeDDC, _ := cmd.Flags().GetBool("ddc")

	allDevices := []brightness.Device{}

	sysfs, err := brightness.NewSysfsBackend()
	if err != nil {
		log.Debugf("Failed to initialize sysfs backend: %v", err)
	} else {
		devices, err := sysfs.GetDevices()
		if err != nil {
			log.Debugf("Failed to get sysfs devices: %v", err)
		} else {
			allDevices = append(allDevices, devices...)
		}
	}

	if includeDDC {
		ddc, err := brightness.NewDDCBackend()
		if err != nil {
			log.Debugf("Failed to initialize DDC backend: %v", err)
		} else {
			time.Sleep(100 * time.Millisecond)
			devices, err := ddc.GetDevices()
			if err != nil {
				log.Debugf("Failed to get DDC devices: %v", err)
			} else {
				allDevices = append(allDevices, devices...)
			}
			ddc.Close()
		}
	}

	if len(allDevices) == 0 {
		fmt.Println("No brightness devices found")
		return
	}

	maxIDLen := len("Device")
	maxNameLen := len("Name")
	for _, dev := range allDevices {
		if len(dev.ID) > maxIDLen {
			maxIDLen = len(dev.ID)
		}
		if len(dev.Name) > maxNameLen {
			maxNameLen = len(dev.Name)
		}
	}

	idPad := maxIDLen + 2
	namePad := maxNameLen + 2

	fmt.Printf("%-*s  %-12s  %-*s  %s\n", idPad, "Device", "Class", namePad, "Name", "Brightness")

	sepLen := idPad + 2 + 12 + 2 + namePad + 2 + 15
	for i := 0; i < sepLen; i++ {
		fmt.Print("─")
	}
	fmt.Println()

	for _, device := range allDevices {
		fmt.Printf("%-*s  %-12s  %-*s  %3d%%\n",
			idPad,
			device.ID,
			device.Class,
			namePad,
			device.Name,
			device.CurrentPercent,
		)
	}
}

func runBrightnessSet(cmd *cobra.Command, args []string) {
	deviceID := args[0]
	var percent int
	if _, err := fmt.Sscanf(args[1], "%d", &percent); err != nil {
		log.Fatalf("Invalid percent value: %s", args[1])
	}

	if percent < 0 || percent > 100 {
		log.Fatalf("Percent must be between 0 and 100")
	}

	includeDDC, _ := cmd.Flags().GetBool("ddc")

	sysfs, err := brightness.NewSysfsBackend()
	if err == nil {
		if err := sysfs.SetBrightness(deviceID, percent); err == nil {
			fmt.Printf("Set %s to %d%%\n", deviceID, percent)
			return
		}
	}

	if includeDDC {
		ddc, err := brightness.NewDDCBackend()
		if err == nil {
			defer ddc.Close()
			time.Sleep(100 * time.Millisecond)
			if err := ddc.SetBrightness(deviceID, percent); err == nil {
				fmt.Printf("Set %s to %d%%\n", deviceID, percent)
				return
			}
		}
	}

	log.Fatalf("Failed to set brightness for device: %s", deviceID)
}

func runBrightnessGet(cmd *cobra.Command, args []string) {
	deviceID := args[0]
	includeDDC, _ := cmd.Flags().GetBool("ddc")

	allDevices := []brightness.Device{}

	sysfs, err := brightness.NewSysfsBackend()
	if err == nil {
		devices, err := sysfs.GetDevices()
		if err == nil {
			allDevices = append(allDevices, devices...)
		}
	}

	if includeDDC {
		ddc, err := brightness.NewDDCBackend()
		if err == nil {
			defer ddc.Close()
			time.Sleep(100 * time.Millisecond)
			devices, err := ddc.GetDevices()
			if err == nil {
				allDevices = append(allDevices, devices...)
			}
		}
	}

	for _, device := range allDevices {
		if device.ID == deviceID {
			fmt.Printf("%s: %d%% (%d/%d)\n",
				device.ID,
				device.CurrentPercent,
				device.Current,
				device.Max,
			)
			return
		}
	}

	log.Fatalf("Device not found: %s", deviceID)
}
