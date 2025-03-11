package linux

import (
	"fmt"
	"strings"

	passGen "github.com/sethvargo/go-password/password"
)

// LinuxUserManager implements UserManager for Linux systems.
type LinuxUserManager struct{}

func (l LinuxUserManager) NewLinuxManager() *LinuxUserManager {
	return &LinuxUserManager{}
}

// AddUser adds a new user to the system.
func (l LinuxUserManager) AddUser(username, homeDir, shell string, createHome, isSystem bool, groups, args []string) (string, []string) {
	baseArgs := []string{}

	if isSystem {
		baseArgs = append(baseArgs, "--system")
	}

	if homeDir != "" {
		baseArgs = append(baseArgs, "--home", homeDir)
	}

	if shell != "" {
		baseArgs = append(baseArgs, "--shell", shell)
	}

	if len(groups) > 0 {
		baseArgs = append(baseArgs, "--groups", strings.Join(groups, ","))
	}

	if len(args) > 0 {
		baseArgs = append(baseArgs, args...)
	}

	if createHome {
		baseArgs = append(baseArgs, "-m")

	}

	args = append(baseArgs, username)

	cmd := "useradd"
	return cmd, args
}

func (l LinuxUserManager) ModifyPassword(username, password string) (string, *strings.Reader, string) {
	cmd := "chpasswd"
	if password == "" {
		password = passGen.MustGenerate(20, 5, 5, false, false)
	}
	stdin := strings.NewReader(fmt.Sprintf("%s:%s", username, password))
	return cmd, stdin, password
}

// RemoveUser removes an existing user from the system.
func (l LinuxUserManager) RemoveUser(username string) (string, []string) {
	cmd := "userdel"

	return cmd, []string{username}
}

// ModifyUser modifies an existing user's details.
func (l LinuxUserManager) ModifyUser(username, homeDir, shell string, groups []string) (string, []string) {
	args := []string{}

	if homeDir != "" {
		args = append(args, "--home", homeDir)
	}

	if shell != "" {
		args = append(args, "--shell", shell)
	}

	if len(groups) > 0 {
		args = append(args, "--groups", strings.Join(groups, ","))
	}

	args = append(args, username)

	cmd := "usermod"

	return cmd, args
}

// UserExists checks if a user exists on the system.
func (l LinuxUserManager) UserExists(username string) (string, []string) {
	cmd := "id"
	return cmd, []string{username}
}
