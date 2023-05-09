#!/bin/bash

set -e

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
    local target="$1"
    local sudo="no"

    if ! command -v sudo >/dev/null; then
        echo "$sudo"
        return
    fi

    if [ -w "$(dirname "$target")" ]; then
        if [ -f "$target" ] && [ ! -w "$target" ]; then
            sudo="yes"
        fi
    else
        sudo="yes"
    fi

    echo "$sudo"
}

pzmod_in_path() {
    local filename="$1"
    local in_path="no"

    if command -v "$filename" >/dev/null; then
        in_path="yes"
    fi

    echo "$in_path"
}

finalize_installation() {
    local sudo_needed="$1"
    local target="${2:-/usr/local/bin/pzmod}"
    local filename="$(basename "$target")"
    local in_path=$(pzmod_in_path "$filename")

    if [ "$sudo_needed" = "yes" ]; then
        sudo chmod +x "$target"
    else
        chmod +x "$target"
    fi

    if [ "$in_path" = "no" ]; then
        echo "Warning: $filename not found in PATH."
        echo "You may want to add the directory to your PATH environment variable,"
        echo "or move the executable to a directory that's already in your PATH."
        echo
    fi

    echo "pzmod successfully installed to $target"
}

download_pzmod() {
    local latest_url="https://api.github.com/repos/kldzj/pzmod/releases/latest"
    local platform="$(uname -s | tr '[:upper:]' '[:lower:]')"
    local arch="$(get_arch)"
    local target="${1:-/usr/local/bin/pzmod}"
    local sudo_needed=$(use_sudo "$target")

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
            finalize_installation "$sudo_needed" "$target"
    else
        wget -O "$target" "$download_url" &&
            finalize_installation "$sudo_needed" "$target"
    fi
}

download_pzmod "$@"
