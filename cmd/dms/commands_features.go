//go:build !distro_binary

package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/AvengeMedia/danklinux/internal/distros"
	"github.com/AvengeMedia/danklinux/internal/errdefs"
	"github.com/AvengeMedia/danklinux/internal/log"
	"github.com/AvengeMedia/danklinux/internal/version"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update DankMaterialShell to the latest version",
	Long:  "Update DankMaterialShell to the latest version using the appropriate package manager for your distribution",
	Run: func(cmd *cobra.Command, args []string) {
		runUpdate()
	},
}

var updateCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Check if updates are available for DankMaterialShell",
	Long:  "Check for available updates without performing the actual update",
	Run: func(cmd *cobra.Command, args []string) {
		runUpdateCheck()
	},
}

var greeterCmd = &cobra.Command{
	Use:   "greeter",
	Short: "Manage DMS greeter installation",
	Long:  "Manage DMS greeter (greetd) installation and configuration",
}

var greeterInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install and configure DMS greeter",
	Long:  "Install greetd and configure it to use DMS as the greeter interface",
	Run: func(cmd *cobra.Command, args []string) {
		if err := installGreeter(); err != nil {
			log.Fatalf("Error installing greeter: %v", err)
		}
	},
}

func runUpdateCheck() {
	fmt.Println("Checking for DankMaterialShell updates...")
	fmt.Println()

	versionInfo, err := version.GetDMSVersionInfo()
	if err != nil {
		log.Fatalf("Error checking for updates: %v", err)
	}

	fmt.Printf("Current version: %s\n", versionInfo.Current)
	fmt.Printf("Latest version:  %s\n", versionInfo.Latest)
	fmt.Println()

	if versionInfo.HasUpdate {
		fmt.Println("✓ Update available!")
		fmt.Println()
		fmt.Println("Run 'dms update' to install the latest version.")
		os.Exit(0)
	} else {
		fmt.Println("✓ You are running the latest version.")
		os.Exit(0)
	}
}

func runUpdate() {
	osInfo, err := distros.GetOSInfo()
	if err != nil {
		log.Fatalf("Error detecting OS: %v", err)
	}

	config, exists := distros.Registry[osInfo.Distribution.ID]
	if !exists {
		log.Fatalf("Unsupported distribution: %s", osInfo.Distribution.ID)
	}

	var updateErr error
	switch config.Family {
	case distros.FamilyArch:
		updateErr = updateArchLinux()
	case distros.FamilyNix:
		updateErr = updateNixOS()
	case distros.FamilySUSE:
		updateErr = updateOtherDistros()
	default:
		updateErr = updateOtherDistros()
	}

	if updateErr != nil {
		if errors.Is(updateErr, errdefs.ErrUpdateCancelled) {
			log.Info("Update cancelled.")
			return
		}
		if errors.Is(updateErr, errdefs.ErrNoUpdateNeeded) {
			return
		}
		log.Fatalf("Error updating DMS: %v", updateErr)
	}

	log.Info("Update complete! Restarting DMS...")
	restartShell()
}

func updateArchLinux() error {
	homeDir, err := os.UserHomeDir()
	if err == nil {
		dmsPath := filepath.Join(homeDir, ".config", "quickshell", "dms")
		if _, err := os.Stat(dmsPath); err == nil {
			return updateOtherDistros()
		}
	}

	var packageName string
	if isPackageInstalled("dms-shell-bin") {
		packageName = "dms-shell-bin"
	} else if isPackageInstalled("dms-shell-git") {
		packageName = "dms-shell-git"
	} else {
		fmt.Println("Info: Neither dms-shell-bin nor dms-shell-git package found.")
		fmt.Println("Info: Falling back to git-based update method...")
		return updateOtherDistros()
	}

	var helper string
	var updateCmd *exec.Cmd

	if commandExists("yay") {
		helper = "yay"
		updateCmd = exec.Command("yay", "-S", packageName)
	} else if commandExists("paru") {
		helper = "paru"
		updateCmd = exec.Command("paru", "-S", packageName)
	} else {
		fmt.Println("Error: Neither yay nor paru found - please install an AUR helper")
		fmt.Println("Info: Falling back to git-based update method...")
		return updateOtherDistros()
	}

	fmt.Printf("This will update DankMaterialShell using %s.\n", helper)
	if !confirmUpdate() {
		return errdefs.ErrUpdateCancelled
	}

	fmt.Printf("\nRunning: %s -S %s\n", helper, packageName)
	updateCmd.Stdout = os.Stdout
	updateCmd.Stderr = os.Stderr
	err = updateCmd.Run()
	if err != nil {
		fmt.Printf("Error: Failed to update using %s: %v\n", helper, err)
	}

	fmt.Println("dms successfully updated")
	return nil
}

