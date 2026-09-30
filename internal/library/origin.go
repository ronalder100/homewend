// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package library

import (
	"regexp"
	"strings"
)

// Origin says where a photo came from: a camera, a messaging app, a screenshot.
//
// It matters because a modern library is not what people picture. On the
// 344 GB export measured here, **9,513 of 38,584 distinct files came from
// WhatsApp** — a quarter of a lifetime's "photos" are pictures other people
// sent — plus 962 from Telegram, 172 screenshots and 65 from Signal. Rescuing
// all of it is right; showing it all mixed together is not, and neither is
// making someone scroll past six thousand forwarded memes to find their
// daughter's birthday.
//
// So the library keeps everything and tells the user what it is.
type Origin string

const (
	OriginCamera     Origin = "camera"
	OriginWhatsApp   Origin = "whatsapp"
	OriginSignal     Origin = "signal"
	OriginTelegram   Origin = "telegram"
	OriginScreenshot Origin = "screenshot"
	OriginOther      Origin = "other"
)

// FromMessaging reports whether a photo arrived through a chat rather than a
// lens. Screenshots are not messaging, but they belong to the same "not really
// a photograph" pile and the user usually wants them treated alike.
func (o Origin) FromMessaging() bool {
	switch o {
	case OriginWhatsApp, OriginSignal, OriginTelegram:
		return true
	}
	return false
}

// Google records the app that produced the file in the sidecar, as an Android
// package name. That is an authoritative answer, in no language, and it beats
// any amount of filename guessing — which is what every other tool does.
var packageOrigins = map[string]Origin{
	"com.whatsapp":                    OriginWhatsApp,
	"com.whatsapp.w4b":                OriginWhatsApp, // WhatsApp Business
	"org.thoughtcrime.securesms":      OriginSignal,
	"org.telegram.messenger":          OriginTelegram,
	"org.telegram.messenger.web":      OriginTelegram,
	"org.telegram.plus":               OriginTelegram,
	"com.android.systemui":            OriginScreenshot,
	"com.google.android.GoogleCamera": OriginCamera,
	"com.android.camera":              OriginCamera,
	"com.android.camera2":             OriginCamera,
	"com.sec.android.app.camera":      OriginCamera, // Samsung
	"net.sourceforge.opencamera":      OriginCamera,
}

// The fallback, for iOS exports and for the years before Google recorded the
// app. Each pattern is a naming convention the app itself imposes, not a guess:
// WhatsApp really does name every image IMG-<date>-WA<counter>.
var filenameOrigins = []struct {
	pattern *regexp.Regexp
	origin  Origin
}{
	{regexp.MustCompile(`(?i)^(IMG|VID|AUD|PTT|STK|DOC)-\d{8}-WA\d+`), OriginWhatsApp},
	{regexp.MustCompile(`(?i)^signal-\d{4}-\d{2}-\d{2}`), OriginSignal},
	{regexp.MustCompile(`(?i)^(photo|video)_\d{4}-\d{2}-\d{2}_`), OriginTelegram},
	{regexp.MustCompile(`(?i)^(photo|video)_\d+@\d{2}-\d{2}-\d{4}`), OriginTelegram},
	{regexp.MustCompile(`(?i)^(screenshot|screen_shot|schermata|captura|bildschirmfoto)[_\- ]`), OriginScreenshot},
	{regexp.MustCompile(`(?i)^(IMG|DSC|DSCF|DSCN|PXL|MVIMG|P|GOPR|DJI)[_\-]?\d{3,}`), OriginCamera},
	{regexp.MustCompile(`(?i)^\d{5}(IMG|PORTRAIT)_\d{5}_BURST`), OriginCamera}, // Pixel bursts
}

// OriginOf works out where a file came from, preferring what Google recorded
// over what the name suggests.
func OriginOf(packageName, filename string) Origin {
	if origin, ok := packageOrigins[packageName]; ok {
		return origin
	}
	base := strings.TrimSpace(filename)
	for _, candidate := range filenameOrigins {
		if candidate.pattern.MatchString(base) {
			return candidate.origin
		}
	}
	if packageName != "" {
		// A package we do not recognise is still not a camera, and pretending
		// otherwise would file a banking app's receipt among the holidays.
		return OriginOther
	}
	return OriginOther
}
