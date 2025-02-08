---
title: "Command Lists"
weight: 2
description: >
  This page tells you how to get use command lists.
---

Command lists are for executing commands in sequence and getting notifications from them.

The top-level object key can be anything you want but not the same as another.

Lists can go in a separate file. Command lists should be in a separate file if:

1. key 'cmd-lists.file' is specified
2. lists.yml or lists.yaml is found in the same directory as the backy config file

{{% notice info %}}
The lists file is also checked in remote resources. 

The lists file is ignored under the following condition:

If a remote config file is specified (on the command-line using `-f`) and the lists file is not found in the same directory, the lists file is assumed to not exist.
{{% /notice %}}

```yaml {lineNos="true" wrap="true" title="yaml"}
  test2:
    name: test2
    order:
      - test
      - test2
    notifications:
      - mail.prod-email
      - matrix.sysadmin
    cron: "0 * * * * *"
```

| key | description | type | required
| --- | --- | --- | --- |
| `order` | Defines the sequence of commands to execute | `[]string` | yes |
| `getOutput` | Command(s) output is in the notification(s) | `bool` | no |
| `notifications` | The notification service(s) and ID(s) to use on success and failure. Must be *`service.id`*. See the [notifications documentation page](/config/notifications/) for more | `[]string` | no |
| `name` | Optional name of the list | `string` | no |
| `cron` | Time at which to schedule the list. Only has affect when cron subcommand is run. | `string` | no |

### Order

The order is an array of commands to execute in order. Each command must be defined.

```yaml
order:
  - cmd-1
  - cmd-2
```

### getOutput

Get command output when a notification is sent.

Is not required. Can be `true` or `false`. Default is `false`.

### Notifications

An array of notification IDs to use on success and failure. Must match any of the `notifications` object map keys.

### Name

Name is optional. If name is not defined, name will be the object's map key.

### Cron mode

Backy also has a cron mode, so one can run `backy cron` and start a process that schedules jobs to run at times defined in the configuration file.

Adding `cron: 0 0 1 * * *` to a `cmd-lists` object will schedule the list at 1 in the morning. See [https://crontab.guru/](https://crontab.guru/) for reference.

{{% notice tip %}}
Note: Backy uses the second field of cron, so add anything except `*` to the beginning of a regular cron expression.
{{% /notice %}}

```yaml {lineNos="true" wrap="true" title="yaml"}
cmd-lists:
  docker-container-backup: # this can be any name you want
    # all commands have to be defined
    order:
      - stop-docker-container
      - backup-docker-container-script
      - shell-cmd
      - hostname
      - start-docker-container
    notifications:
      - matrix.id
    name: backup-some-container
    cron: "0 0 1 * * *"
  hostname:
    name: hostname
    order:
      - hostname
    notifications:
      - mail.prod-email
```
