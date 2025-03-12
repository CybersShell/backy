---
title: "Vault"
weight: 4
description: Set up and configure vault.
---

[Vault](https://www.vaultproject.io/) is a tool for storing secrets and other data securely.

A Vault key can be used by prefixing `%{vault:vault.keys.name}%` in a field that supports external directives.

This is the object in the config file:

```yaml
vault:
  token: hvs.tXqcASvTP8wg92f7riyvGyuf
  address: http://127.0.0.1:8200
  enabled: false
  keys:
    - name: mongourl
      mountpath: secret
      key: data
      path: mongo/url
      type:  # KVv1 or KVv2
    - name: someKeyName
      mountpath: secret
      key: keyData
      type: KVv2
      path: some/path
```
