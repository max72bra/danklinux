package distros

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/AvengeMedia/danklinux/internal/deps"
)

func init() {
	Register("gentoo", "#54487A", FamilyGentoo, func(config DistroConfig, logChan chan<- string) Distribution {
		return NewGentooDistribution(config, logChan)
	})
}

type GentooDistribution struct {
	*BaseDistribution
	*ManualPackageInstaller
	config DistroConfig
}

func NewGentooDistribution(config DistroConfig, logChan chan<- string) *GentooDistribution {
	base := NewBaseDistribution(logChan)
	return &GentooDistribution{
		BaseDistribution:       base,
		ManualPackageInstaller: &ManualPackageInstaller{BaseDistribution: base},
		config:                 config,
	}
}

func (g *GentooDistribution) GetID() string {
	return g.config.ID
}

func (g *GentooDistribution) GetColorHex() string {
	return g.config.ColorHex
}

func (g *GentooDistribution) GetFamily() DistroFamily {
	return g.config.Family
}

func (g *GentooDistribution) GetPackageManager() PackageManagerType {
	return PackageManagerPortage
}

func (g *GentooDistribution) DetectDependencies(ctx context.Context, wm deps.WindowManager) ([]deps.Dependency, error) {
	return g.DetectDependenciesWithTerminal(ctx, wm, deps.TerminalGhostty)
}

func (g *GentooDistribution) DetectDependenciesWithTerminal(ctx context.Context, wm deps.WindowManager, terminal deps.Terminal) ([]deps.Dependency, error) {
	var dependencies []deps.Dependency

	dependencies = append(dependencies, g.detectDMS())

	dependencies = append(dependencies, g.detectSpecificTerminal(terminal))

	dependencies = append(dependencies, g.detectGit())
	dependencies = append(dependencies, g.detectWindowManager(wm))
	dependencies = append(dependencies, g.detectQuickshell())
	dependencies = append(dependencies, g.detectXDGPortal())
	dependencies = append(dependencies, g.detectPolkitAgent())
	dependencies = append(dependencies, g.detectAccountsService())

	if wm == deps.WindowManagerHyprland {
		dependencies = append(dependencies, g.detectHyprlandTools()...)
	}

	if wm == deps.WindowManagerNiri {
		dependencies = append(dependencies, g.detectXwaylandSatellite())
	}

	dependencies = append(dependencies, g.detectMatugen())
	dependencies = append(dependencies, g.detectDgop())
	dependencies = append(dependencies, g.detectHyprpicker())
	dependencies = append(dependencies, g.detectClipboardTools()...)

	return dependencies, nil
}

func (g *GentooDistribution) detectXDGPortal() deps.Dependency {
	status := deps.StatusMissing
	if g.packageInstalled("sys-apps/xdg-desktop-portal-gtk") {
		status = deps.StatusInstalled
	}

	return deps.Dependency{
		Name:        "xdg-desktop-portal-gtk",
		Status:      status,
		Description: "Desktop integration portal for GTK",
		Required:    true,
	}
}

func (g *GentooDistribution) detectPolkitAgent() deps.Dependency {
	status := deps.StatusMissing
	if g.packageInstalled("mate-extra/mate-polkit") {
		status = deps.StatusInstalled
	}

	return deps.Dependency{
		Name:        "mate-polkit",
		Status:      status,
		Description: "PolicyKit authentication agent",
		Required:    true,
	}
}

func (g *GentooDistribution) detectXwaylandSatellite() deps.Dependency {
	status := deps.StatusMissing
	if g.commandExists("xwayland-satellite") {
		status = deps.StatusInstalled
	}

	return deps.Dependency{
		Name:        "xwayland-satellite",
		Status:      status,
		Description: "Xwayland support",
		Required:    true,
	}
}

func (g *GentooDistribution) detectAccountsService() deps.Dependency {
	status := deps.StatusMissing
	if g.packageInstalled("sys-apps/accountsservice") {
		status = deps.StatusInstalled
	}

	return deps.Dependency{
		Name:        "accountsservice",
		Status:      status,
		Description: "D-Bus interface for user account query and manipulation",
		Required:    true,
	}
}

