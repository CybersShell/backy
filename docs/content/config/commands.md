---
title: "Commands"
description: Commands are just that, commands
weight: 1
---

The yaml top-level map can be any string.

The top-level name must be unique.

### Example Config

```yaml
commands:
  stop-docker-container:
    cmd: docker
    Args:
      - compose
      - -f /some/path/to/docker-compose.yaml
      - down
    # if host is not defined, command will be run locally
    # The host has to be defined in either the config file or the SSH Config files
    host: some-host
    hooks
      error:
        - some-other-command-when-failing
      success:
        - success-command
      final:
        - final-command
  backup-docker-container-script:
    cmd: /path/to/local/script
    # script file is input as stdin to SSH
    type: scriptFile # also can be script
    environment:
      - FOO=BAR
      - APP=$VAR
```

Values available for this section **(case-sensitive)**:

| name | notes | type | required
| --- | --- | --- | --- |
| `cmd` | Defines the command to execute | `string` | yes |
| `Args` | Defines the arguments to the command | `[]string` | no |
| `environment` | Defines evironment variables for the command | `[]string` | no |
| `type` | May be `scriptFile`, `script`, or `package`. Runs script from local machine on remote. `Package` is the only one that can be run on local and remote hosts. | `string` | no |
| `getOutput` | Command(s) output is in the notification(s) | `bool` | no |
| `host` | If not specified, the command will execute locally. | `string` | no |
| `scriptEnvFile` | When type is `scriptFile` or `script`, this file is prepended to the input. | `string` | no |
| `shell` | Run the command in the shell | `string` | no |
| `hooks` | Hooks are used at the end of the individual command. Must have at least `error`, `success`, or `final`. | `map[string][]string` | no |

#### cmd

cmd must be a valid command or script to execute.

#### Args

args must be arguments to cmd as they would be passed on the command-line:

```sh
cmd [arg1 arg2 ...]
```

Define them in an array:

```yaml
Args:
  - arg1
  - arg2
  - arg3
```

### getOutput

Get command output when a notification is sent.

Is not required. Can be `true` or `false`.

#### host

{{% notice info %}}
If any `host` is not defined or left blank, the command will run on the local machine.
{{% /notice %}}

Host may or may not be defined in the `hosts` section.

{{% notice info %}}
If any `host` from the commands section does not match any object in the `hosts` section, the `Host` is assumed to be this value. This value will be used to search in the default SSH config files.

For example, say that I have a host defined in my SSH config with the `Host` defined as `web-prod`.
If I assign a value to host as `host: web-prod` and don't specify this value in the `hosts` object, web-prod will be used as the `Host` in searching the SSH config files.
{{% /notice %}}

### shell

If shell is defined, the command will run in the specified shell.
Make sure to escape any shell input.

### scriptEnvFile

Path to a file.

When type is `script` or `scriptFile` , the script is appended to this file.

This is useful for specifying environment variables or other things so they don't have to be included in the script.

### type

May be `scriptFile` or `script`. Runs script from local machine on remote host passed to the SSH session as standard input.

If `type` is `script`, `cmd` is used as the script.

If `type` is `scriptFile`, cmd must be a script file.

If `type` is `package`, there are additional fields that must be specified.

### environment

The environment variables support expansion:

- using escaped values `$VAR` or `${VAR}`

For now, the variables have to be defined in an `.env` file in the same directory as the config file.

If using it with host specified, the SSH server has to be configured to accept those env variables.

If the command is run locally, the OS's environment is added.

### hooks

Hooks are run after the command is run.

Errors are run if the command errors, success if it returns no error. Final hooks are run regardless of error condition.

Values for hooks are as follows:

```yaml
command:
  hook:
    # these commands are defined elsewhere in the file
    error:
      - errcommand
    success:
      - successcommand
    final:
      - donecommand
```

### packages

See the [dedicated page](/config/packages) for package configuration.
