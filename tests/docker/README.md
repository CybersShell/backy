SSH test container
==================

This folder contains a simple Docker-based SSH server used for integration tests.

Quick start
-----------

Start the container (builds image if needed):

```bash
./start.sh
```

Stop the container:

```bash
./stop.sh
```

Access
------

- SSH endpoint: `localhost:2222`
- Test user: `backy` with password `backy` (password auth enabled)
- Root user: `root` with password `test`
- Public key `backytest.pub` is installed for both `backy` and `root`

Running tests
-------------

1. Start the container (`./start.sh`).
2. From the repo root, run your tests (example):

```bash
GO_TEST_SSH_ADDR=localhost:2222 go test ./... -v
```

If your tests rely on an SSH private key, use `tests/docker/backytest` as the private key and restrict access appropriately.
