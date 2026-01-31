#!/usr/bin/env bash
set -euo pipefail

DIR="$(cd "$(dirname "$0")" && pwd)"
echo "Stopping and removing SSH test container..."
docker compose -f "$DIR/compose.yml" down --remove-orphans
echo "Stopped."