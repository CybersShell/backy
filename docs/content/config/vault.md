---
title: "Vault"
weight: 4
---

[Vault](https://www.vaultproject.io/) is a tool for storing secrets and other data securely.

Vault config can be used by prefixing `vault:` in front of a password or ENV var.

This is the object in the config file:

```yaml
vault:
  token: hvs.tXqcASvTP8wg92f7riyvGyuf
  address: http://127.0.0.1:8200
  enabled: false
  keys:
    - name: mongourl
      mountpath: secret
      path: mongo/url
      type:  # KVv1 or KVv2
    - name:
      path:
      type:
      mountpath:
```
