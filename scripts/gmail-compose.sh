#!/bin/bash
# Gmail Compose & Send Helper Script
#
# Quick way for agents to send emails via desktop-cli MCP server
#
# Usage:
#   scripts/gmail-compose.sh "recipient@example.com" "Subject" "Body text"
#   scripts/gmail-compose.sh "matt@example.com" "Test" "This is a test"
#
# Environment variables can override defaults:
#   TO="user@example.com" SUBJECT="Title" BODY="Message" scripts/gmail-compose.sh
#
# Returns YAML output from desktop-cli, parseable by agents

set -e

TO="${1:-${TO:-matt@supplywise.app}}"
SUBJECT="${2:-${SUBJECT:-Test Email}}"
BODY="${3:-${BODY:-Test message from desktop-cli}}"

# Validate inputs
if [ -z "$TO" ]; then
  echo "Error: TO address required" >&2
  exit 1
fi

# Generate YAML with substituted variables
YAML=$(cat <<'EOF'
- open: { url: "https://gmail.com" }
- wait: { for-text: "Compose", timeout: 10 }
- click: { text: "Compose" }
- sleep: { ms: 800 }
- fill:
    fields:
      - label: "To"
        value: "__TO__"
      - label: "Subject"
        value: "__SUBJECT__"
      - label: "Body"
        value: "__BODY__"
    submit: "Send"
- sleep: { ms: 2000 }
EOF
)

# Replace placeholders with actual values
YAML="${YAML//__TO__/$TO}"
YAML="${YAML//__SUBJECT__/$SUBJECT}"
YAML="${YAML//__BODY__/$BODY}"

# Execute via desktop-cli MCP server
desktop-cli do --app "Google Chrome" <<EOYAML
$YAML
EOYAML
