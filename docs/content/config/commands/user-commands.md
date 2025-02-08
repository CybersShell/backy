---
title: "User commands"
weight: 2
description: This is dedicated to user commands.
---

This is dedicated to `user` commands. The command `type` field must be `user`. User is a type that allows one to perform user operations. There are several additional options available when `type` is `user`:

| name | notes | type | required |
| --- | --- | --- | --- |
| `userName` | The name of a user to be configured. | `string` | yes |
| `userOperation` | The type of operation to perform. | `string` | yes |
| `userID` | The user ID to use. | `string` | yes |
| `userGroups` | The groups the user should be added to. | `[]string` | yes |
| `userShell` | The shell for the user. | `string` | yes |
| `userHome` | The user's home directory. | `string` | no |


#### example

The following is an example of a package command:

```yaml
  addUser:
	name: add user backy with custom home dir
    type: user
    userName: backy
    userHome: /opt/backy
    userOperation: add
    host: some-host
```

#### userOperation

The following package operations are supported:

- `add`
- `remove`
- `modify`
- `password`
- `checkIfExists`

### Development

The UserManager interface provides an way easy to add new commands. There is one interface `Usermanager` in directory `pkg/usermanager`.

#### UserManager

```go
// UserManager defines the interface for user management operations.
// All functions but one return a string for the command and any args.
type UserManager interface {
	AddUser(username, homeDir, shell string, isSystem bool, groups, args []string) (string, []string)
	RemoveUser(username string) (string, []string)
	ModifyUser(username, homeDir, shell string, groups []string) (string, []string)
	// Modify password uses chpasswd for Linux systems to build the command to change the password
	// Should return a password as the last argument
	// TODO: refactor when adding more systems instead of Linux
	ModifyPassword(username, password string) (string, *strings.Reader, string)
	UserExists(username string) (string, []string)
}
```