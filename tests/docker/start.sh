#!/usr/bin/env bash
set -euo pipefail

DIR="$(cd "$(dirname "$0")" && pwd)"
echo "Starting SSH test container (building if needed)..."
docker compose -f "$DIR/compose.yml" up -d --build
ssh-keyscan -p 2222 localhost > "$DIR/known_hosts"
echo "Container started on localhost:2222"