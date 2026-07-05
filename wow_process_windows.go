//go:build windows

package main

import (
	"strings"
	"syscall"
	"unsafe"
)

const (
	th32csSnapProcess = 0x00000002
	maxProcessName    = 260
)

type processEntry32 struct {
	Size            uint32
	CntUsage        uint32
	ProcessID       uint32
	DefaultHeapID   uintptr
	ModuleID        uint32
	CntThreads      uint32
	ParentProcessID uint32
	PriClassBase    int32
	Flags           uint32
	ExeFile         [maxProcessName]uint16
}

var (
	modKernel32                  = syscall.NewLazyDLL("kernel32.dll")
	procCreateToolhelp32Snapshot = modKernel32.NewProc("CreateToolhelp32Snapshot")
	procProcess32FirstW          = modKernel32.NewProc("Process32FirstW")
	procProcess32NextW           = modKernel32.NewProc("Process32NextW")
	procCloseHandle              = modKernel32.NewProc("CloseHandle")
)

func isWowProcessRunning() bool {
	snap, _, _ := procCreateToolhelp32Snapshot.Call(uintptr(th32csSnapProcess), 0)
	if snap == uintptr(syscall.InvalidHandle) {
		return false
	}
	defer procCloseHandle.Call(snap)

	var entry processEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	ret, _, _ := procProcess32FirstW.Call(snap, uintptr(unsafe.Pointer(&entry)))
	if ret == 0 {
		return false
	}

	for {
		name := strings.TrimSpace(syscall.UTF16ToString(entry.ExeFile[:]))
		if strings.EqualFold(name, "Wow.exe") {
			return true
		}
		ret, _, _ = procProcess32NextW.Call(snap, uintptr(unsafe.Pointer(&entry)))
		if ret == 0 {
			break
		}
	}
	return false
}
