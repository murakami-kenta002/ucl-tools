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

import (
	"flag"
	"fmt"
	"os"
	"ucl-tools/internal/ucl-client/dcmapi"
)

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage:\n")
	fmt.Fprintf(os.Stderr, " %s [OPTIONS]\n\n", os.Args[0])

	fmt.Fprintf(os.Stderr, "Options:\n")
	fmt.Fprintf(os.Stderr, "  -c\tSpecify a DCM API command (default: none)\n")
	fmt.Fprintf(os.Stderr, "\tget_app_list \n")
	fmt.Fprintf(os.Stderr, "\tlaunch_compositor \n")
	fmt.Fprintf(os.Stderr, "\tstop_compositor   \n")
	fmt.Fprintf(os.Stderr, "\trun               <appName>\n")
	fmt.Fprintf(os.Stderr, "\tstop              <appName>\n")
	fmt.Fprintf(os.Stderr, "  -h\tShow this message\n")
}

func main() {

	var command string
	flag.Usage = printUsage
	flag.StringVar(&command, "c", "", "Specify a dcm api command (default: none)")
	flag.Parse()

	var appName string
	var val int

	flagCnt := len(flag.Args())
	if flagCnt > 0 {
		appName = flag.Arg(0)
	}

	switch command {
	case "launch_compositor":
		val = dcmapi.DcmLaunchCompositor()
	case "launch_compositor_async":
		val = dcmapi.DcmLaunchCompositorAsync()
	case "stop_compositor":
		val = dcmapi.DcmStopCompositor()
	case "run":
		val = dcmapi.DcmRunApp(appName)
	case "run_async":
		val = dcmapi.DcmRunAppAsync(appName)
	case "stop":
		val = dcmapi.DcmStopApp(appName)
	case "stop_all":
		val = dcmapi.DcmStopAppAll()
	case "get_app_status":
		val = dcmapi.DcmGetAppStatus(appName)
	case "get_app_list":
		list := dcmapi.DcmGetExecutableAppList()
		fmt.Fprintf(os.Stderr, "appList: %s\n", list)
		val = 0
	default:
		val = -1
	}

	fmt.Fprintf(os.Stderr, "exit result %d\n", val)
}
