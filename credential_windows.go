//go:build windows

package main

import (
	"errors"
	"syscall"
	"unsafe"
)

const (
	credTypeGeneric         = 1
	credPersistLocalMachine = 2
	errorNotFound           = syscall.Errno(1168)
)

var (
	advapi32       = syscall.NewLazyDLL("advapi32.dll")
	procCredWrite  = advapi32.NewProc("CredWriteW")
	procCredRead   = advapi32.NewProc("CredReadW")
	procCredDelete = advapi32.NewProc("CredDeleteW")
	procCredFree   = advapi32.NewProc("CredFree")
)

// winCredential mirrors CREDENTIALW.
type winCredential struct {
	Flags              uint32
	Type               uint32
	TargetName         *uint16
	Comment            *uint16
	LastWrittenLow     uint32
	LastWrittenHigh    uint32
	CredentialBlobSize uint32
	CredentialBlob     *byte
	Persist            uint32
	AttributeCount     uint32
	Attributes         uintptr
	TargetAlias        *uint16
	UserName           *uint16
}

func credentialWrite(target, secret string) error {
	if secret == "" {
		return errors.New("empty credential")
	}
	name, err := syscall.UTF16PtrFromString(target)
	if err != nil {
		return err
	}
	blob := []byte(secret)
	cred := winCredential{
		Type:               credTypeGeneric,
		TargetName:         name,
		CredentialBlobSize: uint32(len(blob)),
		CredentialBlob:     &blob[0],
		Persist:            credPersistLocalMachine,
	}
	ok, _, callErr := procCredWrite.Call(uintptr(unsafe.Pointer(&cred)), 0)
	if ok == 0 {
		return callErr
	}
	return nil
}

// credentialRead returns "" when no credential is stored.
func credentialRead(target string) (string, error) {
	name, err := syscall.UTF16PtrFromString(target)
	if err != nil {
		return "", err
	}
	var cred *winCredential
	ok, _, callErr := procCredRead.Call(
		uintptr(unsafe.Pointer(name)),
		credTypeGeneric,
		0,
		uintptr(unsafe.Pointer(&cred)),
	)
	if ok == 0 {
		if errors.Is(callErr, errorNotFound) {
			return "", nil
		}
		return "", callErr
	}
	defer procCredFree.Call(uintptr(unsafe.Pointer(cred)))
	if cred.CredentialBlobSize == 0 {
		return "", nil
	}
	return string(unsafe.Slice(cred.CredentialBlob, cred.CredentialBlobSize)), nil
}

func credentialDelete(target string) error {
	name, err := syscall.UTF16PtrFromString(target)
	if err != nil {
		return err
	}
	ok, _, callErr := procCredDelete.Call(uintptr(unsafe.Pointer(name)), credTypeGeneric, 0)
	if ok == 0 && !errors.Is(callErr, errorNotFound) {
		return callErr
	}
	return nil
}
