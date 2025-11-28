//go:build windows

package baml_go

import (
	"fmt"
	"syscall"
)

func loadLibrary(path string) (uintptr, error) {
	handle, err := syscall.LoadLibrary(path)
	if err != nil {
		return 0, err
	}
	return uintptr(handle), nil
}

func closeLibrary(handle uintptr) error {
	err := syscall.FreeLibrary(syscall.Handle(handle))
	return err
}

func getSymbol(handle uintptr, name string) (uintptr, error) {
	addr, err := syscall.GetProcAddress(syscall.Handle(handle), name)
	if err != nil {
		return 0, err
	}
	return uintptr(addr), nil
}

