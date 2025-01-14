package dnf

import (
	"fmt"
	"regexp"
	"strings"

	"git.andrewnw.xyz/CyberShell/backy/pkg/pkgman/pkgcommon"
)

// DnfManager implements PackageManager for systems using YUM.
type DnfManager struct {
	useAuth     bool   // Whether to use an authentication command
	authCommand string // The authentication command, e.g., "sudo"
}

// DefaultAuthCommand is the default command used for authentication.
const DefaultAuthCommand = "sudo"

// NewDnfManager creates a new DnfManager with default settings.
func NewDnfManager() *DnfManager {
	return &DnfManager{
		useAuth:     true,
		authCommand: DefaultAuthCommand,
	}
}

// Configure applies functional options to customize the package manager.
func (y *DnfManager) Configure(options ...pkgcommon.PackageManagerOption) {
	for _, opt := range options {
		opt(y)
	}
}

// Install returns the command and arguments for installing a package.
func (y *DnfManager) Install(pkg, version string, args []string) (string, []string) {
	baseCmd := y.prependAuthCommand("dnf")
	baseArgs := []string{"install", "-y"}
	if version != "" {
		baseArgs = append(baseArgs, fmt.Sprintf("%s-%s", pkg, version))
	} else {
		baseArgs = append(baseArgs, pkg)
	}
	if args != nil {
		baseArgs = append(baseArgs, args...)
	}
	return baseCmd, baseArgs
}

// Remove returns the command and arguments for removing a package.
func (y *DnfManager) Remove(pkg string, args []string) (string, []string) {
	baseCmd := y.prependAuthCommand("dnf")
	baseArgs := []string{"remove", "-y", pkg}
	if args != nil {
		baseArgs = append(baseArgs, args...)
	}
	return baseCmd, baseArgs
}

// Upgrade returns the command and arguments for upgrading a specific package.
func (y *DnfManager) Upgrade(pkg, version string) (string, []string) {
	baseCmd := y.prependAuthCommand("dnf")
	baseArgs := []string{"update", "-y"}
	if version != "" {
		baseArgs = append(baseArgs, fmt.Sprintf("%s-%s", pkg, version))
	} else {
		baseArgs = append(baseArgs, pkg)
	}
	return baseCmd, baseArgs
}

// UpgradeAll returns the command and arguments for upgrading all packages.
func (y *DnfManager) UpgradeAll() (string, []string) {
	baseCmd := y.prependAuthCommand("dnf")
	baseArgs := []string{"update", "-y"}
	return baseCmd, baseArgs
}

// CheckVersion returns the command and arguments for checking the info of a specific package.
func (d *DnfManager) CheckVersion(pkg, version string) (string, []string) {
	baseCmd := d.prependAuthCommand("dnf")
	baseArgs := []string{"info", pkg}

	return baseCmd, baseArgs
}

// Parse parses the dnf info output to extract Installed and Candidate versions.
func (d DnfManager) Parse(output string) (*pkgcommon.PackageVersion, error) {

	// Check for error message in the output
	if strings.Contains(output, "No matching packages to list") {
		return nil, fmt.Errorf("error: package not listed")
	}

	// Define regular expressions to capture installed and available versions
	reInstalled := regexp.MustCompile(`(?m)^Installed packages\s*Name\s*:\s*\S+\s*Epoch\s*:\s*\S+\s*Version\s*:\s*([^\s]+)\s*Release\s*:\s*([^\s]+)`)
	reAvailable := regexp.MustCompile(`(?m)^Available packages\s*Name\s*:\s*\S+\s*Epoch\s*:\s*\S+\s*Version\s*:\s*([^\s]+)\s*Release\s*:\s*([^\s]+)`)

	installedMatch := reInstalled.FindStringSubmatch(output)
	candidateMatch := reAvailable.FindStringSubmatch(output)

	installedVersion := ""
	candidateVersion := ""

	if len(installedMatch) >= 3 {
		installedVersion = fmt.Sprintf("%s-%s", installedMatch[1], installedMatch[2])
	}

	if len(candidateMatch) >= 3 {
		candidateVersion = fmt.Sprintf("%s-%s", candidateMatch[1], candidateMatch[2])
	}

	if installedVersion == "" && candidateVersion == "" {
		return nil, fmt.Errorf("failed to parse versions from dnf output")
	}

	return &pkgcommon.PackageVersion{
		Installed: installedVersion,
		Candidate: candidateVersion,
	}, nil
}

// prependAuthCommand prepends the authentication command if UseAuth is true.
func (y *DnfManager) prependAuthCommand(baseCmd string) string {
	if y.useAuth {
		return y.authCommand + " " + baseCmd
	}
	return baseCmd
}

// SetUseAuth enables or disables authentication.
func (y *DnfManager) SetUseAuth(useAuth bool) {
	y.useAuth = useAuth
}

// SetAuthCommand sets the authentication command.
func (y *DnfManager) SetAuthCommand(authCommand string) {
	y.authCommand = authCommand
}
