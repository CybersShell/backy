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
```backy backup -c /path/to/file```

If you leave the config path blank, the following paths will be searched in order:

- `./backy.yaml`
- `~/.config/backy.yaml`

Create a file at `~/.config/backy.yaml`:

```yaml
commands:
  stop-docker-container:
    cmd: docker
    cmdArgs:
      - compose
      - -f /some/path/to/docker-compose.yaml
      - down
    # if host is not defined, 
    host: some-host 
    env: ~/path/to/env/file
  backup-docker-container-script:
    cmd: /path/to/script
    host: some-host
    env: ~/path/to/env/file
  shell-cmd:
    cmd: rsync
    shell: bash
    cmdArgs:
      - -av some-host:/path/to/data ~/Docker/Backups/docker-data
  hostname:
    cmd: hostname

cmd-configs:
  # this can be any name you want
  cmds-to-run: 
    # all commands have to be defined
    order:
      - stop-docker-container
      - backup-docker-container-script
      - shell-cmd
      - hostname
    notifications:
      - matrix
  hostname:
    order:
      - hostname
    notifications:
      - prod-email

hosts:
  some-host:
    config:
      usefile: true
      user: root
      private-key-path:

logging:
  verbose: true
  file: /path/to/logs/commands.log


notifications:
  prod-email:
    id: prod-email
    type: mail
    host: yourhost.tld
    port: 587
    senderAddress: email@domain.tld
    to:
      - admin@domain.tld
    username: smtp-username@domain.tld
    password: your-password-here
  matrix:
    id: matrix
    type: matrix
    homeserver: your-home-server.tld
    room-id: room-id
    access-token: your-access-token
    user-id: your-user-id

```

Note, let me know if a path lookup fails due to using Go's STDLib `os`
