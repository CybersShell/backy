# Backy - an application to manage backups

This app is in development, and is currently not stable. Expect core functionality to possiblly break.

## Installing

To install:

`go install git.andrewnw.xyz/CyberShell/backy@master`

This assumes you already have a working Go environment, if not please see [this page](https://golang.org/doc/install) first.

You can also download binaries [here](https://git.andrewnw.xyz/CyberShell/backy/releases) and [here](https://github.com/CybersShell/backy/releases).

## Features

- Define lists of commands and run them

- Execute commands over SSH

- More to come.

To run a config:

`backy backup`

Or to use a specific file:
```backy backup -f /path/to/file```

If you leave the config path blank, the following paths will be searched in order:

- `./backy.yaml`
- `~/.config/backy.yaml`

Create a file at `~/.config/backy.yaml`:

```yaml
commands:
  stop-docker-container:
    cmd: docker
    Args:
      - compose
      - -f /some/path/to/docker-compose.yaml
      - down
    # if host is not defined, cmd will be run locally
    host: some-host 
  backup-docker-container-script:
    cmd: /path/to/script
    # The host has to be defined in the config file
    host: some-host
  shell-cmd:
    cmd: rsync
    shell: bash
    Args:
      - -av some-host:/path/to/data ~/Docker/Backups/docker-data
  hostname:
    cmd: hostname

cmd-configs:
  cmds-to-run: # this can be any name you want
    # all commands have to be defined
    order:
      - stop-docker-container
      - backup-docker-container-script
      - shell-cmd
      - hostname
    notifications:
      - matrix
    name: backup-some-server
  hostname:
    name: hostname
    order:
      - hostname
    notifications:
      - prod-email

hosts:
  some-host:
    hostname: some-hostname
    config: ~/.ssh/config
    user: user
    privatekeypath: /path/to/private/key
    port: 22
    password: 


logging:
  verbose: true
  file: /path/to/logs/commands.log
  console: false
  cmd-std-out: false


notifications:
  prod-email:
    id: prod-email
    type: mail
    host: yourhost.tld:port
    senderAddress: email@domain.tld
    to:
      - admin@domain.tld
    username: smtp-username@domain.tld
    password: your-password-here
  matrix:
    id: matrix
    type: matrix
    home-server: your-home-server.tld
    room-id: room-id
    access-token: your-access-token
    user-id: your-user-id

```
