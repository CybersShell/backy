package pkgman

import (
	"fmt"

	"git.andrewnw.xyz/CyberShell/backy/pkg/pkgman/apt"
	packagemanagercommon "git.andrewnw.xyz/CyberShell/backy/pkg/pkgman/common"
	"git.andrewnw.xyz/CyberShell/backy/pkg/pkgman/dnf"
	"git.andrewnw.xyz/CyberShell/backy/pkg/pkgman/yum"
)

// PackageManager is an interface used to define common package commands. This shall be implemented by every package.
type PackageManager interface {
	Install(pkgs []packagemanagercommon.Package, args []string) (string, []string)
	Remove(pkgs []packagemanagercommon.Package, args []string) (string, []string)
	Upgrade(pkgs []packagemanagercommon.Package) (string, []string) // Upgrade a specific package
	UpgradeAll() (string, []string)
	CheckVersion(pkgs []packagemanagercommon.Package) (string, []string)
	ParseRemotePackageManagerVersionOutput(output string) ([]packagemanagercommon.Package, []error)
	// Configure applies functional options to customize the package manager.
	Configure(options ...packagemanagercommon.PackageManagerOption)
}

// PackageManagerFactory returns the appropriate PackageManager based on the package tool.
// Takes variable number of options.
func PackageManagerFactory(managerType string, options ...packagemanagercommon.PackageManagerOption) (PackageManager, error) {
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
func WithAuth(authCommand string) packagemanagercommon.PackageManagerOption {
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
func WithoutAuth() packagemanagercommon.PackageManagerOption {
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
	packagemanagercommon.PackageParser
	SetUseAuth(useAuth bool)
	SetAuthCommand(authCommand string)
	SetPackageParser(parser packagemanagercommon.PackageParser)
}
