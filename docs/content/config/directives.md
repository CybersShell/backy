---
title: "External Directives"
weight: 2
description: How to set up external directives.
---

External directives are for including data that should not be in the config file. The following directives are supported:

- `%{file:path/to/file}%`
- `%{env:ENV_VAR}%`
- `%{vault:vault-key}%`

See the docs of each command if the field is supported.

If the file path does not begin with the root directory marker, usually `/`, the config file's directory will be used as the starting point.