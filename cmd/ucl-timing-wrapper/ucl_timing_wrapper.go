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
	"ucl-tools/internal/ucl"
	. "ucl-tools/internal/ulog"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
)

func timingSyncLaunch(masterIp string, masterPort string, appName string, execComm []string) {
	var err error
	var nCountCmd *exec.Cmd

	nCountCmd = exec.Command("ucl-ncount-worker", masterIp, masterPort, appName)
	nCountCmd.Stdout = os.Stdout
	nCountCmd.Stderr = os.Stderr
	err = nCountCmd.Run()
	if err != nil {
		ELog.Println("cmd.Run fail", err)
		os.Exit(1)
	}

	binary, err := exec.LookPath(execComm[0])
	if err != nil {
		ELog.Println("cmd.LookPath fail", err)
		os.Exit(1)
	}

	env := os.Environ()

	err = syscall.Exec(binary, execComm, env)
	if err != nil {
		ELog.Println("syscall.Exec fail", err)
		os.Exit(1)
	}
}

func timingAsyncLaunch(masterIp string, masterPort string, appName string, execComm []string) {
	var wg sync.WaitGroup
	var err error
	var appCmd, nCountCmd *exec.Cmd

	sigChan := make(chan os.Signal, 1)
	pidChan := make(chan int, 2)
	go ucl.SignalHandler(sigChan, pidChan, false)
	signal.Notify(sigChan,
		syscall.SIGINT,
		syscall.SIGTERM)

	DLog.Println("execComm=  :", execComm)
	appCmd = exec.Command(execComm[0], execComm[1:]...)
	appCmd.Stdout = os.Stdout
	appCmd.Stderr = os.Stderr
	err = appCmd.Start()
	if err != nil {
		ELog.Println("cmd.Start fail:", err)
		/* kill my self */
		syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
		goto WAIT_ALL
	}
	pidChan <- appCmd.Process.Pid
	wg.Add(1)
	go ucl.WaitApp(appCmd, &wg, execComm[0])

	nCountCmd = exec.Command("ucl-ncount-worker", masterIp, masterPort, appName)
	nCountCmd.Stdout = os.Stdout
	nCountCmd.Stderr = os.Stderr
	err = nCountCmd.Run()
	if err != nil {
		ELog.Println("cmd.Run fail")
		/* kill my self */
		syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
		goto WAIT_ALL
	}
WAIT_ALL:
	wg.Wait()
	ILog.Println("I'm finish")
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])

	fmt.Fprintf(os.Stderr, "%s [option] masterIp masterPort appName execComm\n", os.Args[0])

	fmt.Fprintf(os.Stderr, "[option]\n")
	flag.PrintDefaults()
}

func main() {

	flag.Usage = printUsage

	var (
		priorLaunch bool
		verbose     bool
		debug       bool
	)

	flag.BoolVar(&priorLaunch, "p", false, "Launch the wrapped command prior to the opposing wrapped commands")
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
	masterPort := flag.Arg(1)
	appName := flag.Arg(2)
	execCommArg := flag.Args()[3:]
	DLog.Println("ARG Command:", execCommArg)

	/* set output log prefix */
	SetLogPrefix(appName)

	if !priorLaunch {
		timingSyncLaunch(masterIp, masterPort, appName, execCommArg)
	} else {
		timingAsyncLaunch(masterIp, masterPort, appName, execCommArg)
	}
}
