// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build !windows

package session

import (
	"crypto/aes"
	"crypto/cipher"
	"runtime"
)

// On Linux and macOS Chrome encrypts each cookie value with AES-128-CBC, under
// a key derived from a password it normally keeps in the system keyring. The
// browser is launched so that the password is a fixed one instead (see Open):
// otherwise the key differs on every machine, and on a Mac reading it means a
// Keychain prompt.
//
// On macOS, --use-mock-keychain makes Chrome use its fake keychain, whose
// password is "mock_password", derived with 1003 iterations: Chromium's
// crypto/apple/fake_keychain_v2.mm and
// components/os_crypt/async/browser/keychain_key_provider.mm, read 2026-09-26.
func (s *Session) loadCookieKey() ([]byte, error) { return keyFor(runtime.GOOS), nil }

// open decrypts a cookie value's body, after its v10 or v11 prefix: CBC under
// an IV of sixteen spaces, padded as PKCS#7.
func open(key, body []byte) []byte {
	if len(body)%aes.BlockSize != 0 || len(body) == 0 {
		return nil
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil
	}
	out := make([]byte, len(body))
	cipher.NewCBCDecrypter(block, []byte("                ")).CryptBlocks(out, body)

	if pad := int(out[len(out)-1]); pad > 0 && pad <= aes.BlockSize && pad <= len(out) {
		out = out[:len(out)-pad]
	}
	return out
}
