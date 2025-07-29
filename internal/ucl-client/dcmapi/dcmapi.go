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

package dcmapi

import (
	"bytes"
	"net"
	"os"
	"ucl-tools/internal/ucl"
	. "ucl-tools/internal/ulog"
	"unsafe"
)

type Callback func(unsafe.Pointer, unsafe.Pointer)

func uclLcmConnection() (net.Conn, error) {

	var err error
	var conn net.Conn

	vscrnDef, err := ucl.ReadVScrnDef()
	if err != nil {
		return nil, err
	}

	targetAddr, err := ucl.GetLcmNodeAddr(vscrnDef)
	if err != nil {
		return nil, err
	}

	conn, err = ucl.ConnectTarget(targetAddr)
	if err != nil {
		return nil, err
	}
	ILog.Println("Dial connected to ", targetAddr)

	return conn, nil
}

func executeCommand(data string, command string) []byte {

	conn, err := uclLcmConnection()
	if err != nil {
		ELog.Println("UclLcmConnection: ", err)
		return nil
	}
	defer conn.Close()

	err = ucl.SendCommand(conn, command, data)
	if err != nil {
		ELog.Println("sendCommand: ", err)
		return nil
	}

	//wait for message from LifeCycleManager.
	readBuf, err := ucl.ReadStatus(conn)
	if err != nil {
		ILog.Printf("Termination from  LifeCycleManager : %s \n", err)
		return nil
	} else {
		ILog.Printf("Notification from  LifeCycleManager : %s \n", string(readBuf))
		return readBuf
	}
}

func parseStatus(data []byte) int {

	switch string(data) {
	case ucl.STAT_ExecSuccess:
		return 1
	case ucl.STAT_ExecFin:
		return 0
	case ucl.STAT_ExecErr:
		return -1

	case ucl.STAT_AppRunning:
		return 1
	case ucl.STAT_AppStop:
		return 0

	default:
		return -1
	}

}

func DcmGetExecutableAppList() []byte {

	ILog.SetOutput(os.Stderr)
	ILog.Println("DcmGetAppStatus")

	newJson, err := GenerateAppComm("getAppList")
	if err != nil {
		ELog.Printf("Json Command ReadAll error: %s \n", err)
		return nil
	}

	prefix := []byte("AppName:")
	retData := executeCommand(newJson, ucl.CMD_GetExecutableAppList)
	if retData != nil {
		if bytes.HasPrefix(retData, prefix) {
			applist := retData[len(prefix):]
			parts := bytes.Split(applist, []byte{','})
			seen := make(map[string]struct{}, len(parts))
			var out [][]byte
			for _, p := range parts {
				key := string(p)
				if _, exists := seen[key]; !exists {
					seen[key] = struct{}{}
					out = append(out, p)
				}
			}
			return bytes.Join(out, []byte{','})
		} else {
			return nil
		}
	}

	return nil
}

func DcmGetAppStatus(appName string) int {

	ILog.SetOutput(os.Stderr)
	ILog.Println("DcmGetAppStatus: ", appName)

	newJson, err := GenerateAppComm(appName)
	if err != nil {
		ELog.Printf("Json Command ReadAll error: %s \n", err)
		return -1
	}

	retStatus := executeCommand(newJson, ucl.CMD_GetAppStatus)
	ret := parseStatus(retStatus)

	return ret
}

func DcmRunApp(appName string) int {

	ILog.SetOutput(os.Stderr)
	ILog.Println("DcmRunApp: ", appName)

	newJson, err := GenerateAppComm(appName)
	if err != nil {
		ELog.Printf("Json Command ReadAll error: %s \n", err)
		return -1
	}

	retStatus := executeCommand(newJson, ucl.CMD_RunApp)
	ret := parseStatus(retStatus)

	return ret
}

