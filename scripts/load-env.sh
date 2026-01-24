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
eval "$(
    pulumi config --stack "$STACK" -C iac --show-secrets --json | \
    jq -r '
        to_entries | .[] | 
        "export " + (.key | split(":") | last) + "='
'" + 
        (
            if .value | type == "object" 
            then .value.value 
            else .value 
            end | tostring
        ) + "'\n'"
    ')"

echo "Environment loaded successfully from stack: $STACK"
