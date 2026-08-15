---
title: List
---


List commands, lists, or hosts defined in config file. The subcommands take zero or more arguments to print specific commands or lists

Usage:
```
  backy list [command]
```

Available Commands:
  cmds        Prints commands defined in config file.
  lists       Prints lists defined in config file.

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