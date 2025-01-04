package dnf

import (
	"fmt"

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
