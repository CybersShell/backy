package pkgcommon

// PackageManagerOption defines a functional option for configuring a PackageManager.
type PackageManagerOption func(interface{})

// PackageParser defines an interface for parsing package version information.
type PackageParser interface {
	Parse(output string) (*PackageVersion, error)
}

// PackageVersion represents the installed and candidate versions of a package.
type PackageVersion struct {
	Installed string
	Candidate string
	Match     bool
	Message   string
}
