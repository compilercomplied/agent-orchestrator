#!/bin/bash
set -e
set -o pipefail

# Ensure we are in the project root
cd "$(dirname "$0")/.."

source "$(dirname "$0")/load-env.sh" local

docker compose -f docker-compose.e2e.yaml up \
	--build --exit-code-from e2e-tests \
	--abort-on-container-exit