func (g *GentooDistribution) packageInstalled(pkg string) bool {
	cmd := exec.Command("qlist", "-I", pkg)
	err := cmd.Run()
	return err == nil
}

func (g *GentooDistribution) GetPackageMapping(wm deps.WindowManager) map[string]PackageMapping {
	return g.GetPackageMappingWithVariants(wm, make(map[string]deps.PackageVariant))
}

func (g *GentooDistribution) GetPackageMappingWithVariants(wm deps.WindowManager, variants map[string]deps.PackageVariant) map[string]PackageMapping {
	packages := map[string]PackageMapping{
		"git":                    {Name: "dev-vcs/git", Repository: RepoTypeSystem},
		"ghostty":                {Name: "x11-terms/ghostty", Repository: RepoTypeSystem},
		"kitty":                  {Name: "x11-terms/kitty", Repository: RepoTypeSystem},
		"alacritty":              {Name: "x11-terms/alacritty", Repository: RepoTypeSystem},
		"wl-clipboard":           {Name: "gui-apps/wl-clipboard", Repository: RepoTypeSystem},
		"xdg-desktop-portal-gtk": {Name: "sys-apps/xdg-desktop-portal-gtk", Repository: RepoTypeSystem},
		"mate-polkit":            {Name: "mate-extra/mate-polkit", Repository: RepoTypeSystem},
		"accountsservice":        {Name: "sys-apps/accountsservice", Repository: RepoTypeSystem},
		"hyprpicker":             g.getHyprpickerMapping(variants["hyprland"]),

		"quickshell":              g.getQuickshellMapping(variants["quickshell"]),
		"matugen":                 {Name: "x11-misc/matugen", Repository: RepoTypeGURU},
		"cliphist":                {Name: "app-misc/cliphist", Repository: RepoTypeGURU},
		"dms (DankMaterialShell)": g.getDmsMapping(variants["dms (DankMaterialShell)"]),
		"dgop":                    {Name: "dgop", Repository: RepoTypeManual, BuildFunc: "installDgop"},
	}

	switch wm {
	case deps.WindowManagerHyprland:
		packages["hyprland"] = g.getHyprlandMapping(variants["hyprland"])
		packages["grim"] = PackageMapping{Name: "gui-apps/grim", Repository: RepoTypeSystem}
		packages["slurp"] = PackageMapping{Name: "gui-apps/slurp", Repository: RepoTypeSystem}
		packages["hyprctl"] = g.getHyprlandMapping(variants["hyprland"])
		packages["grimblast"] = PackageMapping{Name: "grimblast", Repository: RepoTypeManual, BuildFunc: "installGrimblast"}
		packages["jq"] = PackageMapping{Name: "app-misc/jq", Repository: RepoTypeSystem}
	case deps.WindowManagerNiri:
		packages["niri"] = g.getNiriMapping(variants["niri"])
		packages["xwayland-satellite"] = PackageMapping{Name: "xwayland-satellite", Repository: RepoTypeManual, BuildFunc: "installXwaylandSatellite"}
	}

	return packages
}

func (g *GentooDistribution) getQuickshellMapping(variant deps.PackageVariant) PackageMapping {
	if forceQuickshellGit || variant == deps.VariantGit {
		return PackageMapping{Name: "gui-apps/quickshell", Repository: RepoTypeGURU}
	}
	return PackageMapping{Name: "gui-apps/quickshell", Repository: RepoTypeGURU}
}

func (g *GentooDistribution) getDmsMapping(_ deps.PackageVariant) PackageMapping {
	return PackageMapping{Name: "dms", Repository: RepoTypeManual, BuildFunc: "installDankMaterialShell"}
}

func (g *GentooDistribution) getHyprlandMapping(variant deps.PackageVariant) PackageMapping {
	if variant == deps.VariantGit {
		return PackageMapping{Name: "gui-wm/hyprland", Repository: RepoTypeGURU}
	}
	return PackageMapping{Name: "gui-wm/hyprland", Repository: RepoTypeSystem}
}

func (g *GentooDistribution) getHyprpickerMapping(_ deps.PackageVariant) PackageMapping {
	return PackageMapping{Name: "gui-apps/hyprpicker", Repository: RepoTypeGURU}
}