func DcmRunAppAsync(appName string) int {

	ILog.SetOutput(os.Stderr)
	ILog.Println("DcmRunAppAsync: ", appName)

	newJson, err := GenerateAppComm(appName)
	if err != nil {
		ELog.Printf("Json Command ReadAll error: %s \n", err)
		return -1
	}

	retStatus := executeCommand(newJson, ucl.CMD_RunAppAsync)
	ret := parseStatus(retStatus)

	return ret
}

func DcmRunAppAsyncCb(appName string, callback Callback, funcPointer unsafe.Pointer, argPointer unsafe.Pointer) int {

	ILog.SetOutput(os.Stderr)
	ILog.Println("DcmRunAppAsyncCb: ", appName)

	newJson, err := GenerateAppComm(appName)
	if err != nil {
		ELog.Printf("Json Command ReadAll error: %s \n", err)
		return -1
	}

	retStatus := executeCommand(newJson, ucl.CMD_RunAppAsyncCb)
	ret := parseStatus(retStatus)

	callback(funcPointer, argPointer)

	return ret
}

func DcmStopApp(appName string) int {

	ILog.SetOutput(os.Stderr)
	ILog.Println("DcmStopApp: ", appName)

	newJson, err := GenerateStopAppComm(appName)
	if err != nil {
		ELog.Printf("Json Command ReadAll error: %s \n", err)
		return -1
	}

	retStatus := executeCommand(newJson, ucl.CMD_StopApp)
	ret := parseStatus(retStatus)

	return ret
}

func DcmStopAppAll() int {

	ILog.SetOutput(os.Stderr)
	ILog.Println("DcmStopAppAll")

	newJson, err := GenerateStopAppComm("AllApp")
	if err != nil {
		ELog.Printf("Json Command ReadAll error: %s \n", err)
		return -1
	}

	retStatus := executeCommand(newJson, ucl.CMD_StopAppAll)
	ret := parseStatus(retStatus)

	return ret
}

func DcmLaunchCompositor() int {

	ILog.SetOutput(os.Stderr)
	ILog.Println("DcmLaunchCompositor")

	newJson, err := GenerateLaunchCompsitorsComm()
	if err != nil {
		ELog.Printf("Json Command ReadAll error: %s \n", err)
		return -1
	}

	retStatus := executeCommand(newJson, ucl.CMD_LaunchCompositors)
	ret := parseStatus(retStatus)

	return ret
}

func DcmLaunchCompositorAsync() int {

	ILog.SetOutput(os.Stderr)
	ILog.Println("DcmLaunchCompositorAsync")

	newJson, err := GenerateLaunchCompsitorsComm()
	if err != nil {
		ELog.Printf("Json Command ReadAll error: %s \n", err)
		return -1
	}

	retStatus := executeCommand(newJson, ucl.CMD_LaunchCompositorsAsync)
	ret := parseStatus(retStatus)

	return ret
}

func DcmLaunchCompositorAsyncCb(callback Callback, funcPointer unsafe.Pointer, argPointer unsafe.Pointer) int {

	ILog.SetOutput(os.Stderr)
	ILog.Println("DcmLaunchCompositorAsyncCb")

	newJson, err := GenerateLaunchCompsitorsComm()
	if err != nil {
		ELog.Printf("Json Command Generate error: %s \n", err)
		return -1
	}

	retStatus := executeCommand(newJson, ucl.CMD_LaunchCompositorsAsyncCb)
	ret := parseStatus(retStatus)

	callback(funcPointer, argPointer)

	return ret
}

func DcmStopCompositor() int {

	ILog.SetOutput(os.Stderr)
	ILog.Println("DcmStopApp")

	newJson, err := GenerateStopCompsitorsComm()
	if err != nil {
		ELog.Printf("Json Command Generate error: %s \n", err)
		return -1
	}

	retStatus := executeCommand(newJson, ucl.CMD_StopCompositors)
	ret := parseStatus(retStatus)

	return ret
}

func main() {}
