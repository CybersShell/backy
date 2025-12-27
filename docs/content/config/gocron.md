---
title: "Configuring Cron"
weight: 3
description: >
  Use Cron to run lists at a specified time.
---

Backy provides an easy-to-use way to execute commands at a specified time.

Adding `cron: 0 0 1 * * *` to a `cmdLists` object will schedule the list at 1 in the morning. See [https://crontab.guru/](https://crontab.guru/) for reference.

{{% notice tip %}}
Note: Backy uses the second field of cron, so add anything except `*` to the beginning of a regular cron expression.
{{% /notice %}}

GoCron allows one to configure a server to view the jobs in the scheduler. See [GoCron UI GitHub](https://github.com/go-co-op/gocron-ui).
GoCron can be configured or left alone for defaults.

GoCron configuration:

| key | description | type | required | default
| --- | --- | --- | --- | ---
| `bindAddress` | Interface's IP to bind to. Must not contain port. | `string` | no | `:port`
| `port` | Port to use. | `int` | no | `8888`
| `useSeconds` | Whether to parse the second cron field. | `bool` | no | `false`


```yaml {lineNos="true" wrap="true" title="yaml"}
goCron:
  bindAddress: "0.0.0.0"
  port: 8888
  useSeconds: true

cmdLists:
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