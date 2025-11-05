//go:build !distro_binary

package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/AvengeMedia/danklinux/internal/greeter"
)

func installGreeter() error {
	fmt.Println("=== DMS Greeter Installation ===")

	logFunc := func(msg string) {
		fmt.Println(msg)
	}

	// Step 1: Ensure greetd is installed
	if err := greeter.EnsureGreetdInstalled(logFunc, ""); err != nil {
		return err
	}

	// Step 2: Detect DMS path
	fmt.Println("\nDetecting DMS installation...")
	dmsPath, err := greeter.DetectDMSPath()
	if err != nil {
		return err
	}
	fmt.Printf("✓ Found DMS at: %s\n", dmsPath)

	// Step 3: Detect compositors
	fmt.Println("\nDetecting installed compositors...")
	compositors := greeter.DetectCompositors()
	if len(compositors) == 0 {
		return fmt.Errorf("no supported compositors found (niri or Hyprland required)")
	}

	var selectedCompositor string
	if len(compositors) == 1 {
		selectedCompositor = compositors[0]
		fmt.Printf("✓ Found compositor: %s\n", selectedCompositor)
	} else {
		var err error
		selectedCompositor, err = greeter.PromptCompositorChoice(compositors)
		if err != nil {
			return err
		}
		fmt.Printf("✓ Selected compositor: %s\n", selectedCompositor)
	}

	// Step 4: Setup dms-greeter group and permissions
	fmt.Println("\nSetting up dms-greeter group and permissions...")
	if err := greeter.SetupDMSGroup(logFunc, ""); err != nil {
		return err
	}

	// Step 5: Copy greeter files
	fmt.Println("\nCopying greeter files...")
	if err := greeter.CopyGreeterFiles(dmsPath, selectedCompositor, logFunc, ""); err != nil {
		return err
	}

	// Step 6: Configure greetd
	fmt.Println("\nConfiguring greetd...")
	if err := greeter.ConfigureGreetd(dmsPath, selectedCompositor, logFunc, ""); err != nil {
		return err
	}

	// Step 7: Sync DMS configs
	fmt.Println("\nSynchronizing DMS configurations...")
	if err := greeter.SyncDMSConfigs(dmsPath, logFunc, ""); err != nil {
		return err
	}

	fmt.Println("\n=== Installation Complete ===")
	fmt.Println("\nTo test the greeter, run:")
	fmt.Println("  sudo systemctl start greetd")
	fmt.Println("\nTo enable on boot, run:")
	fmt.Println("  sudo systemctl enable --now greetd")

	return nil
}

func syncGreeter() error {
	fmt.Println("=== DMS Greeter Theme Sync ===")
	fmt.Println()

	logFunc := func(msg string) {
		fmt.Println(msg)
	}

	// Detect DMS path
	fmt.Println("Detecting DMS installation...")
	dmsPath, err := greeter.DetectDMSPath()
	if err != nil {
		return err
	}
	fmt.Printf("✓ Found DMS at: %s\n", dmsPath)

	// Check if greeter cache directory exists
	cacheDir := "/var/cache/dms-greeter"
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		return fmt.Errorf("greeter cache directory not found at %s\nPlease run 'dms greeter install' first", cacheDir)
	}

	// Sync DMS configs
	fmt.Println("\nSynchronizing DMS configurations...")
	if err := greeter.SyncDMSConfigs(dmsPath, logFunc, ""); err != nil {
		return err
	}

	fmt.Println("\n=== Sync Complete ===")
	fmt.Println("\nYour theme, settings, and wallpaper configuration have been synced with the greeter.")
	fmt.Println("The changes will be visible on the next login screen.")

	return nil
}

func checkGreeterStatus() error {
	fmt.Println("=== DMS Greeter Status ===")
	fmt.Println()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}

	currentUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}

	// Check if user is in greeter group
	fmt.Println("Group Membership:")
	groupsCmd := exec.Command("groups", currentUser.Username)
	groupsOutput, err := groupsCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check groups: %w", err)
	}

	inGreeterGroup := strings.Contains(string(groupsOutput), "greeter")
	if inGreeterGroup {
		fmt.Println("  ✓ User is in greeter group")
	} else {
		fmt.Println("  ✗ User is NOT in greeter group")
		fmt.Println("    Run 'dms greeter install' to add user to greeter group")
	}

	// Check if greeter cache directory exists
	cacheDir := "/var/cache/dms-greeter"
	fmt.Println("\nGreeter Cache Directory:")
	if stat, err := os.Stat(cacheDir); err == nil && stat.IsDir() {
		fmt.Printf("  ✓ %s exists\n", cacheDir)
	} else {
		fmt.Printf("  ✗ %s not found\n", cacheDir)
		fmt.Println("    Run 'dms greeter install' to create cache directory")
		return nil
	}

	// Check symlinks
	fmt.Println("\nConfiguration Symlinks:")
	symlinks := []struct {
		source string
		target string
		desc   string
	}{
		{
			source: filepath.Join(homeDir, ".config", "DankMaterialShell", "settings.json"),
			target: filepath.Join(cacheDir, "settings.json"),
			desc:   "Settings",
		},
		{
			source: filepath.Join(homeDir, ".local", "state", "DankMaterialShell", "session.json"),
			target: filepath.Join(cacheDir, "session.json"),
			desc:   "Session state",
		},
		{
			source: filepath.Join(homeDir, ".cache", "quickshell", "dankshell", "dms-colors.json"),
			target: filepath.Join(cacheDir, "colors.json"),
			desc:   "Color theme",
		},
	}

	allGood := true
	for _, link := range symlinks {
		// Check if target symlink exists
		targetInfo, err := os.Lstat(link.target)
		if err != nil {
			fmt.Printf("  ✗ %s: symlink not found at %s\n", link.desc, link.target)
			allGood = false
			continue
		}

		// Check if it's a symlink
		if targetInfo.Mode()&os.ModeSymlink == 0 {
			fmt.Printf("  ✗ %s: %s is not a symlink\n", link.desc, link.target)
			allGood = false
			continue
		}

		// Check if symlink points to correct source
		linkDest, err := os.Readlink(link.target)
		if err != nil {
			fmt.Printf("  ✗ %s: failed to read symlink\n", link.desc)
			allGood = false
			continue
		}

		if linkDest != link.source {
			fmt.Printf("  ✗ %s: symlink points to wrong location\n", link.desc)
			fmt.Printf("    Expected: %s\n", link.source)
			fmt.Printf("    Got: %s\n", linkDest)
			allGood = false
			continue
		}

		// Check if source file exists
		if _, err := os.Stat(link.source); os.IsNotExist(err) {
			fmt.Printf("  ⚠ %s: symlink OK, but source file doesn't exist yet\n", link.desc)
			fmt.Printf("    Will be created when you run DMS\n")
			continue
		}

		fmt.Printf("  ✓ %s: synced correctly\n", link.desc)
	}

	fmt.Println()
	if allGood && inGreeterGroup {
		fmt.Println("✓ All checks passed! Greeter is properly configured.")
		fmt.Println("\nTo re-sync after theme changes, run:")
		fmt.Println("  dms greeter sync")
	} else if !allGood {
		fmt.Println("⚠ Some issues detected. Run 'dms greeter sync' to fix symlinks.")
	}

	return nil
}
