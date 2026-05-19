#!/bin/bash
set -e

# Proxy to the downloaded binary
exec "$HOME/.local/bin/agentboard" "$@"