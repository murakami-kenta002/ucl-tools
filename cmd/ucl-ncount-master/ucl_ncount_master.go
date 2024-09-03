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
	"sync"
)

func handleWorkerConnection(conn net.Conn, wg *sync.WaitGroup, waitChan chan int) {
	defer conn.Close()
	defer wg.Done()

	ILog.Printf("connected from %v\n", conn.(*net.TCPConn).RemoteAddr())

	cbio := bufio.NewReader(conn)
	buf, err := ucl.ConnReadWithSize(cbio)
	if err != nil {
		ELog.Println("ConnReadWithSize error: ", err)
		os.Exit(1)
	}
	DLog.Printf("read magicCode[%p] : %s", conn, string(buf))

	/* wait until all worker connected */
	<-waitChan

	/* write back magic code */
	ucl.ConnWriteWithSize(conn, buf)
}

func WorkerAcceptLoop(listenIp string, listenPort int, numWorker int) {

	regist := make(chan int, 2)
	go ucl.WorkerConnWatchDog(10, regist, numWorker)

	listenAddr := listenIp + ":" + strconv.Itoa(listenPort)

	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		ELog.Printf("Listen error: %s", err)
		os.Exit(1)
	}
	defer listener.Close()

	waitChans := make([]chan int, numWorker)
	for i := range waitChans {
		waitChans[i] = make(chan int, 1)
	}

	var wg sync.WaitGroup

	connected := 0

	for {
		DLog.Printf("listen: accepting current num = %d\n", connected)

		conn, err := listener.Accept()
		if err != nil {
			ELog.Printf("Accept error: %s", err)
			continue
		}

		DLog.Printf("accept:wake up current num = %d\n", connected)

		regist <- 1
		wg.Add(1)
		go handleWorkerConnection(conn, &wg, waitChans[connected])
		connected++

		if connected >= numWorker {
			ILog.Printf("all connected (%d, %d)", connected, numWorker)
			break
		}
	}

	for _, slvCh := range waitChans {
		slvCh <- 1
	}

	ILog.Printf("write back magic code to all worker")

	wg.Wait()

	os.Exit(0)
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
