#!/usr/bin/env bash
#
# dmon installer — downloads a released binary from GitHub and installs it
# cleanly into a per-user, versioned location with a stable symlink.
#
#   # latest release:
#   curl -fsSL https://github.com/ElyessBenSassi/dmon-docker-MadeEz/releases/latest/download/install.sh | bash
#
#   # pin a version:
#   curl -fsSL https://github.com/ElyessBenSassi/dmon-docker-MadeEz/releases/download/v0.2.0/install.sh | DMON_VERSION=v0.2.0 bash
#
# Layout:
#   ~/.local/share/dmon/<version>/dmon      (binary)
#   ~/.local/bin/dmon -> ../share/dmon/...  (stable path on PATH)
#
# Env:
#   DMON_VERSION   release tag to install (default: latest)
#   DMON_PREFIX    install root (default: ~/.local)
set -euo pipefail

REPO="ElyessBenSassi/dmon-docker-MadeEz"
PREFIX="${DMON_PREFIX:-$HOME/.local}"
BINDIR="$PREFIX/bin"

# --- detect platform ---------------------------------------------------------
os="$(uname -s)"
arch="$(uname -m)"
case "$os" in
  Linux)  os=linux  ;;
  Darwin) os=darwin ;;
  *) echo "Unsupported OS: $os" >&2; exit 1 ;;
esac
case "$arch" in
  x86_64 | amd64)  arch=amd64 ;;
  aarch64 | arm64) arch=arm64 ;;
  *) echo "Unsupported architecture: $arch" >&2; exit 1 ;;
esac

# --- resolve version tag -----------------------------------------------------
tag="${DMON_VERSION:-latest}"
if [[ "$tag" == "latest" ]]; then
  # Follow the /releases/latest redirect to discover the concrete tag.
  final="$(curl -fsSL -o /dev/null -w '%{url_effective}' "https://github.com/$REPO/releases/latest")"
  tag="${final##*/tag/}"
  if [[ -z "$tag" || "$tag" == "$final" ]]; then
    echo "Could not resolve the latest release tag." >&2
    exit 1
  fi
fi
ver="${tag#v}"   # strip leading v for asset names

base="https://github.com/$REPO/releases/download/$tag"
asset="dmon_${ver}_${os}_${arch}.tar.gz"

# --- download ----------------------------------------------------------------
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

echo "Downloading dmon $tag ($os/$arch)..."
curl -fSL "$base/$asset" -o "$tmp/$asset" \
  || { echo "No release asset for $os/$arch at $base/$asset" >&2; exit 1; }

# --- verify checksum ---------------------------------------------------------
if curl -fsSL "$base/dmon_${ver}_checksums.txt" -o "$tmp/checksums.txt" 2>/dev/null; then
  sum_tool=""
  command -v sha256sum >/dev/null 2>&1 && sum_tool="sha256sum"
  [[ -z "$sum_tool" ]] && command -v shasum >/dev/null 2>&1 && sum_tool="shasum -a 256"
  if [[ -n "$sum_tool" ]]; then
    ( cd "$tmp" && grep " ${asset}\$" checksums.txt | $sum_tool -c - ) >/dev/null \
      || { echo "Checksum verification failed for $asset" >&2; exit 1; }
    echo "Checksum verified."
  fi
fi

# --- install -----------------------------------------------------------------
LIBDIR="$PREFIX/share/dmon/$ver"
mkdir -p "$LIBDIR" "$BINDIR"
tar -xzf "$tmp/$asset" -C "$tmp" dmon
install -m 0755 "$tmp/dmon" "$LIBDIR/dmon"
ln -sfn "$LIBDIR/dmon" "$BINDIR/dmon"

echo "Installed dmon $ver"
echo "  binary:  $LIBDIR/dmon"
echo "  symlink: $BINDIR/dmon -> $LIBDIR/dmon"

case ":$PATH:" in
  *":$BINDIR:"*) ;;
  *)
    echo
    echo "note: $BINDIR is not on your PATH. Add this to your shell profile:"
    echo "  export PATH=\"$BINDIR:\$PATH\""
    ;;
esac
