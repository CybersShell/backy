package pkgman

import (
	"fmt"

	"git.andrewnw.xyz/CyberShell/backy/pkg/pkgman/apt"
	"git.andrewnw.xyz/CyberShell/backy/pkg/pkgman/dnf"
	"git.andrewnw.xyz/CyberShell/backy/pkg/pkgman/pkgcommon"
	"git.andrewnw.xyz/CyberShell/backy/pkg/pkgman/yum"
)

// PackageManager is an interface used to define common package commands. This shall be implemented by every package.
type PackageManager interface {
	Install(pkg, version string, args []string) (string, []string)
	Remove(pkg string, args []string) (string, []string)
	Upgrade(pkg, version string) (string, []string) // Upgrade a specific package
	UpgradeAll() (string, []string)
	CheckVersion(pkg, version string) (string, []string)
	Parse(output string) (*pkgcommon.PackageVersion, error)
	// Configure applies functional options to customize the package manager.
	Configure(options ...pkgcommon.PackageManagerOption)
}

// PackageManagerFactory returns the appropriate PackageManager based on the package tool.
// Takes variable number of options.
func PackageManagerFactory(managerType string, options ...pkgcommon.PackageManagerOption) (PackageManager, error) {
	var manager PackageManager

	switch managerType {
	case "apt":
		manager = apt.NewAptManager()
	case "yum":
		manager = yum.NewYumManager()
	case "dnf":
		manager = dnf.NewDnfManager()
	default:
		return nil, fmt.Errorf("unsupported package manager: %s", managerType)
	}

	// Apply options to the manager
	manager.Configure(options...)
	return manager, nil
}

// WithAuth enables authentication and sets the authentication command.
func WithAuth(authCommand string) pkgcommon.PackageManagerOption {
	return func(manager interface{}) {
		if configurable, ok := manager.(interface {
			SetUseAuth(bool)
			SetAuthCommand(string)
		}); ok {
			configurable.SetUseAuth(true)
			configurable.SetAuthCommand(authCommand)
		}
	}
}

// WithoutAuth disables authentication.
func WithoutAuth() pkgcommon.PackageManagerOption {
	return func(manager interface{}) {
		if configurable, ok := manager.(interface {
			SetUseAuth(bool)
		}); ok {
			configurable.SetUseAuth(false)
		}
	}
}

// ConfigurablePackageManager defines methods for setting configuration options.
type ConfigurablePackageManager interface {
	pkgcommon.PackageParser
	SetUseAuth(useAuth bool)
	SetAuthCommand(authCommand string)
	SetPackageParser(parser pkgcommon.PackageParser)
}
