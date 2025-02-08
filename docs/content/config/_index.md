---
title: "Configuring Backy"
weight: 3
description: >
  This page tells you how to configure Backy.
---

This is the section on the config file.

To use a specific file:
```backy [command] -f /path/to/file```

You can also use a remote file:
```
backy [command] -f `s3/http source`
```

See remote resources docs for specific info.

If you leave the config path blank, the following paths will be searched in order:

1. `./backy.yml`
2. `./backy.yaml`
3. The same two files above contained in a `backy` subdirectory under in what is returned by Go's `os` package function `UserConfigDir()`.

{{% expand title="`UserConfigDir()` documentation:" %}}

Up-to date documentation for this function may be found on [GoDoc](https://pkg.go.dev/os#UserConfigDir).

>UserConfigDir returns the default root directory to use for user-specific configuration data. Users should create their own application-specific subdirectory within this one and use that.

>On Unix systems, it returns $XDG_CONFIG_HOME as specified by https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html if non-empty, else $HOME/.config. On Darwin, it returns $HOME/Library/Application Support. On Windows, it returns %AppData%. On Plan 9, it returns $home/lib.

>If the location cannot be determined (for example, $HOME is not defined), then it will return an error.

{{% /expand %}}

See the rest of the documentation, titles included below, in this section to configure it.

{{% children description="true" %}}