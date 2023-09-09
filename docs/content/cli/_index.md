---
title: CLI
weight: 4
---

This page lists documentation for the CLI.

## Backy 
 
```
Backy is a command-line application useful for configuring backups, or any commands run in sequence.

Usage:
  backy [command]

Available Commands:
  backup      Runs commands defined in config file.
  completion  Generate the autocompletion script for the specified shell
  cron        Starts a scheduler that runs lists defined in config file.
  exec        Runs commands defined in config file in order given.
  help        Help about any command
  list        Lists commands, lists, or hosts defined in config file.
  version     Prints the version and exits

Flags:
  -f, --config string   config file to read from
  -h, --help            help for backy
  -v, --verbose         Sets verbose level

Use "backy [command] --help" for more information about a command.
```
 
# Subcommands

## backup

```
Backup executes commands defined in config file.
Use the --lists or -l flag to execute the specified lists. If not flag is not given, all lists will be executed.

Usage:
  backy backup [--lists=list1,list2,... | -l list1, list2,...] [flags]

Flags:
  -h, --help            help for backup
  -l, --lists strings   Accepts comma-separated names of command lists to execute.

Global Flags:
  -f, --config string   config file to read from
  -v, --verbose         Sets verbose level
```

## cron

```
Cron starts a scheduler that executes command lists at the time defined in config file.

Usage:
  backy cron [flags]

Flags:
  -h, --help   help for cron

Global Flags:
  -f, --config string   config file to read from
  -v, --verbose         Sets verbose level
```

## exec

```
Exec executes commands defined in config file in order given.

Usage:
  backy exec command ... [flags]

Flags:
  -h, --help   help for exec

Global Flags:
  -f, --config string   config file to read from
  -v, --verbose         Sets verbose level
```

## version

```
Prints the version and exits. No arguments just prints the version number only.

Usage:
  backy version [flags]

Flags:
  -h, --help   help for version
  -n, --num    Output the version number only.
  -V, --vpre   Output the version with v prefixed.

Global Flags:
  -f, --config string   config file to read from
  -v, --verbose         Sets verbose level
```

## list

```
Backup lists commands or groups defined in config file.
Use the --lists or -l flag to list the specified lists. If not flag is not given, all lists will be executed.

Usage:
  backy list [--list=list1,list2,... | -l list1, list2,...] [ -cmd cmd1 cmd2 cmd3...] [flags]

Flags:
  -h, --help            help for list
  -l, --lists strings   Accepts comma-separated names of command lists to list.

Global Flags:
  -f, --config string   config file to read from
  -v, --verbose         Sets verbose level
```
