// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build !windows

package session

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"runtime"
	"testing"
)

// A v10 value, as Chrome writes it on Linux and macOS: the prefix, then the
// CBC encryption of a 32-byte domain hash and the value, padded. decrypt
// should give the value back, hash stripped.
func TestDecryptV10(t *testing.T) {
	key := keyFor(runtime.GOOS)
	sess := &Session{key: key}

	hash := make([]byte, 32) // zero bytes: unprintable, as a domain hash is
	plain := append(hash, []byte("SID=a-session-value")...)
	pad := aes.BlockSize - len(plain)%aes.BlockSize
	plain = append(plain, bytes.Repeat([]byte{byte(pad)}, pad)...)

	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	body := make([]byte, len(plain))
	cipher.NewCBCEncrypter(block, []byte("                ")).CryptBlocks(body, plain)
	raw := append([]byte("v10"), body...)

	if got := sess.decrypt(raw); got != "SID=a-session-value" {
		t.Errorf("decrypt: got %q, want %q", got, "SID=a-session-value")
	}

	// A value the browser left in the clear is returned as it is.
	if got := sess.decrypt([]byte("plain")); got != "plain" {
		t.Errorf("clear value: got %q", got)
	}
	// Garbage under a v10 prefix decrypts to nothing, not to a panic.
	if got := sess.decrypt([]byte("v10short")); got != "" {
		t.Errorf("short v10: got %q, want empty", got)
	}
}
