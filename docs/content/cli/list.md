---
title: List
---


List commands, lists, or hosts defined in config file

Usage:
```
  backy list [command]
```

Available Commands:
  cmds        List commands defined in config file.
  lists       List lists defined in config file.

Flags:
```
  -h, --help   help for list
```

Global Flags:
```
      --cmdStdOut            Pass to print command output to stdout
  -f, --config string        config file to read from
      --log-file string      log file to write to
      --s3-endpoint string   Sets the S3 endpoint used for config file fetching. Overrides S3_ENDPOINT env variable.
  -v, --verbose              Sets verbose level
```