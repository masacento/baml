package baml

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"unsafe"

	"github.com/boundaryml/baml/engine/language_client_go/baml_go"
	"github.com/boundaryml/baml/engine/language_client_go/pkg/cffi"
)

type BamlRuntime struct {
	runtime uintptr
}

func NewClientRegistry() *ClientRegistry {
	return &ClientRegistry{}
}

func InvokeRuntimeCli(args []string) int {
	if err := baml_go.GetInitError(); err != nil {
		fmt.Printf("BAML Init Error: %v\n", err)
		return 1
	}

	// Convert []string to char**
	cArgs := make([]*byte, len(args)+1)
	keepAlive := make([][]byte, len(args))

	for i, arg := range args {
		s := arg + "\x00"
		b := []byte(s)
		keepAlive[i] = b
		cArgs[i] = &b[0]
	}
	cArgs[len(args)] = nil

	ptr := &cArgs[0]
	ret := cffi.InvokeRuntimeCliFn(uintptr(unsafe.Pointer(ptr)))

	runtime.KeepAlive(keepAlive)
	runtime.KeepAlive(cArgs)

	return int(ret)
}

func CreateRuntime(
	root_path string,
	src_files map[string]string,
	env_vars map[string]string,
) (BamlRuntime, error) {
	if err := baml_go.GetInitError(); err != nil {
		return BamlRuntime{}, err
	}

	src_files_json, err := json.Marshal(src_files)
	if err != nil {
		return BamlRuntime{}, err
	}

	env_vars_json, err := json.Marshal(env_vars)
	if err != nil {
		return BamlRuntime{}, err
	}

	handle := cffi.CreateBamlRuntimeFn(root_path, string(src_files_json), string(env_vars_json))
	if handle == 0 {
		return BamlRuntime{}, fmt.Errorf("failed to create BAML runtime")
	}

	rt := BamlRuntime{runtime: handle}

	// Ensure callbacks are registered
	ensureCallbacksRegistered()

	return rt, nil
}

func (r *BamlRuntime) Close() {
	if r.runtime != 0 {
		cffi.DestroyBamlRuntimeFn(r.runtime)
		r.runtime = 0
	}
}

// CallFunction calls a BAML function.
// The args parameter should be the protobuf encoded bytes of the function arguments.
func (r *BamlRuntime) CallFunction(ctx context.Context, fnName string, args []byte, onTick OnTickCallbackData) (*ResultCallback, error) {
	id, ch := create_unique_id(ctx, onTick)

	var argsPtr *byte
	if len(args) > 0 {
		argsPtr = &args[0]
	}

	errStrPtr := cffi.CallFunctionFromCFn(r.runtime, fnName, argsPtr, uintptr(len(args)), id)

	if errStrPtr != nil {
		// Immediate error
		errStr := cStringToGoString(errStrPtr)
		callbackMutex.Lock()
		if cb, ok := dynamicCallbacks[id]; ok {
			close(cb.channel)
			delete(dynamicCallbacks, id)
		}
		callbackMutex.Unlock()

		return nil, fmt.Errorf("baml call failed: %s", errStr)
	}

	// Wait for callback
	select {
	case res := <-ch:
		return &res, nil
	case <-ctx.Done():
		// Cancel the call
		cffi.CancelFunctionCallFn(id)
		return nil, ctx.Err()
	}
}

// CallFunctionStream calls a BAML function with streaming.
// The args parameter should be the protobuf encoded bytes of the function arguments.
func (r *BamlRuntime) CallFunctionStream(ctx context.Context, fnName string, args []byte, onTick OnTickCallbackData) (<-chan ResultCallback, error) {
	id, ch := create_unique_id(ctx, onTick)

	var argsPtr *byte
	if len(args) > 0 {
		argsPtr = &args[0]
	}

	errStrPtr := cffi.CallFunctionStreamFromCFn(r.runtime, fnName, argsPtr, uintptr(len(args)), id)

	if errStrPtr != nil {
		errStr := cStringToGoString(errStrPtr)
		callbackMutex.Lock()
		if cb, ok := dynamicCallbacks[id]; ok {
			close(cb.channel)
			delete(dynamicCallbacks, id)
		}
		callbackMutex.Unlock()
		return nil, fmt.Errorf("baml call failed: %s", errStr)
	}

	return ch, nil
}

// CallFunctionParse calls a BAML parse function.
// The args parameter should be the protobuf encoded bytes of the function arguments.
func (r *BamlRuntime) CallFunctionParse(ctx context.Context, fnName string, args []byte) (any, error) {
	// Parse function doesn't have OnTick
	id, ch := create_unique_id(ctx, nil)

	var argsPtr *byte
	if len(args) > 0 {
		argsPtr = &args[0]
	}

	errStrPtr := cffi.CallFunctionParseFromCFn(r.runtime, fnName, argsPtr, uintptr(len(args)), id)

	if errStrPtr != nil {
		errStr := cStringToGoString(errStrPtr)
		callbackMutex.Lock()
		if cb, ok := dynamicCallbacks[id]; ok {
			close(cb.channel)
			delete(dynamicCallbacks, id)
		}
		callbackMutex.Unlock()
		return nil, fmt.Errorf("baml call failed: %s", errStr)
	}

	// Wait for callback
	select {
	case res := <-ch:
		if res.Error != nil {
			return nil, res.Error
		}
		if res.HasData {
			return res.Data, nil
		}
		if res.HasStreamData {
			return res.StreamData, nil
		}
		return nil, fmt.Errorf("no data returned")
	case <-ctx.Done():
		cffi.CancelFunctionCallFn(id)
		return nil, ctx.Err()
	}
}

func cStringToGoString(ptr *byte) string {
	if ptr == nil {
		return ""
	}
	var length int
	for {
		p := unsafe.Pointer(uintptr(unsafe.Pointer(ptr)) + uintptr(length))
		val := *(*byte)(p)
		if val == 0 {
			break
		}
		length++
	}
	return string(unsafe.Slice(ptr, length))
}
