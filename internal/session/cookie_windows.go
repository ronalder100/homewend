// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package session

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

// On Windows Chrome encrypts each cookie value with AES-256-GCM, under a key it
// keeps in the profile's own Local State file. That key is itself wrapped with
// DPAPI, tied to the Windows user account, so the one who can unwrap it is the
// account that owns the profile — us, reading our own profile back. This is the
// Windows counterpart of the AES-128-CBC path on Linux and macOS (see
// cookie_unix.go): same profile, same cookies, read the same way.
//
// Measured on Windows Server 2025 with Chrome 154 and Edge 153: the key is
// base64 under os_crypt.encrypted_key, prefixed "DPAPI", and every cookie value
// began with v10 — the app-bound (v20) encryption was not in use on a profile
// of ours (2026-10-01).
func (s *Session) loadCookieKey() ([]byte, error) {
	data, err := os.ReadFile(filepath.Join(s.Profile, "Local State"))
	if err != nil {
		return nil, fmt.Errorf("reading the cookie key: %w", err)
	}
	var state struct {
		OSCrypt struct {
			EncryptedKey string `json:"encrypted_key"`
		} `json:"os_crypt"`
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("reading the cookie key: %w", err)
	}
	blob, err := base64.StdEncoding.DecodeString(state.OSCrypt.EncryptedKey)
	if err != nil {
		return nil, fmt.Errorf("reading the cookie key: %w", err)
	}
	const prefix = "DPAPI"
	if !strings.HasPrefix(string(blob), prefix) {
		return nil, fmt.Errorf("reading the cookie key: not a DPAPI key")
	}
	return unprotect(blob[len(prefix):])
}

// unprotect unwraps a DPAPI blob with the current Windows user's key, the
// CryptUnprotectData the browser itself uses (dpapi.h).
func unprotect(blob []byte) ([]byte, error) {
	if len(blob) == 0 {
		return nil, fmt.Errorf("unwrapping the cookie key: empty")
	}
	in := windows.DataBlob{Size: uint32(len(blob)), Data: &blob[0]}
	var out windows.DataBlob
	if err := windows.CryptUnprotectData(&in, nil, nil, 0, nil, 0, &out); err != nil {
		return nil, fmt.Errorf("unwrapping the cookie key: %w", err)
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))
	return append([]byte(nil), unsafe.Slice(out.Data, out.Size)...), nil
}

func (s *Session) decrypt(raw []byte) string {
	if len(raw) < 3 {
		return ""
	}
	switch string(raw[:3]) {
	case "v10", "v11":
	default:
		return string(raw) // stored in the clear
	}
	key, err := s.cookieKey()
	if err != nil {
		return ""
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return ""
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return ""
	}
	// After the v10 prefix: a nonce, the ciphertext, then the GCM tag.
	body := raw[3:]
	if len(body) < gcm.NonceSize() {
		return ""
	}
	out, err := gcm.Open(nil, body[:gcm.NonceSize()], body[gcm.NonceSize():], nil)
	if err != nil {
		return ""
	}
	return string(stripDomainHash(out))
}