func (g *GentooDistribution) getNiriMapping(variant deps.PackageVariant) PackageMapping {
	if variant == deps.VariantGit {
		return PackageMapping{Name: "gui-wm/niri", Repository: RepoTypeGURU}
	}
	return PackageMapping{Name: "gui-wm/niri", Repository: RepoTypeSystem}
}

func (g *GentooDistribution) getPrerequisites() []string {
	return []string{
		"app-eselect/eselect-repository",
		"dev-vcs/git",
		"sys-devel/make",
		"app-arch/unzip",
		"dev-util/pkgconf",
	}
}

func (g *GentooDistribution) InstallPrerequisites(ctx context.Context, sudoPassword string, progressChan chan<- InstallProgressMsg) error {
	prerequisites := g.getPrerequisites()
	var missingPkgs []string

	progressChan <- InstallProgressMsg{
		Phase:      PhasePrerequisites,
		Progress:   0.06,
		Step:       "Checking prerequisites...",
		IsComplete: false,
		LogOutput:  "Checking prerequisite packages",
	}

	for _, pkg := range prerequisites {
		checkCmd := exec.CommandContext(ctx, "qlist", "-I", pkg)
		if err := checkCmd.Run(); err != nil {
			missingPkgs = append(missingPkgs, pkg)
		}
	}

	_, err := exec.LookPath("go")
	if err != nil {
		g.log("go not found in PATH, will install dev-lang/go")
		missingPkgs = append(missingPkgs, "dev-lang/go")
	} else {
		g.log("go already available in PATH")
	}

	if len(missingPkgs) == 0 {
		g.log("All prerequisites already installed")
		return nil
	}

	g.log(fmt.Sprintf("Installing prerequisites: %s", strings.Join(missingPkgs, ", ")))
	progressChan <- InstallProgressMsg{
		Phase:       PhasePrerequisites,
		Progress:    0.08,
		Step:        fmt.Sprintf("Installing %d prerequisites...", len(missingPkgs)),
		IsComplete:  false,
		NeedsSudo:   true,
		CommandInfo: fmt.Sprintf("sudo emerge --ask=n %s", strings.Join(missingPkgs, " ")),
		LogOutput:   fmt.Sprintf("Installing prerequisites: %s", strings.Join(missingPkgs, ", ")),
	}

	args := []string{"emerge", "--ask=n", "--quiet"}
	args = append(args, missingPkgs...)
	cmdStr := fmt.Sprintf("echo '%s' | sudo -S %s", sudoPassword, strings.Join(args, " "))
	cmd := exec.CommandContext(ctx, "bash", "-c", cmdStr)
	output, err := cmd.CombinedOutput()
	if err != nil {
		g.logError("failed to install prerequisites", err)
		g.log(fmt.Sprintf("Prerequisites command output: %s", string(output)))
		return fmt.Errorf("failed to install prerequisites: %w", err)
	}
	g.log(fmt.Sprintf("Prerequisites install output: %s", string(output)))

	return nil
}

