#!/bin/bash
set -e

# Ensure we are in the project root
cd "$(dirname "$0")/.."

# Check for dependencies
if ! command -v jq &> /dev/null;
then
    echo "Error: jq is required but not installed."
    exit 1
fi

echo "Loading configuration from Pulumi..."

eval $(pulumi config -C iac --show-secrets --json | jq -r '
  to_entries | 
  .[] | 
  "export " + (.key | split(":") | last) + "='\''" + (if .value | type == "object" then .value.value else .value end | tostring) + "'\''"
')

echo "Configuration loaded. Starting E2E tests..."

docker compose -f docker-compose.e2e.yaml up --build --exit-code-from e2e-tests --abort-on-container-exit
