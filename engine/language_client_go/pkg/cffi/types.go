package cffi

type Buffer struct {
	Ptr uintptr
	Len uintptr
}

// Global function pointers to be populated by RegisterFunctions
// purego: unsupported kind unsafe.Pointer -> changing to uintptr where possible for function args/returns
// Also struct fields must not use unsafe.Pointer for purego if it checks recursively.
var (
	VersionFn                 func() *byte
	CreateBamlRuntimeFn       func(rootPath string, srcFiles string, envVars string) uintptr
	DestroyBamlRuntimeFn      func(runtime uintptr)
	InvokeRuntimeCliFn        func(args uintptr) int32 // args is char**
	RegisterCallbacksFn       func(onSuccess, onError, onTick uintptr)
	CallFunctionFromCFn       func(runtime uintptr, fnName string, args *byte, argsLen uintptr, id uint32) *byte
	CallFunctionStreamFromCFn func(runtime uintptr, fnName string, args *byte, argsLen uintptr, id uint32) *byte
	CallFunctionParseFromCFn  func(runtime uintptr, fnName string, args *byte, argsLen uintptr, id uint32) *byte
	CancelFunctionCallFn      func(id uint32) *byte
	CallObjectConstructorFn   func(encodedArgs *byte, argsLen uintptr) Buffer
	CallObjectMethodFn        func(runtime uintptr, encodedArgs *byte, argsLen uintptr) Buffer
	FreeBufferFn              func(buf Buffer)
)