func (g *GentooDistribution) InstallPackages(ctx context.Context, dependencies []deps.Dependency, wm deps.WindowManager, sudoPassword string, reinstallFlags map[string]bool, progressChan chan<- InstallProgressMsg) error {
	progressChan <- InstallProgressMsg{
		Phase:      PhasePrerequisites,
		Progress:   0.05,
		Step:       "Checking system prerequisites...",
		IsComplete: false,
		LogOutput:  "Starting prerequisite check...",
	}

	if err := g.InstallPrerequisites(ctx, sudoPassword, progressChan); err != nil {
		return fmt.Errorf("failed to install prerequisites: %w", err)
	}

	systemPkgs, guruPkgs, manualPkgs := g.categorizePackages(dependencies, wm, reinstallFlags)

	if len(guruPkgs) > 0 {
		progressChan <- InstallProgressMsg{
			Phase:      PhaseSystemPackages,
			Progress:   0.15,
			Step:       "Enabling GURU repository...",
			IsComplete: false,
			LogOutput:  "Setting up GURU repository for additional packages",
		}
		if err := g.enableGURURepo(ctx, sudoPassword, progressChan); err != nil {
			return fmt.Errorf("failed to enable GURU repository: %w", err)
		}
	}

	if len(systemPkgs) > 0 {
		progressChan <- InstallProgressMsg{
			Phase:      PhaseSystemPackages,
			Progress:   0.35,
			Step:       fmt.Sprintf("Installing %d system packages...", len(systemPkgs)),
			IsComplete: false,
			NeedsSudo:  true,
			LogOutput:  fmt.Sprintf("Installing system packages: %s", strings.Join(systemPkgs, ", ")),
		}
		if err := g.installPortagePackages(ctx, systemPkgs, sudoPassword, progressChan); err != nil {
			return fmt.Errorf("failed to install Portage packages: %w", err)
		}
	}

	guruPkgNames := g.extractPackageNames(guruPkgs)
	if len(guruPkgNames) > 0 {
		progressChan <- InstallProgressMsg{
			Phase:      PhaseAURPackages,
			Progress:   0.65,
			Step:       fmt.Sprintf("Installing %d GURU packages...", len(guruPkgNames)),
			IsComplete: false,
			LogOutput:  fmt.Sprintf("Installing GURU packages: %s", strings.Join(guruPkgNames, ", ")),
		}
		if err := g.installGURUPackages(ctx, guruPkgNames, sudoPassword, progressChan); err != nil {
			return fmt.Errorf("failed to install GURU packages: %w", err)
		}
	}

	if len(manualPkgs) > 0 {
		progressChan <- InstallProgressMsg{
			Phase:      PhaseSystemPackages,
			Progress:   0.85,
			Step:       fmt.Sprintf("Building %d packages from source...", len(manualPkgs)),
			IsComplete: false,
			LogOutput:  fmt.Sprintf("Building from source: %s", strings.Join(manualPkgs, ", ")),
		}
		if err := g.InstallManualPackages(ctx, manualPkgs, sudoPassword, progressChan); err != nil {
			return fmt.Errorf("failed to install manual packages: %w", err)
		}
	}

	progressChan <- InstallProgressMsg{
		Phase:      PhaseConfiguration,
		Progress:   0.90,
		Step:       "Configuring system...",
		IsComplete: false,
		LogOutput:  "Starting post-installation configuration...",
	}

	progressChan <- InstallProgressMsg{
		Phase:      PhaseComplete,
		Progress:   1.0,
		Step:       "Installation complete!",
		IsComplete: true,
		LogOutput:  "All packages installed and configured successfully",
	}

	return nil
}

func (g *GentooDistribution) categorizePackages(dependencies []deps.Dependency, wm deps.WindowManager, reinstallFlags map[string]bool) ([]string, []PackageMapping, []string) {
	systemPkgs := []string{}
	guruPkgs := []PackageMapping{}
	manualPkgs := []string{}

	variantMap := make(map[string]deps.PackageVariant)
	for _, dep := range dependencies {
		variantMap[dep.Name] = dep.Variant
	}

	packageMap := g.GetPackageMappingWithVariants(wm, variantMap)

	for _, dep := range dependencies {
		if dep.Status == deps.StatusInstalled && !reinstallFlags[dep.Name] {
			continue
		}

		pkgInfo, exists := packageMap[dep.Name]
		if !exists {
			g.log(fmt.Sprintf("Warning: No package mapping for %s", dep.Name))
			continue
		}

		switch pkgInfo.Repository {
		case RepoTypeSystem:
			systemPkgs = append(systemPkgs, pkgInfo.Name)
		case RepoTypeGURU:
			guruPkgs = append(guruPkgs, pkgInfo)
		case RepoTypeManual:
			manualPkgs = append(manualPkgs, dep.Name)
		}
	}

	return systemPkgs, guruPkgs, manualPkgs
}

func (g *GentooDistribution) extractPackageNames(packages []PackageMapping) []string {
	names := make([]string, len(packages))
	for i, pkg := range packages {
		names[i] = pkg.Name
	}
	return names
}

