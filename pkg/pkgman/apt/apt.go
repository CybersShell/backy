package apt

import (
	"fmt"
	"regexp"
	"strings"

	"git.andrewnw.xyz/CyberShell/backy/pkg/pkgman/pkgcommon"
)

// AptManager implements PackageManager for systems using APT.
type AptManager struct {
	useAuth     bool   // Whether to use an authentication command
	authCommand string // The authentication command, e.g., "sudo"
	Parser      pkgcommon.PackageParser
}

// DefaultAuthCommand is the default command used for authentication.
const DefaultAuthCommand = "sudo"

const DefaultPackageCommand = "apt-get"

// NewAptManager creates a new AptManager with default settings.
func NewAptManager() *AptManager {
	return &AptManager{
		useAuth:     true,
		authCommand: DefaultAuthCommand,
	}
}

// Install returns the command and arguments for installing a package.
func (a *AptManager) Install(pkg, version string, args []string) (string, []string) {
	baseCmd := a.prependAuthCommand(DefaultPackageCommand)
	baseArgs := []string{"update", "&&", baseCmd, "install", "-y"}
	if version != "" {
		baseArgs = append(baseArgs, fmt.Sprintf("%s=%s", pkg, version))
	} else {
		baseArgs = append(baseArgs, pkg)
	}
	if args != nil {
		baseArgs = append(baseArgs, args...)
	}
	return baseCmd, baseArgs
}

// Remove returns the command and arguments for removing a package.
func (a *AptManager) Remove(pkg string, args []string) (string, []string) {
	baseCmd := a.prependAuthCommand(DefaultPackageCommand)
	baseArgs := []string{"remove", "-y", pkg}
	if args != nil {
		baseArgs = append(baseArgs, args...)
	}
	return baseCmd, baseArgs
}

// Upgrade returns the command and arguments for upgrading a specific package.
func (a *AptManager) Upgrade(pkg, version string) (string, []string) {
	baseCmd := a.prependAuthCommand(DefaultPackageCommand)
	baseArgs := []string{"update", "&&", baseCmd, "install", "--only-upgrade", "-y "}
	if version != "" {
		baseArgs = append(baseArgs, fmt.Sprintf("%s=%s", pkg, version))
	} else {
		baseArgs = append(baseArgs, pkg)
	}
	return baseCmd, baseArgs
}

// CheckVersion returns the command and arguments for checking the info of a specific package.
func (a *AptManager) CheckVersion(pkg, version string) (string, []string) {
	baseCmd := a.prependAuthCommand("apt-cache")
	baseArgs := []string{"policy", pkg}

	return baseCmd, baseArgs
}

// UpgradeAll returns the command and arguments for upgrading all packages.
func (a *AptManager) UpgradeAll() (string, []string) {
	baseCmd := a.prependAuthCommand(DefaultPackageCommand)
	baseArgs := []string{"update", "&&", baseCmd, "upgrade", "-y"}
	return baseCmd, baseArgs
}

// Configure applies functional options to customize the package manager.
func (a *AptManager) Configure(options ...pkgcommon.PackageManagerOption) {
	for _, opt := range options {
		opt(a)
	}
}

// prependAuthCommand prepends the authentication command if UseAuth is true.
func (a *AptManager) prependAuthCommand(baseCmd string) string {
	if a.useAuth {
		return a.authCommand + " " + baseCmd
	}
	return baseCmd
}

// SetUseAuth enables or disables authentication.
func (a *AptManager) SetUseAuth(useAuth bool) {
	a.useAuth = useAuth
}

// SetAuthCommand sets the authentication command.
func (a *AptManager) SetAuthCommand(authCommand string) {
	a.authCommand = authCommand
}

// SetPackageParser assigns a PackageParser.
func (a *AptManager) SetPackageParser(parser pkgcommon.PackageParser) {
	a.Parser = parser
}

// Parse parses the apt-cache policy output to extract Installed and Candidate versions.
func (a *AptManager) Parse(output string) (*pkgcommon.PackageVersion, error) {
	// Check for error message in the output
	if strings.Contains(output, "Unable to locate package") {
		return nil, fmt.Errorf("error: %s", strings.TrimSpace(output))
	}

	reInstalled := regexp.MustCompile(`Installed:\s*([^\s]+)`)
	reCandidate := regexp.MustCompile(`Candidate:\s*([^\s]+)`)

	installedMatch := reInstalled.FindStringSubmatch(output)
	candidateMatch := reCandidate.FindStringSubmatch(output)

	if len(installedMatch) < 2 || len(candidateMatch) < 2 {
		return nil, fmt.Errorf("failed to parse Installed or Candidate versions from apt output. check package name")
	}

	return &pkgcommon.PackageVersion{
		Installed: strings.TrimSpace(installedMatch[1]),
		Candidate: strings.TrimSpace(candidateMatch[1]),
		Match:     installedMatch[1] == candidateMatch[1],
	}, nil
}
