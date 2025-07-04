package apt

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"

	packagemanagercommon "git.andrewnw.xyz/CyberShell/backy/pkg/pkgman/common"
)

// AptManager implements PackageManager for systems using APT.
type AptManager struct {
	useAuth     bool   // Whether to use an authentication command
	authCommand string // The authentication command, e.g., "sudo"
	Parser      packagemanagercommon.PackageParser
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
func (a *AptManager) Install(pkgs []packagemanagercommon.Package, args []string) (string, []string) {
	baseCmd := a.prependAuthCommand(DefaultPackageCommand)
	baseArgs := []string{"update", "&&", baseCmd, "install", "-y"}
	for _, p := range pkgs {
		baseArgs = append(baseArgs, p.Name)
	}

	if args != nil {
		baseArgs = append(baseArgs, args...)
	}
	return baseCmd, baseArgs
}

// Remove returns the command and arguments for removing a package.
func (a *AptManager) Remove(pkgs []packagemanagercommon.Package, args []string) (string, []string) {
	baseCmd := a.prependAuthCommand(DefaultPackageCommand)
	baseArgs := []string{"remove", "-y"}
	for _, p := range pkgs {
		baseArgs = append(baseArgs, p.Name)
	}
	if args != nil {
		baseArgs = append(baseArgs, args...)
	}
	return baseCmd, baseArgs
}

func (a *AptManager) Upgrade(pkgs []packagemanagercommon.Package) (string, []string) {
	baseCmd := a.prependAuthCommand(DefaultPackageCommand)
	baseArgs := []string{"update", "&&", baseCmd, "install", "--only-upgrade", "-y"}
	for _, p := range pkgs {
		baseArgs = append(baseArgs, p.Name)
	}

	return baseCmd, baseArgs
}

func (a *AptManager) CheckVersion(pkgs []packagemanagercommon.Package) (string, []string) {
	baseCmd := a.prependAuthCommand("apt-cache")
	baseArgs := []string{"policy"}
	for _, p := range pkgs {
		baseArgs = append(baseArgs, p.Name)
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
func (a *AptManager) Configure(options ...packagemanagercommon.PackageManagerOption) {
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

// Parse parses the apt-cache policy output to extract Installed and Candidate versions.
func (a *AptManager) ParseRemotePackageManagerVersionOutput(output string) ([]packagemanagercommon.Package, []error) {
	var (
		packageName        string
		installedString    string
		candidateString    string
		countRelevantLines int
	)
	// Check for error message in the output
	if strings.Contains(output, "Unable to locate package") {
		return nil, []error{fmt.Errorf("error: %s", strings.TrimSpace(output))}
	}
	packages := []packagemanagercommon.Package{}
	outputBuf := bytes.NewBufferString(output)
	outputScan := bufio.NewScanner(outputBuf)
	for outputScan.Scan() {
		line := outputScan.Text()
		if !strings.HasPrefix(line, "  ") && strings.HasSuffix(line, ":") {
			// count++
			packageName = strings.TrimSpace(strings.TrimSuffix(line, ":"))
		}
		if strings.Contains(line, "Installed:") {
			countRelevantLines++
			installedString = strings.TrimPrefix(strings.TrimSpace(line), "Installed:")
		}

		if strings.Contains(line, "Candidate:") {
			countRelevantLines++
			candidateString = strings.TrimPrefix(strings.TrimSpace(line), "Candidate:")
		}

		if countRelevantLines == 2 {
			countRelevantLines = 0
			packages = append(packages, packagemanagercommon.Package{
				Name: packageName,
				VersionCheck: packagemanagercommon.PackageVersion{
					Installed: strings.TrimSpace(installedString),
					Candidate: strings.TrimSpace(candidateString),
					Match:     installedString == candidateString,
				}},
			)
		}
	}

	return packages, nil
}

func SearchPackages(pkgs []string, version string) (string, []string) {
	baseCommand := "dpkg-query"
	baseArgs := []string{"-W", "-f='${Package}\t${Architecture}\t${db:Status-Status}\t${Version}\t${Installed-Size}\t${Binary:summary}\n'"}
	baseArgs = append(baseArgs, pkgs...)

	return baseCommand, baseArgs
}
