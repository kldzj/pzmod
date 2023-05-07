#!/bin/bash

get_arch() {
    local raw_arch="$(uname -m)"
    local arch

    case "$raw_arch" in
    x86_64)
        arch="x86_64"
        ;;
    i386 | i686)
        arch="i386"
        ;;
    aarch64)
        arch="arm64"
        ;;
    *)
        echo "Unsupported architecture: $raw_arch"
        exit 1
        ;;
    esac

    echo "$arch"
}

use_sudo() {
    if command -v sudo >/dev/null 2>&1 && [ "$(id -u)" -ne 0 ]; then
        echo "yes"
    else
        echo "no"
    fi
}

download_pzmod() {
    local latest_url="https://api.github.com/repos/kldzj/pzmod/releases/latest"
    local platform="$(uname -s | tr '[:upper:]' '[:lower:]')"
    local arch="$(get_arch)"
    local target="${1:-/usr/local/bin/pzmod}"
    local sudo_needed=$(use_sudo)

    local download_url=$(curl -s "$latest_url" |
        grep "browser_download_url" |
        grep "${platform}_${arch}" |
        cut -d '"' -f 4)

    if [ -z "$download_url" ]; then
        echo "Error: No prebuilt pzmod binary available for ${platform}_${arch}."
        exit 1
    fi

    if [ "$sudo_needed" = "yes" ]; then
        sudo wget -O "$target" "$download_url" &&
            sudo chmod +x "$target" &&
            echo "pzmod successfully downloaded to $target"
    else
        wget -O "$target" "$download_url" &&
            chmod +x "$target" &&
            echo "pzmod successfully downloaded to $target"
    fi
}

download_pzmod "$@"
