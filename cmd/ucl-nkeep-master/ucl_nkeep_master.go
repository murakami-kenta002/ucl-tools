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
	"net"
	"os"
	"strconv"
	"time"
)

func watchDogWrite(conn net.Conn) {

	t := time.NewTicker(10 * time.Second)
	defer t.Stop()

	sendMsg := "Init Check from Master"
	ucl.ConnWriteWithSize(conn, []byte(sendMsg))

	for {
		select {
		case <-t.C:
			sendMsg := "Check from Master"
			ucl.ConnWriteWithSize(conn, []byte(sendMsg))
		}
	}
}

func handleWorkerConnection(conn net.Conn) {
	defer conn.Close()

	ILog.Printf("connected from %v\n", conn.(*net.TCPConn).RemoteAddr())

	go watchDogWrite(conn)

	for {
		cbio := bufio.NewReader(conn)
		buf, err := ucl.ConnReadWithSize(cbio)
		if err != nil {
			ELog.Printf("disconnected from %v\n", conn.(*net.TCPConn).RemoteAddr())
			os.Exit(1)
		} else if len(buf) == 0 {
			break
		}
		DLog.Printf(string(buf))
	}
}

func WorkerAcceptLoop(listenIp string, listenPort int, numWorker int) {

	regist := make(chan int, 2)
	go ucl.WorkerConnWatchDog(5, regist, numWorker)

	listenAddr := listenIp + ":" + strconv.Itoa(listenPort)

	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		ELog.Printf("Listen error: %s", err)
		os.Exit(1)
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			ELog.Printf("Accept error: %s", err)
		}
		regist <- 1
		go handleWorkerConnection(conn)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])

	fmt.Fprintf(os.Stderr, "%s [option] listenIp ListenPort numWorker appName\n", os.Args[0])

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

	DLog.Printf("ARG0:%s, ARG1:%s, ARG2:%s ARG3:%s", flag.Arg(0), flag.Arg(1), flag.Arg(2), flag.Arg(3))

	listenIp := flag.Arg(0)
	listenPort, _ := strconv.Atoi(flag.Arg(1))
	numWorker, _ := strconv.Atoi(flag.Arg(2))
	appName := flag.Arg(3)

	/* set output log prefix */
	SetLogPrefix(appName)

	WorkerAcceptLoop(listenIp, listenPort, numWorker)
}
