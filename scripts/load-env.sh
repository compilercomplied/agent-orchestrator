#!/bin/bash

# This script loads environment variables from Pulumi into the current shell session.
# Usage: source scripts/load-env.sh [stack-name]

STACK=${1:-local}

if [ "$BASH_SOURCE" == "$0" ]; then
    echo "Error: This script must be sourced."
    echo "Usage: source scripts/load-env.sh [$STACK]"
    exit 1
fi

if ! command -v pulumi &> /dev/null; then
    echo "Error: pulumi CLI is not installed."
    return 1
fi

if ! command -v jq &> /dev/null; then
    echo "Error: jq is required but not installed."
    return 1
fi

if [ -z "$PULUMI_CONFIG_PASSPHRASE" ]; then
    echo "Warning: PULUMI_CONFIG_PASSPHRASE is not set. You might be prompted for it."
fi

echo "Loading configuration from Pulumi (stack: $STACK)..."

# Extract configuration from Pulumi and export it
# Handles both simple values and secret objects
pulumi config --stack "$STACK" -C iac --show-secrets --json | \
jq -r --arg q "'" '
    to_entries | .[] | 
    (.key | split(":") | last) + "=" + $q + 
    (
        if .value | type == "object" 
        then .value.value 
        else .value 
        end | tostring
    ) + $q
' > .env

if [ $? -ne 0 ]; then
    echo "Error: Failed to extract configuration from Pulumi."
    return 1
fi

# Load the .env file into the current shell
set -a
source .env
set +a

echo "Environment loaded successfully from stack: $STACK"
