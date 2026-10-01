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

# A bar while the binary comes down, when someone is watching: the script
# itself arrives on stdin, so stderr is what says whether this is a terminal.
if [ -t 2 ]; then
	progress=--progress-bar
	# curl measures the terminal on stdin, finds the pipe and assumes 79
	# columns: in a narrower window every redraw wraps and leaves a line
	# behind. COLUMNS is what it reads first.
	COLUMNS="$(stty size <&2 2>/dev/null | cut -d' ' -f2)"
	export COLUMNS
else
	progress=--silent
fi

echo "downloading $binary"
curl -fSL $progress "$url/$binary" -o "$tmp/homewend" || fail "download failed: $url/$binary"
curl -fsSL "$url/homewend_checksums.txt" -o "$tmp/checksums.txt" || fail "download failed: $url/homewend_checksums.txt"

want="$(grep " $binary\$" "$tmp/checksums.txt" | cut -d' ' -f1)"
[ -n "$want" ] || fail "$binary is not in the release's checksums"
[ "$(sha256 "$tmp/homewend")" = "$want" ] || fail "checksum mismatch: the download is damaged, nothing was installed"

mkdir -p "$dir"
chmod 755 "$tmp/homewend"
mv "$tmp/homewend" "$dir/homewend"
# What to do next, in full: whoever piped this into sh has nothing else to
# read yet.
case ":$PATH:" in
*":$dir:"*) run=homewend ;;
*) run="$dir/homewend" ;;
esac
# The commands to type in homewend.app's blue, when this is a terminal.
if [ -t 1 ]; then
	on="$(printf '\033[38;2;122;162;247m')" off="$(printf '\033[0m')"
else
	on="" off=""
fi
cat <<EOF

homewend $("$dir/homewend" version) is installed in $dir.

Next, sign in to Google, once. A small window opens; your password goes to
Google only:

  ${on}$run login${off}

Then bring your photos home, here one year of them:

  ${on}$run get --year 2025 --library ~/Pictures/Homewend${off}

All the commands: ${on}$run help${off}
EOF
case ":$PATH:" in
*":$dir:"*) ;;
*) echo "($dir is not in your PATH: add it to type just \"homewend\".)" ;;
esac
