#!/bin/sh
# homewend — Copyright (C) 2026 Ron Alder
# SPDX-License-Identifier: AGPL-3.0-or-later

# Installs the latest homewend release for this machine:
#
#   curl -fsSL https://homewend.app/install.sh | sh
#
# The binary goes to ~/.local/bin, or to HOMEWEND_INSTALL_DIR if set. Nothing
# needs root, and the download is checked against the release's checksums
# before it is installed.
set -eu

repo="ronalder100/homewend"
dir="${HOMEWEND_INSTALL_DIR:-$HOME/.local/bin}"

fail() {
	echo "homewend: $*" >&2
	exit 1
}

case "$(uname -s)" in
Linux) os=linux ;;
Darwin) os=darwin ;;
*) fail "no build for $(uname -s) yet: Mac and Linux only" ;;
esac
case "$(uname -m)" in
x86_64 | amd64) arch=amd64 ;;
arm64 | aarch64) arch=arm64 ;;
*) fail "no build for $(uname -m) yet: amd64 and arm64 only" ;;
esac

if command -v sha256sum >/dev/null 2>&1; then
	sha256() { sha256sum "$1" | cut -d' ' -f1; }
else
	sha256() { shasum -a 256 "$1" | cut -d' ' -f1; }
fi

binary="homewend_${os}_${arch}"
url="https://github.com/$repo/releases/latest/download"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

echo "downloading $binary"
curl -fsSL "$url/$binary" -o "$tmp/homewend" || fail "download failed: $url/$binary"
curl -fsSL "$url/homewend_checksums.txt" -o "$tmp/checksums.txt" || fail "download failed: $url/homewend_checksums.txt"

want="$(grep " $binary\$" "$tmp/checksums.txt" | cut -d' ' -f1)"
[ -n "$want" ] || fail "$binary is not in the release's checksums"
[ "$(sha256 "$tmp/homewend")" = "$want" ] || fail "checksum mismatch: the download is damaged, nothing was installed"

mkdir -p "$dir"
chmod 755 "$tmp/homewend"
mv "$tmp/homewend" "$dir/homewend"
echo "installed $("$dir/homewend" version) to $dir/homewend"

case ":$PATH:" in
*":$dir:"*) echo "next: homewend login" ;;
*) echo "$dir is not in your PATH: add it, or run $dir/homewend login" ;;
esac
