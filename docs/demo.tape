# homewend — Copyright (C) 2026 Ron Alder
# SPDX-License-Identifier: AGPL-3.0-or-later

# The terminal recording at the top of the README, made with VHS
# (https://github.com/charmbracelet/vhs):
#
#   vhs docs/demo.tape
#
# It is a real run, never a mock-up: sign in with `homewend login` first, on a
# machine with a browser. The wait for Google and the downloads are cut out
# with Hide, and the recording resumes on the screen as the run left it.

Output docs/demo.gif

Require homewend

Set Shell bash
Set FontSize 16
Set Width 1200
Set Height 640
Set Padding 24
Set TypingSpeed 40ms

Hide
Type "export LIBRARY=$(mktemp -d)/Homewend && clear"
Enter
Show

Type "homewend login"
Enter
Wait /signed in/
Sleep 1.5s

Type "homewend get --year 2025 --library $LIBRARY"
Enter
Sleep 6s

Hide
Wait+Screen@180m /declared \d+, on disk \d+, missing \d+/
Show
Sleep 8s
