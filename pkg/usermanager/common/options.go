package common

// ConfigurablePackageManager defines methods for setting configuration options.
type ConfigurableUserManager interface {
	SetUseAuth(useAuth bool)
	SetAuthCommand(authCommand string)
}
