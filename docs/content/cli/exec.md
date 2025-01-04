---
title: Exec
---

The `exec` subcommand can do some things that the configuration file can't do yet. The command `exec host` can execute commands on many hosts.

`exec host` takes the following arguments:

```sh
  -c, --commands strings   Accepts comma-separated names of commands.
  -h, --help               help for host
  -m, --hosts strings      Accepts comma-separated names of hosts.
```

The commands have to be defined in the config file. The hosts need to at least be in the ssh_config(5) file.

```sh
backy exec host [--commands=command1,command2, ... | -c command1,command2, ...] [--hosts=host1,hosts2, ... | -m host1,host2, ...]  [flags]
```
