package apt

import (
	"fmt"

	"git.andrewnw.xyz/CyberShell/backy/pkg/pkgman/pkgcommon"
)

// AptManager implements PackageManager for systems using APT.
type AptManager struct {
	useAuth     bool   // Whether to use an authentication command
	authCommand string // The authentication command, e.g., "sudo"
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
