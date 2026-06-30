#!/usr/bin/env bash
# pzmod installer for Linux and macOS.
#
#   curl -fsSL https://pzmod.dev/install.sh | bash
#   curl -fsSL https://pzmod.dev/install.sh | bash -s -- ~/.local/bin/pzmod   # custom path
#
# Environment:
#   PZMOD_VERSION   install a specific release tag (e.g. v3.0.0); defaults to latest.
#
# On Windows, use install.ps1 instead:
#   irm https://pzmod.dev/install.ps1 | iex

set -euo pipefail

REPO="kldzj/pzmod"
DEFAULT_TARGET="/usr/local/bin/pzmod"

err()  { printf 'pzmod install: %s\n' "$*" >&2; exit 1; }
info() { printf '%s\n' "$*"; }

command -v curl >/dev/null 2>&1 || err "curl is required but was not found"

detect_os() {
  case "$(uname -s)" in
    Linux)  echo linux ;;
    Darwin) echo darwin ;;
    *)      err "unsupported OS '$(uname -s)' - on Windows use install.ps1" ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64 | amd64)  echo x86_64 ;;
    arm64 | aarch64) echo arm64 ;;
    i386 | i686)     echo i386 ;;
    *)               err "unsupported architecture '$(uname -m)'" ;;
  esac
}

latest_tag() {
  # Follow the /releases/latest redirect to read the tag; no API token needed.
  local url
  url="$(curl -fsSL -o /dev/null -w '%{url_effective}' "https://github.com/${REPO}/releases/latest")" || return 1
  case "$url" in
    */releases/tag/*) printf '%s\n' "${url##*/tag/}" ;;
    *)                return 1 ;;
  esac
}

sha256_of() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

main() {
  local target os arch tag asset base tmp sudo expected actual
  target="${1:-$DEFAULT_TARGET}"
  os="$(detect_os)"
  arch="$(detect_arch)"
  tag="${PZMOD_VERSION:-$(latest_tag || err "could not determine the latest release")}"
  asset="pzmod_${os}_${arch}"
  base="https://github.com/${REPO}/releases/download/${tag}"

  info "Installing pzmod ${tag} (${os}/${arch}) to ${target}"

  tmp="$(mktemp -d)"
  trap 'rm -rf "$tmp"' EXIT

  curl -fSL --proto '=https' --tlsv1.2 -o "$tmp/pzmod" "${base}/${asset}" \
    || err "no prebuilt binary for ${asset} in ${tag} (see ${base})"

  # Verify against the release checksums when we can compute a digest.
  if curl -fsSL -o "$tmp/checksums.txt" "${base}/checksums.txt" 2>/dev/null; then
    actual="$(sha256_of "$tmp/pzmod")"
    expected="$(awk -v f="$asset" '$2 == f {print $1}' "$tmp/checksums.txt")"
    if [ -n "$actual" ] && [ -n "$expected" ]; then
      [ "$actual" = "$expected" ] || err "checksum mismatch for ${asset} (expected ${expected}, got ${actual})"
      info "Checksum verified."
    else
      info "Warning: could not verify checksum (no sha256 tool or entry); continuing."
    fi
  else
    info "Warning: checksums.txt unavailable; skipping verification."
  fi

  chmod +x "$tmp/pzmod"

  local dir
  dir="$(dirname "$target")"
  sudo=""
  if [ -d "$dir" ]; then
    [ -w "$dir" ] || sudo="sudo"
  else
    mkdir -p "$dir" 2>/dev/null || sudo="sudo"
  fi
  if [ -n "$sudo" ]; then
    command -v sudo >/dev/null 2>&1 \
      || err "cannot write to ${dir}; re-run as root or choose a writable path, e.g. bash -s -- ~/.local/bin/pzmod"
    info "Elevated permissions are required to write to ${dir}."
  fi

  $sudo mkdir -p "$dir"
  $sudo mv "$tmp/pzmod" "$target"
  $sudo chmod +x "$target"

  info "pzmod installed to ${target}"
  case ":${PATH}:" in
    *":${dir}:"*) info "Run 'pzmod' to get started." ;;
    *) info "Note: ${dir} is not on your PATH - add it, or run ${target} directly." ;;
  esac
}

main "$@"
