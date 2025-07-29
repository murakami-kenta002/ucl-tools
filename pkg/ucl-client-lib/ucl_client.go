// SPDX-License-Identifier: Apache-2.0
/**
 * Copyright (c) 2024  Panasonic Automotive Systems, Co., Ltd.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

/*
#include <stdint.h>
#include <stdlib.h>
#include <string.h>

static inline void invokeCallback(uintptr_t f, void* arg) {
         ((void(*)(void*))f)(arg);
}
*/
import "C"
import (
	"sync"
	"ucl-tools/internal/ucl-client/dcmapi"
	"unsafe"
)

var Mutex struct {
	sync.Mutex
}

func doCallbackFunc(callBackFunc unsafe.Pointer, argPointer unsafe.Pointer) {
	Mutex.Lock()
	callback := uintptr(callBackFunc)
	if callback != 0 {
		C.invokeCallback(C.uintptr_t(callback), argPointer)
	}
	Mutex.Unlock()
}

var lastAllocated unsafe.Pointer

//export dcm_free_executable_app_list
func dcm_free_executable_app_list() {
	Mutex.Lock()
	defer Mutex.Unlock()
	if lastAllocated != nil {
		C.free(lastAllocated)
		lastAllocated = nil
	}
}

func allocAndKeep(data []byte) (*C.char, C.int) {
	n := len(data)
	if n == 0 {
		return nil, 0
	}

	ptr := C.malloc(C.size_t(n))
	if ptr == nil {
		return nil, 0
	}

	C.memcpy(ptr, unsafe.Pointer(&data[0]), C.size_t(n))

	Mutex.Lock()
	lastAllocated = ptr
	Mutex.Unlock()

	return (*C.char)(ptr), C.int(n)
}

//export dcm_get_executable_app_list
func dcm_get_executable_app_list(length *C.int) *C.char {

	dcm_free_executable_app_list()

	val := dcmapi.DcmGetExecutableAppList()
	cstr, clen := allocAndKeep(val)
	*length = clen

	return cstr
}

//export dcm_get_app_status
func dcm_get_app_status(appNameChar *C.char) C.int {
	appName := C.GoString(appNameChar)

	val := dcmapi.DcmGetAppStatus(appName)

	//1 = run, 0 = stop, -1 = fail
	return C.int(val)
}

//export dcm_run_app
func dcm_run_app(appNameChar *C.char) C.int {
	appName := C.GoString(appNameChar)

	val := dcmapi.DcmRunApp(appName)

	//0 = end_ok, -1 = end_fail
	return C.int(val)
}

//export dcm_run_app_async
func dcm_run_app_async(appNameChar *C.char) C.int {
	appName := C.GoString(appNameChar)

	val := dcmapi.DcmRunAppAsync(appName)

	//1 = start, 0 = end_ok, -1 = end_fail
	return C.int(val)
}

//export dcm_run_app_async_cb
func dcm_run_app_async_cb(appNameChar *C.char, callback unsafe.Pointer, argPointer unsafe.Pointer) C.int {
	callBackFunc := (unsafe.Pointer)(callback)
	appName := C.GoString(appNameChar)

	go dcmapi.DcmRunAppAsyncCb(appName, doCallbackFunc, callBackFunc, argPointer)

	return 0
}

//export dcm_stop_app
func dcm_stop_app(appNameChar *C.char) C.int {
	appName := C.GoString(appNameChar)

	val := dcmapi.DcmStopApp(appName)

	//0 = end_ok, -1 = end_fail
	return C.int(val)
}

//export dcm_stop_app_all
func dcm_stop_app_all() C.int {

	val := dcmapi.DcmStopAppAll()

	//0 = end_ok, -1 = end_fail
	return C.int(val)
}

//export dcm_launch_compositor
func dcm_launch_compositor() C.int {

	val := dcmapi.DcmLaunchCompositor()

	//0 = end_ok, -1 = end_fail
	return C.int(val)
}

//export dcm_launch_compositor_async
func dcm_launch_compositor_async() C.int {

	val := dcmapi.DcmLaunchCompositorAsync()

	//1 = start, 0 = end_ok, -1 = end_fail
	return C.int(val)

}

//export dcm_launch_compositor_async_cb
func dcm_launch_compositor_async_cb(callback unsafe.Pointer, argPointer unsafe.Pointer) C.int {
	callBackFunc := (unsafe.Pointer)(callback)

	go dcmapi.DcmLaunchCompositorAsyncCb(doCallbackFunc, callBackFunc, argPointer)

	return 0
}

//export dcm_stop_compositor
func dcm_stop_compositor() C.int {

	val := dcmapi.DcmStopCompositor()

	//0 = ok, -1 = fail
	return C.int(val)
}

func main() {}
