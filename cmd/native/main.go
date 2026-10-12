package main

/*
#include <stdlib.h>
*/
import "C"

import (
	"ALLinSSL/backend/desktop"
	"encoding/json"
	"fmt"
	"net/http"
	"unsafe"
)

var engine desktop.Runtime

//export AllinSSLControl
func AllinSSLControl(input *C.char) (output *C.char) {
	defer func() {
		if recovered := recover(); recovered != nil {
			data, _ := json.Marshal(map[string]any{"status": false, "message": fmt.Sprintf("业务引擎初始化失败: %v", recovered)})
			output = C.CString(string(data))
		}
	}()
	var command struct {
		Action    string `json:"action"`
		Directory string `json:"directory"`
		Password  string `json:"password"`
	}
	err := json.Unmarshal([]byte(C.GoString(input)), &command)
	entry := ""
	if err == nil {
		switch command.Action {
		case "start":
			entry, err = engine.Start(command.Directory)
		case "quit":
			engine.Stop()
		case "reset_password":
			err = engine.ResetPassword(command.Password)
		default:
			err = fmt.Errorf("不支持的软件控制操作")
		}
	}
	message := ""
	if err != nil {
		message = err.Error()
	}
	data, _ := json.Marshal(map[string]any{"status": err == nil, "message": message, "entry": entry})
	return C.CString(string(data))
}

//export AllinSSLRequest
func AllinSSLRequest(input *C.char, body unsafe.Pointer, length C.int, outputLength *C.longlong) unsafe.Pointer {
	var request struct {
		Method  string      `json:"method"`
		Path    string      `json:"path"`
		Headers http.Header `json:"headers"`
	}
	err := json.Unmarshal([]byte(C.GoString(input)), &request)
	status, headers, result := 400, http.Header{"Content-Type": {"application/json"}}, []byte(`{"status":false,"message":"软件请求格式错误"}`)
	if err == nil {
		status, headers, result = engine.Request(request.Method, request.Path, request.Headers, C.GoBytes(body, length))
	}
	metadata, _ := json.Marshal(map[string]any{"status": status, "headers": headers})
	data := append(append(metadata, '\n'), result...)
	*outputLength = C.longlong(len(data))
	return C.CBytes(data)
}

//export AllinSSLFree
func AllinSSLFree(pointer unsafe.Pointer) {
	C.free(pointer)
}

func main() {}
