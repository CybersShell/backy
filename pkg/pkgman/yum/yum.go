package yum

import (
	"fmt"

	"git.andrewnw.xyz/CyberShell/backy/pkg/pkgman/pkgcommon"
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
func (y *YumManager) Configure(options ...pkgcommon.PackageManagerOption) {
	for _, opt := range options {
		opt(y)
	}
}

// Install returns the command and arguments for installing a package.
func (y *YumManager) Install(pkg, version string, args []string) (string, []string) {
	baseCmd := y.prependAuthCommand("yum")
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
func (y *YumManager) Remove(pkg string, args []string) (string, []string) {
	baseCmd := y.prependAuthCommand("yum")
	baseArgs := []string{"remove", "-y", pkg}
	if args != nil {
		baseArgs = append(baseArgs, args...)
	}
	return baseCmd, baseArgs
}

// Upgrade returns the command and arguments for upgrading a specific package.
func (y *YumManager) Upgrade(pkg, version string) (string, []string) {
	baseCmd := y.prependAuthCommand("yum")
	baseArgs := []string{"update", "-y"}
	if version != "" {
		baseArgs = append(baseArgs, fmt.Sprintf("%s-%s", pkg, version))
	} else {
		baseArgs = append(baseArgs, pkg)
	}
	return baseCmd, baseArgs
}

// UpgradeAll returns the command and arguments for upgrading all packages.
func (y *YumManager) UpgradeAll() (string, []string) {
	baseCmd := y.prependAuthCommand("yum")
	baseArgs := []string{"update", "-y"}
	return baseCmd, baseArgs
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
