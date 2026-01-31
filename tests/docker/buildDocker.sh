#!/usr/bin/env bash
set -euo pipefail

# Build and run the test SSH container from the tests/docker directory
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

docker container rm -f ssh_server_container 2>/dev/null || true
docker build -t ssh_server_image "$SCRIPT_DIR"
docker run -d -p 2222:22 --name ssh_server_container ssh_server_image
sleep 5
ssh-keyscan -p 2222 localhost > $SCRIPT_DIR/known_hosts