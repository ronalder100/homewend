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

# The script itself arrives on stdin, so stderr is what says whether this is
# a terminal and how wide it is. Everything below is drawn and wrapped to that
# width: a line longer than the window wraps wherever the terminal likes.
cols=0
if [ -t 2 ]; then
	cols="$(stty size <&2 2>/dev/null | cut -d' ' -f2)"
fi
[ "${cols:-0}" -gt 0 ] || cols=80

# homewend.app's blue, and a dark slate for the part of the bar still to
# come: the terminal's own dim is a light grey that competes with the blue.
blue="$(printf '\033[38;2;122;162;247m')" dim="$(printf '\033[38;2;59;66;97m')" off="$(printf '\033[0m')"
case "${LC_ALL:-${LC_CTYPE:-${LANG:-}}}" in
*[Uu][Tt][Ff]-8* | *[Uu][Tt][Ff]8*) cell_done="━" cell_left="━" ;;
*) cell_done="#" cell_left="-" ;;
esac

mb() {
	echo "$(($1 / 1048576)).$(($1 % 1048576 * 10 / 1048576))"
}

# One line of progress. curl's own bar cannot be styled, so this one is drawn
# from the size in the response headers and the size of the file so far.
bar() {
	total=0 have=0
	if [ -f "$tmp/headers" ]; then
		# The redirects on the way announce a length of 0; the last one counts.
		total="$(tr -d '\r' <"$tmp/headers" | awk 'tolower($1) == "content-length:" { n = $2 } END { print n + 0 }')"
	fi
	if [ -f "$tmp/homewend" ]; then
		have="$(($(wc -c <"$tmp/homewend")))"
	fi
	cells=$((cols > 64 ? 40 : cols - 24))
	fill=$((total > 0 ? have * cells / total : 0))
	full="" rest="" i=0
	while [ "$i" -lt "$cells" ]; do
		if [ "$i" -lt "$fill" ]; then full="$full$cell_done"; else rest="$rest$cell_left"; fi
		i=$((i + 1))
	done
	printf '\r%s%s%s%s%s %3d%%  %s / %s MB\033[K' "$blue" "$full" "$dim" "$rest" "$off" \
		"$((total > 0 ? have * 100 / total : 0))" "$(mb "$have")" "$(mb "$total")" >&2
}

echo "downloading $binary"
curl -fsSL -D "$tmp/headers" "$url/$binary" -o "$tmp/homewend" 2>"$tmp/error" &
download=$!
if [ -t 2 ]; then
	while kill -0 "$download" 2>/dev/null; do
		bar
		sleep 0.1
	done
	bar
	echo >&2
fi
wait "$download" || {
	cat "$tmp/error" >&2
	fail "download failed: $url/$binary"
}
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
	on="$blue"
else
	on="" off=""
fi
# Prose breaks between words, at the width of the window.
say() {
	echo "$*" | fold -s -w "$((cols > 80 ? 80 : cols))"
}
echo
say "homewend $("$dir/homewend" version) is installed in $dir."
echo
say "Next, sign in to Google, once. A small window opens; your password goes to Google only:"
echo
echo "  ${on}$run login${off}"
echo
say "Then bring your photos home, here one year of them:"
echo
echo "  ${on}$run get --year 2025 --library ~/Pictures/Homewend${off}"
echo
echo "All the commands: ${on}$run help${off}"
case ":$PATH:" in
*":$dir:"*) ;;
*) say "($dir is not in your PATH: add it to type just \"homewend\".)" ;;
esac
