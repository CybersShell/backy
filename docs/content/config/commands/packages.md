---
title: "Packages"
weight: 2
description: This is dedicated to package commands.
---

This is dedicated to `package` commands. The command `type` field must be `package`. Package is a type that allows one to perform package operations. There are several additional options available when `type` is `package`:

| name | notes | type | required |
| --- | --- | --- | --- |
| `packageName` | The name of a package to be modified. | `[]packagemanagercommon.Package` | yes |
| `packageManager` | The name of the package manger to be used. | `string` | yes |
| `packageOperation` | The type of operation to perform. | `string` | yes |
| `packageVersion` | The version of a package. | `string` | no |


#### example

The following is an example of a package command:

```yaml
 update-docker:
    type: package
    shell: zsh
    packages:
	  - name: docker-ce
	    version: 10
    packageManager: apt
    packageOperation: install
    host: debian-based-host
```

#### packageOperation

The following package operations are supported:

- `install`
- `remove`
- `upgrade`
- `checkVersion`

#### packageManager

The following package managers are recognized:

- `apt`
- `yum`
- `dnf`

#### package command args

You can add additional arguments using the standard `Args` key. This is useful for adding more packages, yet it does not work with `checkVersion`.

### Development

The PackageManager interface provides an easy way to enforce functions and options. There are two interfaces, `PackageManager` and `ConfigurablePackageManager` in the directory `pkg/pkgman`. Go's import-cycle "feature" caused me to implement functional options using a third interface. `PackageManagerOption`is a function that takes an interface.

#### PackageManager

```go
// PackageManager is an interface used to define common package commands. This shall be implemented by every package.
type PackageManager interface {
	Install(pkg, version string, args []string) (string, []string)
	Remove(pkg string, args []string) (string, []string)
	Upgrade(pkg, version string) (string, []string) // Upgrade a specific package
	UpgradeAll() (string, []string)

	// Configure applies functional options to customize the package manager.
	Configure(options ...pkgcommon.PackageManagerOption)
}
```

There are a few functional options that should be implemented using the `ConfigurablePackageManager` interface:

```go
// ConfigurablePackageManager defines methods for setting configuration options.
type ConfigurablePackageManager interface {
	SetUseAuth(useAuth bool)
	SetAuthCommand(authCommand string)
}
```