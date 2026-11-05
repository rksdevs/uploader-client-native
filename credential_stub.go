//go:build !windows

package main

import "errors"

func credentialWrite(target, secret string) error {
	return errors.New("credential storage is only available on Windows")
}

func credentialRead(target string) (string, error) { return "", nil }

func credentialDelete(target string) error { return nil }
