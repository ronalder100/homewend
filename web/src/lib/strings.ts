// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

// Every sentence the window shows a person. A second language is a second
// table with the same keys.
export const text = {
	brand: 'Homewend',
	menu: 'Show or hide the sidebar',
	importFolder: 'Import a folder',

	signInTitle: 'Google Photos,\non your own disk.',
	signInLead:
		'No zips to babysit, no download to restart by hand. Plain folders, sorted by date and album, and nothing passes through our servers.',
	signInCall: 'Bring your photos home.',
	signInCallSub: 'One sign-in, and Homewend does the rest.',
	signInButton: 'Sign in to Google',
	signInWaiting: 'Waiting for Google',
	stepSignIn: 'Sign in',
	stepSignInDesc: 'Homewend never sees your password.',
	stepChoose: 'Choose',
	stepChooseDesc: 'Everything, a year, or a takeout you made.',
	stepBringHome: 'Bring home',
	stepBringHomeDesc: 'Homewend does the rest.',

	allPhotos: 'All photos',
	noDate: 'No date',
	photosAndVideos: (n: number) => `${n.toLocaleString('en')} photos and videos`,
	noPhotos: 'No photos yet',
	noPhotosDesc: 'They appear here as each part of the takeout arrives.',
	seeDownload: 'See the download',
	signingIn: 'Sign in in the browser window that opened, then come back here.',
	years: 'YEARS',
	albums: 'ALBUMS',
	takeouts: 'Takeouts',
	settings: 'Settings',
	more: (n: number) => `${n} more`,
	fewer: 'Show fewer',
	showAccount: (name: string) => `Show ${name}'s photos`,

	settingsTitle: 'Settings',
	appearance: 'APPEARANCE',
	theme: 'Theme',
	themeDesc: "System follows your computer's light or dark mode.",
	themeSystem: 'System',
	themeLight: 'Light',
	themeDark: 'Dark'
} as const;
