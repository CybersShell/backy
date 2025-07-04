---
title: "Hosts"
weight: 2
description: >
  This page tells you how to use hosts.
---

| Key                  | Description                                                   | Type     | Required | External directive support |
|----------------------|---------------------------------------------------------------|----------|----------|----------------------------|
| `OS`                 | Operating system of the host (used for package commands)      | `string` | no       | No                         |
| `config`             | Path to the SSH config file                                   | `string` | no       | No                         |
| `host`               | Specifies the `Host` ssh_config(5) directive                  | `string` | yes      | No                         |
| `hostname`           | Hostname of the host                                          | `string` | no       | No                         |
| `knownHostsFile`     | Path to the known hosts file                                  | `string` | no       | No                         |
| `port`               | Port number to connect to                                     | `uint16` | no       | No                         |
| `proxyjump`          | Proxy jump hosts, comma-separated                             | `string` | no       | No                         |
| `password`           | Password for SSH authentication                               | `string` | no       | No                         |
| `privateKeyPath`     | Path to the private key file                                  | `string` | no       | No                         |
| `privateKeyPassword` | Password for the private key file                             | `string` | no       | Yes                        |
| `user`               | Username for SSH authentication                               | `string` | no       | No                         |

## exec host subcommand

Backy has a subcommand `exec host`. This subcommand takes the flags of `-m host1 -m host2`. For now these hosts need to be defined in the config file.
