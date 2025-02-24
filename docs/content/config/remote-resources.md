---
title: "Remote resources"
weight: 2
description: This is dedicated to configuring remote resources.
---

Remote resources can be used for a lot of things, including config files and scripts.

## Config file

For the main config file to be fetched remotely, pass the URL using `-f [url]`.

If using S3, you should use the s3 protocol URI: `s3://bucketName/key/path`. You will also need to set the env variable `S3_ENDPOINT` to the appropriate value. The flag `--s3-endpoint` can be used to override this value or to set this value, if not already set.

## Scripts

Remote script support is currently limited to http/https endpoints.