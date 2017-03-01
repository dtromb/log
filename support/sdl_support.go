//+build sdl

package support

import (
	"unsafe"
)

/*
	#cgo pkg-config: sdl2
	#include <SDL_log.h>
*/
import "C"

//export sdlLogOutputDispatch
func sdlLogOutputDispatch(userdata *C.char, category C.int, pri C.SDL_LogPriority, msg *C.char) {
	slu := (*SdlLogUserdata)(unsafe.Pointer(userdata))
	<-slu.lock
	defer func() { slu.lock <- true }()
	for ctx, _ := range slu.contexts {
		<-ctx.lock
		defer func() {
			select {
				case ctx.lock <- true:
				default: {}
			}
		}()
		cat, has := ctx.getCategoryByCode(int(category))
		if !has {
			ctx.lock <- true
			continue
		}
		pri := SdlLogPriority(pri).Level()
		msg := C.GoString(msg)
		ctx.dispatch(cat, pri, msg)
	}
}