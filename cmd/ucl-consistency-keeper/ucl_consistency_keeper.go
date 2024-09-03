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
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

func launchChildren(masterIp string, masterPort int, appName string, execComm string) {
	var wg sync.WaitGroup
	var err error
	var appCmd, nkeeperCmd *exec.Cmd

	sigChan := make(chan os.Signal, 1)
	pidChan := make(chan int, 2)
	go ucl.SignalHandler(sigChan, pidChan, false)
	signal.Notify(sigChan,
		syscall.SIGINT,
		syscall.SIGTERM)

	cmd := strings.Split(execComm, " ")
	appCmd = exec.Command(cmd[0], cmd[1:]...)
	appCmd.Stdout = os.Stdout
	appCmd.Stderr = os.Stderr
	err = appCmd.Start()
	if err != nil {
		ELog.Println("cmd.Start fail")
		/* kill my self */
		syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
		goto WAIT_ALL
	}
	pidChan <- appCmd.Process.Pid
	wg.Add(1)
	go ucl.WaitApp(appCmd, &wg, cmd[0])

	nkeeperCmd = exec.Command("ucl-nkeep-worker", masterIp, strconv.Itoa(masterPort), appName)
	nkeeperCmd.Stdout = os.Stdout
	nkeeperCmd.Stderr = os.Stderr
	err = nkeeperCmd.Start()
	if err != nil {
		ELog.Println("cmd.Start fail")
		/* kill my self */
		syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
		goto WAIT_ALL
	}
	pidChan <- nkeeperCmd.Process.Pid
	wg.Add(1)
	go ucl.WaitApp(nkeeperCmd, &wg, "ucl-nkeep-worker")

WAIT_ALL:
	wg.Wait()
	ILog.Println("I'm finish")
}

func mainLoop(masterIp string, masterPort int, appName string) {

	/* read execute command from stdin */
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	execComm := scanner.Text()
	DLog.Println(execComm)

	launchChildren(masterIp, masterPort, appName, execComm)
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
