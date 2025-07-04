package yum

import (
	"fmt"
	"regexp"
	"strings"

	packagemanagercommon "git.andrewnw.xyz/CyberShell/backy/pkg/pkgman/common"
)

// YumManager implements PackageManager for systems using YUM.
type YumManager struct {
	useAuth     bool   // Whether to use an authentication command
	authCommand string // The authentication command, e.g., "sudo"
}

// DefaultAuthCommand is the default command used for authentication.
const DefaultAuthCommand = "sudo"

// NewYumManager creates a new YumManager with default settings.
func NewYumManager() *YumManager {
	return &YumManager{
		useAuth:     true,
		authCommand: DefaultAuthCommand,
	}
}

// Configure applies functional options to customize the package manager.
func (y *YumManager) Configure(options ...packagemanagercommon.PackageManagerOption) {
	for _, opt := range options {
		opt(y)
	}
}

// Install returns the command and arguments for installing a package.
func (y *YumManager) Install(pkgs []packagemanagercommon.Package, args []string) (string, []string) {
	baseCmd := y.prependAuthCommand("yum")
	baseArgs := []string{"install", "-y"}
	for _, p := range pkgs {
		baseArgs = append(baseArgs, p.Name)
	}

	if args != nil {
		baseArgs = append(baseArgs, args...)
	}
	return baseCmd, baseArgs
}

// Remove returns the command and arguments for removing a package.
func (y *YumManager) Remove(pkgs []packagemanagercommon.Package, args []string) (string, []string) {
	baseCmd := y.prependAuthCommand("yum")
	baseArgs := []string{"remove", "-y"}
	for _, p := range pkgs {
		baseArgs = append(baseArgs, p.Name)
	}

	if args != nil {
		baseArgs = append(baseArgs, args...)
	}
	return baseCmd, baseArgs
}

// Upgrade returns the command and arguments for upgrading a specific package.
func (y *YumManager) Upgrade(pkgs []packagemanagercommon.Package) (string, []string) {
	baseCmd := y.prependAuthCommand("yum")
	baseArgs := []string{"update", "-y"}
	for _, p := range pkgs {
		baseArgs = append(baseArgs, p.Name)
	}

	return baseCmd, baseArgs
}

// UpgradeAll returns the command and arguments for upgrading all packages.
func (y *YumManager) UpgradeAll() (string, []string) {
	baseCmd := y.prependAuthCommand("yum")
	baseArgs := []string{"update", "-y"}
	return baseCmd, baseArgs
}

// CheckVersion returns the command and arguments for checking the info of a specific package.
func (y *YumManager) CheckVersion(pkgs []packagemanagercommon.Package) (string, []string) {
	baseCmd := y.prependAuthCommand("yum")
	baseArgs := []string{"info"}
	for _, p := range pkgs {
		baseArgs = append(baseArgs, p.Name)
	}

	return baseCmd, baseArgs
}

// Parse parses the dnf info output to extract Installed and Candidate versions.
func (y YumManager) ParseRemotePackageManagerVersionOutput(output string) ([]packagemanagercommon.Package, []error) {

	// Check for error message in the output
	if strings.Contains(output, "No matching packages to list") {
		return nil, []error{fmt.Errorf("error: package not listed")}
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
		return nil, []error{fmt.Errorf("failed to parse versions from dnf output")}
	}

	return nil, nil
}

// prependAuthCommand prepends the authentication command if UseAuth is true.
func (y *YumManager) prependAuthCommand(baseCmd string) string {
	if y.useAuth {
		return y.authCommand + " " + baseCmd
	}
	return baseCmd
}

// SetUseAuth enables or disables authentication.
func (y *YumManager) SetUseAuth(useAuth bool) {
	y.useAuth = useAuth
}

// SetAuthCommand sets the authentication command.
func (y *YumManager) SetAuthCommand(authCommand string) {
	y.authCommand = authCommand
}
