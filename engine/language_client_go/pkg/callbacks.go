package baml

import (
	"context"
	"math/rand"
	"sync"
	"unsafe"

	"github.com/boundaryml/baml/engine/language_client_go/baml_go"
	"github.com/boundaryml/baml/engine/language_client_go/baml_go/serde"
	"github.com/boundaryml/baml/engine/language_client_go/pkg/cffi"
	"github.com/ebitengine/purego"
	"google.golang.org/protobuf/proto"
)

type BamlError struct {
	Message string
}

func (e BamlError) Error() string {
	return e.Message
}

type BamlClientError struct {
	BamlError
}

type BamlClientHttpError struct {
	BamlClientError
}

type ResultCallback struct {
	Error         error
	HasStreamData bool
	HasData       bool
	StreamData    any
	Data          any
}

type CallbackData struct {
	channel chan ResultCallback
	ctx     context.Context
	onTick  OnTickCallbackData
}

type OnTickCallbackData interface {
	Collector() Collector
	OnTick() TickCallback
}

// Map to store callbacks by ID
var (
	dynamicCallbacks = make(map[uint32]CallbackData)
	callbackMutex    sync.RWMutex
	typeMap          serde.TypeMap
	registerOnce     sync.Once
)

func SetTypeMap(t serde.TypeMap) {
	typeMap = t
}

func ensureCallbacksRegistered() {
	if baml_go.GetInitError() != nil {
		return
	}
	registerOnce.Do(func() {
		success := purego.NewCallback(trigger_callback)
		fail := purego.NewCallback(error_callback)
		tick := purego.NewCallback(on_tick_callback)

		cffi.RegisterCallbacksFn(success, fail, tick)
	})
}

func on_tick_callback(_ purego.CDecl, id uint32) {
	callbackMutex.RLock()
	callback, exists := dynamicCallbacks[id]
	callbackMutex.RUnlock()

	if exists {
		data := callback.onTick
		if data != nil {
			last, err := data.Collector().Last()
			if err != nil {
				return
			}
			data.OnTick()(callback.ctx, TickReason_Unknown, last)
		}
	}
}

func error_callback(_ purego.CDecl, id uint32, isDone int32, content *byte, length uintptr) {
	callbackMutex.RLock()
	callback, exists := dynamicCallbacks[id]
	callbackMutex.RUnlock()

	if exists {
		// Copy content
		contentBytes := make([]byte, length)
		if length > 0 {
			src := unsafe.Slice(content, length)
			copy(contentBytes, src)
		}

		// Parse the content as a string
		content_str := string(contentBytes)

		// Send the error to the callback
		if content_str == "AbortError" {
			// Special handling for AbortError
			callback.channel <- ResultCallback{Error: callback.ctx.Err()}
		} else {
			err := BamlError{Message: content_str}
			callback.channel <- ResultCallback{Error: err}
		}

		close(callback.channel)
		callbackMutex.Lock()
		defer callbackMutex.Unlock()
		delete(dynamicCallbacks, id)
	}
}

func trigger_callback(_ purego.CDecl, id uint32, isDone int32, content *byte, length uintptr) {
	callbackMutex.RLock()
	callback, exists := dynamicCallbacks[id]
	callbackMutex.RUnlock()

	if exists {
		// Copy content
		contentBytes := make([]byte, length)
		if length > 0 {
			src := unsafe.Slice(content, length)
			copy(contentBytes, src)
		}

		var content_holder cffi.CFFIValueHolder
		err := proto.Unmarshal(contentBytes, &content_holder)
		if err != nil {
			callback.channel <- ResultCallback{Error: err}
			close(callback.channel)
			callbackMutex.Lock()
			defer callbackMutex.Unlock()
			delete(dynamicCallbacks, id)
			return
		}

		decoded_data := serde.Decode(&content_holder, typeMap).Interface()

		var res ResultCallback
		if isDone == 1 {
			res = ResultCallback{HasData: true, Data: decoded_data}
		} else {
			res = ResultCallback{HasStreamData: true, StreamData: decoded_data}
		}

		callback.channel <- res
		if isDone == 1 {
			close(callback.channel)
			callbackMutex.Lock()
			defer callbackMutex.Unlock()
			delete(dynamicCallbacks, id)
		}
	}
}

func create_unique_id(ctx context.Context, onTick OnTickCallbackData) (uint32, chan ResultCallback) {
	ensureCallbacksRegistered()
	callbackMutex.Lock()
	defer callbackMutex.Unlock()
	id := uint32(rand.Intn(1000000))
	for _, exists := dynamicCallbacks[id]; exists; {
		id = uint32(rand.Intn(1000000))
	}
	dynamicCallbacks[id] = CallbackData{channel: make(chan ResultCallback), ctx: ctx, onTick: onTick}
	return id, dynamicCallbacks[id].channel
}