func (g *GentooDistribution) enableGURURepo(ctx context.Context, sudoPassword string, progressChan chan<- InstallProgressMsg) error {
	checkCmd := exec.CommandContext(ctx, "eselect", "repository", "list", "-i")
	output, err := checkCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check GURU repository status: %w", err)
	}

	if strings.Contains(string(output), "guru") {
		g.log("GURU repository already enabled")
		return nil
	}

	g.log("Enabling GURU repository...")
	progressChan <- InstallProgressMsg{
		Phase:       PhaseSystemPackages,
		Progress:    0.20,
		Step:        "Enabling GURU repo...",
		IsComplete:  false,
		NeedsSudo:   true,
		CommandInfo: "sudo eselect repository enable guru",
	}

	cmd := exec.CommandContext(ctx, "bash", "-c",
		fmt.Sprintf("echo '%s' | sudo -S eselect repository enable guru 2>&1", sudoPassword))
	enableOutput, err := cmd.CombinedOutput()
	if err != nil {
		g.logError("failed to enable GURU repo", err)
		g.log(fmt.Sprintf("GURU enable command output: %s", string(enableOutput)))
		return fmt.Errorf("failed to enable GURU repo: %w", err)
	}
	g.log(fmt.Sprintf("GURU repo enabled successfully: %s", string(enableOutput)))

	g.log("Syncing GURU repository...")
	progressChan <- InstallProgressMsg{
		Phase:       PhaseSystemPackages,
		Progress:    0.25,
		Step:        "Syncing GURU repo...",
		IsComplete:  false,
		CommandInfo: "emaint sync -r guru",
	}

	syncCmd := exec.CommandContext(ctx, "emaint", "sync", "-r", "guru")
	syncOutput, err := syncCmd.CombinedOutput()
	if err != nil {
		g.logError("failed to sync GURU repo", err)
		g.log(fmt.Sprintf("GURU sync command output: %s", string(syncOutput)))
		return fmt.Errorf("failed to sync GURU repo: %w", err)
	}
	g.log(fmt.Sprintf("GURU repo synced successfully: %s", string(syncOutput)))

	return nil
}

func (g *GentooDistribution) installPortagePackages(ctx context.Context, packages []string, sudoPassword string, progressChan chan<- InstallProgressMsg) error {
	if len(packages) == 0 {
		return nil
	}

	g.log(fmt.Sprintf("Installing Portage packages: %s", strings.Join(packages, ", ")))

	args := []string{"emerge", "--ask=n", "--quiet"}
	args = append(args, packages...)

	progressChan <- InstallProgressMsg{
		Phase:       PhaseSystemPackages,
		Progress:    0.40,
		Step:        "Installing system packages...",
		IsComplete:  false,
		NeedsSudo:   true,
		CommandInfo: fmt.Sprintf("sudo %s", strings.Join(args, " ")),
	}

	cmdStr := fmt.Sprintf("echo '%s' | sudo -S %s", sudoPassword, strings.Join(args, " "))
	cmd := exec.CommandContext(ctx, "bash", "-c", cmdStr)
	return g.runWithProgress(cmd, progressChan, PhaseSystemPackages, 0.40, 0.60)
}

func (g *GentooDistribution) installGURUPackages(ctx context.Context, packages []string, sudoPassword string, progressChan chan<- InstallProgressMsg) error {
	if len(packages) == 0 {
		return nil
	}

	g.log(fmt.Sprintf("Installing GURU packages: %s", strings.Join(packages, ", ")))

	args := []string{"emerge", "--ask=n", "--quiet"}
	args = append(args, packages...)

	progressChan <- InstallProgressMsg{
		Phase:       PhaseAURPackages,
		Progress:    0.70,
		Step:        "Installing GURU packages...",
		IsComplete:  false,
		NeedsSudo:   true,
		CommandInfo: fmt.Sprintf("sudo %s", strings.Join(args, " ")),
	}

	cmdStr := fmt.Sprintf("echo '%s' | sudo -S %s", sudoPassword, strings.Join(args, " "))
	cmd := exec.CommandContext(ctx, "bash", "-c", cmdStr)
	return g.runWithProgress(cmd, progressChan, PhaseAURPackages, 0.70, 0.85)
}
