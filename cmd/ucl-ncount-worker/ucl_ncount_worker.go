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
	"bufio"
	"flag"
	"fmt"
	"ucl-tools/internal/ucl"
	. "ucl-tools/internal/ulog"
	"math/rand"
	"net"
	"os"
	"strconv"
	"time"
)

func IsSliceByteEq(a, b []byte) bool {

	// If one is nil, the other must also be nil.
	if (a == nil) != (b == nil) {
		ILog.Printf("slice mismatch(nil)")
		return false
	}

	if len(a) != len(b) {
		ILog.Printf("slice mismatch(len:%d, %d)", len(a), len(b))
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			ILog.Printf("slice mismatch(cont)")
			return false
		}
	}

	return true
}

func replyWait(conn net.Conn, magicCode []byte) {

	cbio := bufio.NewReader(conn)
	buf, err := ucl.ConnReadWithSize(cbio)
	if err != nil {
		ELog.Println("ConnReadWithSize error: ", err)
		os.Exit(1)
	}
	DLog.Println(string(buf))

	if !IsSliceByteEq(buf, magicCode) {
		ELog.Printf("magic code mismatch")
		os.Exit(1)
	}

	ILog.Println("reply OK from Master")
}

func mainLoop(masterIp string, masterPort int, appName string) {

	rand.Seed(time.Now().UnixNano())

	magicCode := appName + strconv.Itoa(rand.Intn(1024))
	DLog.Println("magicCode", magicCode)

	waitConn := ucl.ConnectMaster(masterIp, masterPort, []byte(magicCode))

	replyWait(waitConn, []byte(magicCode))

	os.Exit(0)
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])

	fmt.Fprintf(os.Stderr, "%s [option] masterIp masterPort appName\n", os.Args[0])

	fmt.Fprintf(os.Stderr, "[option]\n")
	flag.PrintDefaults()
}

func main() {

	flag.Usage = printUsage

	var (
		verbose bool
		debug   bool
	)

	flag.BoolVar(&verbose, "v", true, "verbose info log")
	flag.BoolVar(&debug, "d", false, "verbose debug log")
	flag.Parse()

	if verbose == true {
		ILog.SetOutput(os.Stderr)
	}

	if debug == true {
		DLog.SetOutput(os.Stderr)
	}

	DLog.Printf("ARG0:%s, ARG1:%s, ARG2:%s", flag.Arg(0), flag.Arg(1), flag.Arg(2))

	masterIp := flag.Arg(0)
	masterPort, _ := strconv.Atoi(flag.Arg(1))
	appName := flag.Arg(2)

	/* set output log prefix */
	SetLogPrefix(appName)

	mainLoop(masterIp, masterPort, appName)
}
