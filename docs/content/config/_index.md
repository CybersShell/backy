---
title: "Configuring Backy"
weight: 3
description: >
  This page tells you how to configure Backy.
---

This is the section on the config file.

To use a specific file use the `-f` CLI flag:
```backy [command] -f /path/to/file```

If you leave the config path blank, the following paths will be searched in order:

1. `./backy.yml`
2. `./backy.yaml`
3. `~/.config/backy.yml`
4. `~/.config/backy.yaml`

Create a file at `~/.config/backy.yml`.

See the rest of the documentation in this section to configure it.