func updateNixOS() error {
	fmt.Println("This will update DankMaterialShell using nix profile.")
	if !confirmUpdate() {
		return errdefs.ErrUpdateCancelled
	}

	fmt.Println("\nRunning: nix profile upgrade github:AvengeMedia/DankMaterialShell")
	updateCmd := exec.Command("nix", "profile", "upgrade", "github:AvengeMedia/DankMaterialShell")
	updateCmd.Stdout = os.Stdout
	updateCmd.Stderr = os.Stderr
	err := updateCmd.Run()
	if err != nil {
		fmt.Printf("Error: Failed to update using nix profile: %v\n", err)
		fmt.Println("Falling back to git-based update method...")
		return updateOtherDistros()
	}

	fmt.Println("dms successfully updated")
	return nil
}

func updateOtherDistros() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}

	dmsPath := filepath.Join(homeDir, ".config", "quickshell", "dms")

	if _, err := os.Stat(dmsPath); os.IsNotExist(err) {
		return fmt.Errorf("DMS configuration directory not found at %s", dmsPath)
	}

	fmt.Printf("Found DMS configuration at %s\n", dmsPath)

	versionInfo, err := version.GetDMSVersionInfo()
	if err == nil && !versionInfo.HasUpdate {
		fmt.Println()
		fmt.Printf("Current version: %s\n", versionInfo.Current)
		fmt.Printf("Latest version:  %s\n", versionInfo.Latest)
		fmt.Println()
		fmt.Println("✓ You are already running the latest version.")
		return errdefs.ErrNoUpdateNeeded
	}

	fmt.Println("\nThis will update:")
	fmt.Println("  1. The dms binary from GitHub releases")
	fmt.Println("  2. DankMaterialShell configuration using git")
	if !confirmUpdate() {
		return errdefs.ErrUpdateCancelled
	}

	fmt.Println("\n=== Updating dms binary ===")
	if err := updateDMSBinary(); err != nil {
		fmt.Printf("Warning: Failed to update dms binary: %v\n", err)
		fmt.Println("Continuing with shell configuration update...")
	} else {
		fmt.Println("dms binary successfully updated")
	}

	fmt.Println("\n=== Updating DMS shell configuration ===")

	if err := os.Chdir(dmsPath); err != nil {
		return fmt.Errorf("failed to change to DMS directory: %w", err)
	}

	statusCmd := exec.Command("git", "status", "--porcelain")
	statusOutput, _ := statusCmd.Output()
	hasLocalChanges := len(strings.TrimSpace(string(statusOutput))) > 0

	currentRefCmd := exec.Command("git", "symbolic-ref", "-q", "HEAD")
	currentRefOutput, _ := currentRefCmd.Output()
	onBranch := len(currentRefOutput) > 0

	var currentTag string
	var currentBranch string

	if !onBranch {
		tagCmd := exec.Command("git", "describe", "--exact-match", "--tags", "HEAD")
		if tagOutput, err := tagCmd.Output(); err == nil {
			currentTag = strings.TrimSpace(string(tagOutput))
		}
	} else {
		branchCmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
		if branchOutput, err := branchCmd.Output(); err == nil {
			currentBranch = strings.TrimSpace(string(branchOutput))
		}
	}

	fmt.Println("Fetching latest changes...")
	fetchCmd := exec.Command("git", "fetch", "origin", "--tags", "--force")
	fetchCmd.Stdout = os.Stdout
	fetchCmd.Stderr = os.Stderr
	if err := fetchCmd.Run(); err != nil {
		return fmt.Errorf("failed to fetch changes: %w", err)
	}

	if currentTag != "" {
		latestTagCmd := exec.Command("git", "tag", "-l", "v0.1.*", "--sort=-version:refname")
		latestTagOutput, err := latestTagCmd.Output()
		if err != nil {
			return fmt.Errorf("failed to get latest tag: %w", err)
		}

		tags := strings.Split(strings.TrimSpace(string(latestTagOutput)), "\n")
		if len(tags) == 0 || tags[0] == "" {
			return fmt.Errorf("no v0.1.* tags found")
		}
		latestTag := tags[0]

		if latestTag == currentTag {
			fmt.Printf("Already on latest tag: %s\n", currentTag)
			return nil
		}

		fmt.Printf("Current tag: %s\n", currentTag)
		fmt.Printf("Latest tag: %s\n", latestTag)

		if hasLocalChanges {
			fmt.Println("\nWarning: You have local changes in your DMS configuration.")
			if offerReclone(dmsPath) {
				return nil
			}
			return errdefs.ErrUpdateCancelled
		}

		fmt.Printf("Updating to %s...\n", latestTag)
		checkoutCmd := exec.Command("git", "checkout", latestTag)
		checkoutCmd.Stdout = os.Stdout
		checkoutCmd.Stderr = os.Stderr
		if err := checkoutCmd.Run(); err != nil {
			fmt.Printf("Error: Failed to checkout %s: %v\n", latestTag, err)
			if offerReclone(dmsPath) {
				return nil
			}
			return fmt.Errorf("update cancelled")
		}

		fmt.Printf("\nUpdate complete! Updated from %s to %s\n", currentTag, latestTag)
		return nil
	}

	if currentBranch == "" {
		currentBranch = "master"
	}

	fmt.Printf("Current branch: %s\n", currentBranch)

	if hasLocalChanges {
		fmt.Println("\nWarning: You have local changes in your DMS configuration.")
		if offerReclone(dmsPath) {
			return nil
		}
		return errdefs.ErrUpdateCancelled
	}

	pullCmd := exec.Command("git", "pull", "origin", currentBranch)
	pullCmd.Stdout = os.Stdout
	pullCmd.Stderr = os.Stderr
	if err := pullCmd.Run(); err != nil {
		fmt.Printf("Error: Failed to pull latest changes: %v\n", err)
		if offerReclone(dmsPath) {
			return nil
		}
		return fmt.Errorf("update cancelled")
	}

	fmt.Println("\nUpdate complete!")
	return nil
}

