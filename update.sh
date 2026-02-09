#!/bin/bash

(
  cd /Users/matt/Downloads
  rm /usr/local/bin/desktop-cli
  curl -L https://github.com/mj1618/desktop-cli/releases/download/latest/desktop-cli_darwin_arm64.tar.gz | tar xz
  mv desktop-cli /usr/local/bin/
)
