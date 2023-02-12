# Backy - an application to manage backups

This app is in development, and is currently not stable. Expect core functionality to possiblly break.

## Installing

To install:

`go install git.andrewnw.xyz/CyberShell/backy@master`

This assumes you already have a working Go environment, if not please see [this page](https://golang.org/doc/install) first.

You can also download binaries [here](https://git.andrewnw.xyz/CyberShell/backy/releases) and [here](https://github.com/CybersShell/backy/releases).

## Features

- Allows easy configuration of executable commands

- Allows for commands to be run on many hosts over SSH

- Commands can be grouped in list to run in specific order

- Notifications on completion and failure

- Run in cron mode

- For any command, especially backup commands

To run a config:

`backy backup`

Or to use a specific file:
```backy backup -f /path/to/file```

If you leave the config path blank, the following paths will be searched in order:

- `./backy.yml`
- `./backy.yaml`
- `~/.config/backy.yml`
- `~/.config/backy.yaml`

Create a file at `~/.config/backy.yml`.

See the config file in the examples directory to configure it.  
