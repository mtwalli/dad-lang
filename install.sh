#!/bin/sh
# Ḍād (ضاد) installer for macOS and Linux.
#
#   curl -fsSL https://raw.githubusercontent.com/mtwalli/dad-lang/main/install.sh | sh
#
# Options (environment variables):
#   DAD_VERSION      release tag to install, e.g. v0.1.0 (default: latest)
#   DAD_INSTALL_DIR  where to put the binaries (default: $HOME/.dad/bin)
#   DAD_DOWNLOAD_BASE  download from this URL instead of GitHub Releases (mirrors, testing)
set -eu

REPO="mtwalli/dad-lang"
VERSION="${DAD_VERSION:-latest}"
INSTALL_DIR="${DAD_INSTALL_DIR:-$HOME/.dad/bin}"

say() { printf '%s\n' "$*"; }
fail() { printf 'error: %s\n' "$*" >&2; exit 1; }

case "$(uname -s)" in
  Darwin) os="darwin" ;;
  Linux) os="linux" ;;
  *) fail "unsupported system $(uname -s). On Windows, use install.ps1." ;;
esac

case "$(uname -m)" in
  x86_64 | amd64) arch="amd64" ;;
  arm64 | aarch64) arch="arm64" ;;
  *) fail "unsupported processor $(uname -m)." ;;
esac

name="dad_${os}_${arch}"
if [ -n "${DAD_DOWNLOAD_BASE:-}" ]; then
  base="$DAD_DOWNLOAD_BASE"
elif [ "$VERSION" = "latest" ]; then
  base="https://github.com/${REPO}/releases/latest/download"
else
  base="https://github.com/${REPO}/releases/download/${VERSION}"
fi

if command -v curl >/dev/null 2>&1; then
  fetch() { curl -fsSL "$1" -o "$2"; }
elif command -v wget >/dev/null 2>&1; then
  fetch() { wget -q "$1" -O "$2"; }
else
  fail "curl or wget is required."
fi

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT INT TERM

say "Downloading Ḍād ($VERSION) for $os/$arch…"
fetch "$base/$name.tar.gz" "$tmp/$name.tar.gz" || fail "download failed: $base/$name.tar.gz"
fetch "$base/checksums.txt" "$tmp/checksums.txt" || fail "download failed: $base/checksums.txt"

expected="$(grep " $name.tar.gz\$" "$tmp/checksums.txt" | cut -d ' ' -f 1)"
[ -n "$expected" ] || fail "no checksum listed for $name.tar.gz"
if command -v sha256sum >/dev/null 2>&1; then
  actual="$(sha256sum "$tmp/$name.tar.gz" | cut -d ' ' -f 1)"
else
  actual="$(shasum -a 256 "$tmp/$name.tar.gz" | cut -d ' ' -f 1)"
fi
[ "$expected" = "$actual" ] || fail "checksum mismatch for $name.tar.gz"

tar -xzf "$tmp/$name.tar.gz" -C "$tmp"
mkdir -p "$INSTALL_DIR"
cp "$tmp/$name/dad" "$INSTALL_DIR/dad"
chmod 755 "$INSTALL_DIR/dad"
ln -sf "$INSTALL_DIR/dad" "$INSTALL_DIR/ضاد"

say "Installed: $INSTALL_DIR/dad (also available as ضاد)"
"$INSTALL_DIR/dad" version

case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *)
    case "${SHELL:-}" in
      */zsh) profile="$HOME/.zshrc" ;;
      */bash) profile="$HOME/.bashrc" ;;
      */fish) profile="" ;;
      *) profile="$HOME/.profile" ;;
    esac
    say ""
    if [ -n "$profile" ]; then
      say "Add Ḍād to your PATH by running:"
      say "  echo 'export PATH=\"$INSTALL_DIR:\$PATH\"' >> $profile && . $profile"
    else
      say "Add Ḍād to your PATH by running:"
      say "  fish_add_path $INSTALL_DIR"
    fi
    ;;
esac
