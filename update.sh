#!/bin/bash
set -euo pipefail

# (
#   cd /Users/matt/Downloads
#   rm /usr/local/bin/desktop-cli
#   curl -L https://github.com/mj1618/desktop-cli/releases/download/latest/desktop-cli_darwin_arm64.tar.gz | tar xz
#   mv desktop-cli /usr/local/bin/
# )

cd "$(dirname "$0")"

# Embed git commit and build date so dev builds are identifiable
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS="-X github.com/mj1618/desktop-cli/internal/version.Commit=${COMMIT}"
LDFLAGS="${LDFLAGS} -X github.com/mj1618/desktop-cli/internal/version.BuildDate=${BUILD_DATE}"

echo "Building desktop-cli (commit: ${COMMIT}, date: ${BUILD_DATE})..."
rm -f desktop-cli

if ! go build -ldflags "${LDFLAGS}" -o desktop-cli .; then
  echo "ERROR: build failed, not replacing installed binary" >&2
  exit 1
fi

codesign --force --sign - ./desktop-cli
sudo rm -f /usr/local/bin/desktop-cli
sudo mv desktop-cli /usr/local/bin/

echo "Installed desktop-cli to /usr/local/bin/desktop-cli"
desktop-cli --version