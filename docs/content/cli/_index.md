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
  backup      Runs commands defined in config file. Use -l flag multiple times to run multiple lists.
  completion  Generate the autocompletion script for the specified shell
  cron        Starts a scheduler that runs lists defined in config file.
  exec        Runs commands defined in config file in order given.
  help        Help about any command
  list        List commands, lists, or hosts defined in config file.
  version     Prints the version and exits

Flags:
      --cmdStdOut            Pass to print command output to stdout
  -f, --config string        config file to read from
  -h, --help                 help for backy
      --hostsConfig string   yaml hosts file to read from
      --logFile string       log file to write to
      --s3Endpoint string    Sets the S3 endpoint used for config file fetching. Overrides S3_ENDPOINT env variable.
  -v, --verbose              Sets verbose level

Use "backy [command] --help" for more information about a command.
```
 
# Subcommands

## backup

```
Backup executes commands defined in config file.
Use the --lists or -l flag to execute the specified lists. If not flag is not given, all lists will be executed.

Usage:
  backy backup [--lists=list1 --lists list2 ... | -l list1 -l list2 ...] [flags]

Flags:
  -h, --help                help for backup
  -l, --lists stringArray   Accepts comma-separated names of command lists to execute.

Global Flags:
      --cmdStdOut            Pass to print command output to stdout
  -f, --config string        config file to read from
      --hostsConfig string   yaml hosts file to read from
      --logFile string       log file to write to
      --s3Endpoint string    Sets the S3 endpoint used for config file fetching. Overrides S3_ENDPOINT env variable.
  -v, --verbose              Sets verbose level
```

## cron

```
Cron starts a scheduler that executes command lists at the time defined in config file.

Usage:
  backy cron [flags]

Flags:
  -h, --help   help for cron

Global Flags:
      --cmdStdOut            Pass to print command output to stdout
  -f, --config string        config file to read from
      --hostsConfig string   yaml hosts file to read from
      --logFile string       log file to write to
      --s3Endpoint string    Sets the S3 endpoint used for config file fetching. Overrides S3_ENDPOINT env variable.
  -v, --verbose              Sets verbose level
```

## exec

```
Exec executes commands defined in config file in order given.

Usage:
  backy exec command ... [flags]
  backy exec [command]

Available Commands:
  host        Runs command defined in config file on the hosts in order specified.
  hosts       Runs command defined in config file on the hosts in order specified.

Flags:
  -h, --help   help for exec

Global Flags:
      --cmdStdOut            Pass to print command output to stdout
  -f, --config string        config file to read from
      --hostsConfig string   yaml hosts file to read from
      --logFile string       log file to write to
      --s3Endpoint string    Sets the S3 endpoint used for config file fetching. Overrides S3_ENDPOINT env variable.
  -v, --verbose              Sets verbose level

Use "backy exec [command] --help" for more information about a command.
```

### exec host

```
Host executes specified commands on the hosts defined in config file.
Use the --commands or -c flag to choose the commands.

Usage:
  backy exec host [--command=command1 --command=command2 ... | -c command1 -c command2 ...] [--hosts=host1 --hosts=hosts2 ... | -m host1 -m host2 ...]  [flags]

Flags:
  -c, --command stringArray   Accepts space-separated names of commands. Specify multiple times for multiple commands.
  -h, --help                  help for host
  -m, --hosts stringArray     Accepts space-separated names of hosts. Specify multiple times for multiple hosts.

Global Flags:
      --cmdStdOut            Pass to print command output to stdout
  -f, --config string        config file to read from
      --hostsConfig string   yaml hosts file to read from
      --logFile string       log file to write to
      --s3Endpoint string    Sets the S3 endpoint used for config file fetching. Overrides S3_ENDPOINT env variable.
  -v, --verbose              Sets verbose level
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
      --cmdStdOut            Pass to print command output to stdout
  -f, --config string        config file to read from
      --hostsConfig string   yaml hosts file to read from
      --logFile string       log file to write to
      --s3Endpoint string    Sets the S3 endpoint used for config file fetching. Overrides S3_ENDPOINT env variable.
  -v, --verbose              Sets verbose level
```

## list

```
List commands, lists, or hosts defined in config file

Usage:
  backy list [command]

Available Commands:
  cmds        List commands defined in config file.
  lists       List lists defined in config file.

Flags:
  -h, --help   help for list

Global Flags:
      --cmdStdOut            Pass to print command output to stdout
  -f, --config string        config file to read from
      --hostsConfig string   yaml hosts file to read from
      --logFile string       log file to write to
      --s3Endpoint string    Sets the S3 endpoint used for config file fetching. Overrides S3_ENDPOINT env variable.
  -v, --verbose              Sets verbose level

Use "backy list [command] --help" for more information about a command.
```
## list cmds

```
List commands defined in config file

Usage:
  backy list cmds [cmd1 cmd2 cmd3...] [flags]

Flags:
  -h, --help   help for cmds

Global Flags:
      --cmdStdOut            Pass to print command output to stdout
  -f, --config string        config file to read from
      --hostsConfig string   yaml hosts file to read from
      --logFile string       log file to write to
      --s3Endpoint string    Sets the S3 endpoint used for config file fetching. Overrides S3_ENDPOINT env variable.
  -v, --verbose              Sets verbose level
```
## list lists

```
List lists defined in config file

Usage:
  backy list lists [list1 list2 ...] [flags]

Flags:
  -h, --help   help for lists

Global Flags:
      --cmdStdOut            Pass to print command output to stdout
  -f, --config string        config file to read from
      --hostsConfig string   yaml hosts file to read from
      --logFile string       log file to write to
      --s3Endpoint string    Sets the S3 endpoint used for config file fetching. Overrides S3_ENDPOINT env variable.
  -v, --verbose              Sets verbose level
```
