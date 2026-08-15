---
title: Exec
---

~~The `exec` subcommand can do some things that the configuration file can't do yet. The command `exec host` can execute commands on many hosts.~~

{{% notice info %}}
  For now only the `exec host` sub-command is implemented. More to come.
{{% /notice %}}

{{% notice warning %}}
  The config file's `hosts` are overridden by this command. Hooks are not altered. For consistent results, create a local config file with only the hosts you want to use.
{{% /notice %}}


`exec host` takes the following arguments:

```sh
  -c, --command stringArray   Accepts space-separated names of commands. Specify multiple times for multiple commands.
  -h, --help                  help for host
  -m, --hosts stringArray     Accepts space-separated names of hosts. Specify multiple times for multiple hosts.
```

The commands have to be defined in the config file. The hosts need to at least be in the ssh_config(5) file.

```sh
backy exec host [--commands=command1 -commands=command2 ... | -c command1 -c command2 ...] [--hosts=host1 --hosts=hosts2 ... | -m host1 -m host2 ...]  [flags]
```
