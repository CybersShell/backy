---
title: "Hosts"
weight: 2
description: >
  This page tells you how to use hosts.
---

| Key                  | Description                                                   | Type     | Required |
|----------------------|---------------------------------------------------------------|----------|----------|
| `OS`                 | Operating system of the host (used for package commands)      | `string` | no       |
| `config`             | Path to the SSH config file                                   | `string` | no       |
| `host`               | Specifies the `Host` ssh_config(5) directive                  | `string` | yes      |
| `hostname`           | Hostname of the host                                          | `string` | no       |
| `knownhostsfile`     | Path to the known hosts file                                  | `string` | no       |
| `port`               | Port number to connect to                                     | `uint16` | no       |
| `proxyjump`          | Proxy jump hosts, comma-separated                             | `string` | no       |
| `password`           | Password for SSH authentication                               | `string` | no       |
| `privatekeypath`     | Path to the private key file                                  | `string` | no       |
| `privatekeypassword` | Password for the private key file                             | `string` | no       |
| `user`               | Username for SSH authentication                               | `string` | no       |

## exec host subcommand

Backy has a subcommand `exec host`. This subcommand takes the flags of `-m host1 -m host2`. For now these hosts need to be defined in the config file.