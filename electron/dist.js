// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

// Builds the engine, then the app around it, both with the version from
// version.js. Here and not in package.json's scripts: npm runs those with
// cmd.exe on Windows, which has no $(...) to put the version in.
// Nothing is published from here: the release workflow uploads what it needs.
const { execFileSync } = require('node:child_process');
const path = require('node:path');
const builder = require('electron-builder');
const version = require('./version.js');

// The trailing separator makes -o a folder: go build names the binary,
// homewend or homewend.exe.
execFileSync(
	'go',
	['build', '-trimpath', '-ldflags', `-X main.version=${version}`, '-o', path.join('electron', 'engine') + path.sep, './cmd/homewend'],
	{ cwd: path.join(__dirname, '..'), stdio: 'inherit' },
);
builder.build({ config: { extraMetadata: { version } }, publish: 'never' }).catch((err) => {
	console.error(err);
	process.exit(1);
});
