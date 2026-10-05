// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

// The version the app and its engine are built with, the CLI's: the release
// tag when the release workflow builds it, else what git says of this commit.
// A build between releases is a prerelease of the last one (0.1.2-dev.5.gabc1234),
// which the updater knows to leave alone.
const { execSync } = require('node:child_process');

const ref = process.env.GITHUB_REF_NAME || '';
const described = ref.startsWith('v')
	? ref
	: execSync('git describe --tags --match "v*"', { cwd: __dirname }).toString().trim();
const [, release, ahead, commit] = described.match(/^v(\d+\.\d+\.\d+)(?:-(\d+)-(g[0-9a-f]+))?$/) ?? [];
if (!release) throw new Error(`not a version: ${described}`);
process.stdout.write(ahead ? `${release}-dev.${ahead}.${commit}` : release);