func offerReclone(dmsPath string) bool {
	fmt.Println("\nWould you like to backup and re-clone the repository? (y/N): ")
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil || !strings.HasPrefix(strings.ToLower(strings.TrimSpace(response)), "y") {
		return false
	}

	timestamp := time.Now().Unix()
	backupPath := fmt.Sprintf("%s.backup-%d", dmsPath, timestamp)

	fmt.Printf("Backing up current directory to %s...\n", backupPath)
	if err := os.Rename(dmsPath, backupPath); err != nil {
		fmt.Printf("Error: Failed to backup directory: %v\n", err)
		return false
	}

	fmt.Println("Cloning fresh copy...")
	cloneCmd := exec.Command("git", "clone", "https://github.com/AvengeMedia/DankMaterialShell.git", dmsPath)
	cloneCmd.Stdout = os.Stdout
	cloneCmd.Stderr = os.Stderr
	if err := cloneCmd.Run(); err != nil {
		fmt.Printf("Error: Failed to clone repository: %v\n", err)
		fmt.Printf("Restoring backup...\n")
		os.Rename(backupPath, dmsPath)
		return false
	}

	fmt.Printf("Successfully re-cloned repository (backup at %s)\n", backupPath)
	return true
}

func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

func isPackageInstalled(packageName string) bool {
	cmd := exec.Command("pacman", "-Q", packageName)
	err := cmd.Run()
	return err == nil
}

