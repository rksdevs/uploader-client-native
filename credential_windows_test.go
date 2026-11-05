//go:build windows

package main

import "testing"

func TestCredentialRoundTrip(t *testing.T) {
	const target = "WoWLogsUploader/test-roundtrip"
	t.Cleanup(func() { _ = credentialDelete(target) })

	if err := credentialWrite(target, "wlds_secret-value"); err != nil {
		t.Fatal(err)
	}
	got, err := credentialRead(target)
	if err != nil {
		t.Fatal(err)
	}
	if got != "wlds_secret-value" {
		t.Fatalf("got %q", got)
	}
	if err := credentialDelete(target); err != nil {
		t.Fatal(err)
	}
	if got, err := credentialRead(target); err != nil || got != "" {
		t.Fatalf("after delete: %q %v", got, err)
	}
	if err := credentialDelete(target); err != nil {
		t.Fatalf("deleting a missing credential should be fine: %v", err)
	}
}
