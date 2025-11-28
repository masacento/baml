//go:build darwin || linux

package baml_go

import (
	"github.com/ebitengine/purego"
)

func loadLibrary(path string) (uintptr, error) {
	return purego.Dlopen(path, purego.RTLD_NOW|purego.RTLD_GLOBAL)
}

func closeLibrary(handle uintptr) error {
	return purego.Dlclose(handle)
}

func getSymbol(handle uintptr, name string) (uintptr, error) {
	return purego.Dlsym(handle, name)
}

