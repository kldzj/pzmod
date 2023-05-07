#!/bin/bash

download_pzmod() {
  local latest_url="https://api.github.com/repos/kldzj/pzmod/releases/latest"
  local platform="$(uname -s | tr '[:upper:]' '[:lower:]')"
  local arch="$(uname -m)"
  local target="${1:-/usr/local/bin/pzmod}"

  local download_url=$(curl -s "$latest_url" \
    | grep "browser_download_url" \
    | grep "${platform}_${arch}" \
    | cut -d '"' -f 4)

  sudo wget -O "$target" "$download_url" \
    && sudo chmod +x "$target" \
    && echo "pzmod successfully downloaded to $target"
}

download_pzmod "$@"