func confirmUpdate() bool {
	fmt.Print("Do you want to proceed with the update? (y/N): ")
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("Error reading input: %v\n", err)
		return false
	}
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

func updateDMSBinary() error {
	arch := ""
	switch strings.ToLower(os.Getenv("HOSTTYPE")) {
	case "x86_64", "amd64":
		arch = "amd64"
	case "aarch64", "arm64":
		arch = "arm64"
	default:
		cmd := exec.Command("uname", "-m")
		output, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("failed to detect architecture: %w", err)
		}
		archStr := strings.TrimSpace(string(output))
		switch archStr {
		case "x86_64":
			arch = "amd64"
		case "aarch64":
			arch = "arm64"
		default:
			return fmt.Errorf("unsupported architecture: %s", archStr)
		}
	}

	fmt.Println("Fetching latest release version...")
	cmd := exec.Command("curl", "-s", "https://api.github.com/repos/AvengeMedia/danklinux/releases/latest")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to fetch latest release: %w", err)
	}

	version := ""
	for _, line := range strings.Split(string(output), "\n") {
		if strings.Contains(line, "\"tag_name\"") {
			parts := strings.Split(line, "\"")
			if len(parts) >= 4 {
				version = parts[3]
				break
			}
		}
	}

	if version == "" {
		return fmt.Errorf("could not determine latest version")
	}

	fmt.Printf("Latest version: %s\n", version)

	tempDir, err := os.MkdirTemp("", "dms-update-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	binaryURL := fmt.Sprintf("https://github.com/AvengeMedia/danklinux/releases/download/%s/dms-%s.gz", version, arch)
	checksumURL := fmt.Sprintf("https://github.com/AvengeMedia/danklinux/releases/download/%s/dms-%s.gz.sha256", version, arch)

	binaryPath := filepath.Join(tempDir, "dms.gz")
	checksumPath := filepath.Join(tempDir, "dms.gz.sha256")

	fmt.Println("Downloading dms binary...")
	downloadCmd := exec.Command("curl", "-L", binaryURL, "-o", binaryPath)
	if err := downloadCmd.Run(); err != nil {
		return fmt.Errorf("failed to download binary: %w", err)
	}

	fmt.Println("Downloading checksum...")
	downloadCmd = exec.Command("curl", "-L", checksumURL, "-o", checksumPath)
	if err := downloadCmd.Run(); err != nil {
		return fmt.Errorf("failed to download checksum: %w", err)
	}

	fmt.Println("Verifying checksum...")
	checksumData, err := os.ReadFile(checksumPath)
	if err != nil {
		return fmt.Errorf("failed to read checksum file: %w", err)
	}
	expectedChecksum := strings.Fields(string(checksumData))[0]

	actualCmd := exec.Command("sha256sum", binaryPath)
	actualOutput, err := actualCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}
	actualChecksum := strings.Fields(string(actualOutput))[0]

	if expectedChecksum != actualChecksum {
		return fmt.Errorf("checksum verification failed\nExpected: %s\nGot: %s", expectedChecksum, actualChecksum)
	}

	fmt.Println("Decompressing binary...")
	decompressCmd := exec.Command("gunzip", binaryPath)
	if err := decompressCmd.Run(); err != nil {
		return fmt.Errorf("failed to decompress binary: %w", err)
	}

	decompressedPath := filepath.Join(tempDir, "dms")

	if err := os.Chmod(decompressedPath, 0755); err != nil {
		return fmt.Errorf("failed to make binary executable: %w", err)
	}

	currentPath, err := exec.LookPath("dms")
	if err != nil {
		return fmt.Errorf("could not find current dms binary: %w", err)
	}

	fmt.Printf("Installing to %s...\n", currentPath)

	replaceCmd := exec.Command("sudo", "install", "-m", "0755", decompressedPath, currentPath)
	replaceCmd.Stdin = os.Stdin
	replaceCmd.Stdout = os.Stdout
	replaceCmd.Stderr = os.Stderr
	if err := replaceCmd.Run(); err != nil {
		return fmt.Errorf("failed to replace binary: %w", err)
	}

	return nil
}
