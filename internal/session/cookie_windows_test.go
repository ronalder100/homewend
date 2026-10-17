// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package session

import (
	"crypto/aes"
	"crypto/cipher"
	"testing"
)

// A v10 value, as Chrome writes it on Windows: the prefix, a nonce, then the
// GCM seal of a 32-byte domain hash and the value. decrypt should give the
// value back, hash stripped. The key is seeded so DPAPI is not involved.
func TestDecryptV10(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	sess := &Session{key: key}

	hash := make([]byte, 32) // zero bytes: unprintable, as a domain hash is
	plain := append(hash, []byte("SID=a-session-value")...)

	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatal(err)
	}
	nonce := make([]byte, gcm.NonceSize())
	raw := append([]byte("v10"), nonce...)
	raw = gcm.Seal(raw, nonce, plain, nil)

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
