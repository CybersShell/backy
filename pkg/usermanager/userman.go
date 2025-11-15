package usermanager

import (
	"fmt"
	"strings"

	"git.andrewnw.xyz/CyberShell/backy/pkg/usermanager/linux"
)

// UserManager defines the interface for user management operations.
// All functions but one return a string for the command and any args.
type UserManager interface {
	AddUser(username, homeDir, shell string, createHome, isSystem bool, groups, args []string) (string, []string)
	RemoveUser(username string) (string, []string)
	ModifyUser(username, homeDir, shell string, groups []string) (string, []string)
	// Modify password uses chpasswd for Linux systems to build the command to change the password
	// Should return a password as the last argument
	// TODO: refactor when adding more systems instead of Linux
	ModifyPassword(username, password string) (string, *strings.Reader, string)
	UserExists(username string) (string, []string)
}

// NewUserManager returns a UserManager-compatible struct
func NewUserManager(system string) (UserManager, error) {
	var manager UserManager

	switch system {
	case "linux", "Linux":
		manager = linux.LinuxUserManager{}
	default:
		return nil, fmt.Errorf("usermanger system %s is not recognized", system)
	}

	return manager, nil

}
