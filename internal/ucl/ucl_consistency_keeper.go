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

package ucl

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
	. "ucl-tools/internal/ulog"
)

type ConsistencyProtocol struct {
	CommType string `json:"type"`
	CommData string `json:"data"`
}

func NcountMaster(
	numWorker int,
	sendNodeChans []chan []byte,
	waitNCountChan chan int,
	commTaskCtx *CommTaskContext,
	wg *sync.WaitGroup) {

	defer wg.Done()

	ILog.Printf("Waiting for nCount from (%d) targets...\n", numWorker)

	connected := 0
	t := time.NewTicker(10 * time.Second)
	defer t.Stop()

LOOP:
	for {
		select {
		case <-commTaskCtx.Ctx.Done():
			ELog.Printf("close go routine")
			return

		case <-waitNCountChan:
			connected++
			if connected >= numWorker {
				break LOOP
			}
		case <-t.C:
			commTaskCtx.Cancel()
			ELog.Printf("WatchDog Worker is insufficient(%d < %d)", connected, numWorker)
			return
		}
	}

	cp := ConsistencyProtocol{
		CommType: "ncount",
		CommData: "",
	}
	sendMsg, _ := json.Marshal(cp)
	for _, sendNodeChan := range sendNodeChans {
		sendNodeChan <- sendMsg
	}

	ILog.Printf("nCount completed for (%d) targets.", numWorker)
}

func ncountWorker(
	sendLcmChan chan []byte,
	waitNCountChan chan []byte,
	commTaskCtx *CommTaskContext) error {

	rand.Seed(time.Now().UnixNano())

	magicCode := commTaskCtx.AppName + strconv.Itoa(rand.Intn(1024))
	DLog.Println("magicCode: ", magicCode)

	cp := ConsistencyProtocol{
		CommType: "ncount",
		CommData: magicCode,
	}
	sendMsg, _ := json.Marshal(cp)
	sendLcmChan <- sendMsg

	select {
	case recvMsg := <-waitNCountChan:
		if !reflect.DeepEqual(recvMsg, sendMsg) {
			ILog.Println("nCount reflect.DeepEqual ERR")
			commTaskCtx.Cancel()
			return errors.New(fmt.Sprintf("magic code mismatch"))
		}
	case <-commTaskCtx.Ctx.Done():
		return errors.New(fmt.Sprintf("other reason"))
	}

	ILog.Println("nCount replay OK from Master")
	return nil
}

func NkeepMaster(
	sendNodeChan chan []byte,
	waitNKeepChan chan int,
	commTaskCtx *CommTaskContext,
	wg *sync.WaitGroup) {

	defer wg.Done()

	retryInterval := 10 * time.Second
	timeoutInterval := 4 * time.Second

	sendt := time.NewTicker(retryInterval)
	recvt := time.NewTicker(timeoutInterval)
	defer sendt.Stop()
	defer recvt.Stop()

	cp := ConsistencyProtocol{
		CommType: "nkeep",
		CommData: "Check from Master",
	}
	sendMsg, _ := json.Marshal(cp)
	sendNodeChan <- sendMsg

	for {
		select {
		case <-commTaskCtx.Ctx.Done():
			return

		case <-sendt.C:
			sendNodeChan <- sendMsg
			recvt = time.NewTicker(timeoutInterval)

		case <-recvt.C:
			ELog.Printf("nkeepMaster CheckAlive Timeout: No response")
			commTaskCtx.Cancel()
			return

		case <-waitNKeepChan:
			recvt.Stop()
			sendt = time.NewTicker(retryInterval)
		}
	}
}

func NkeepWorker(
	sendLcmChan chan []byte,
	waitNKeepChan chan int,
	commTaskCtx *CommTaskContext,
	wg *sync.WaitGroup) error {

	defer wg.Done()

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "Unknown-Host"
	}

	cp := ConsistencyProtocol{
		CommType: "nkeep",
		CommData: "Response from " + commTaskCtx.AppName + " on " + hostname,
	}
	sendMsg, err := json.Marshal(cp)

	for {
		select {
		case <-commTaskCtx.Ctx.Done():
			ILog.Println("connectionLoop ERR")
			return nil

		case <-waitNKeepChan:
			sendLcmChan <- sendMsg
		}
	}

	return nil
}

func processExitHandler(
	pid int,
	commTaskCtx *CommTaskContext,
	wg *sync.WaitGroup) {

	defer wg.Done()

	for {
		select {
		case <-commTaskCtx.Ctx.Done():
			watchDogKill(pid, true)
			return
		}
	}
}

func waitProcess(
	cmd *exec.Cmd,
	commTaskCtx *CommTaskContext,
	wg *sync.WaitGroup) {

	defer wg.Done()

	pid := cmd.Process.Pid
	DLog.Printf("wait process(app=%s pid:%d)\n", commTaskCtx.AppName, pid)
	cmd.Wait()

	commTaskCtx.Cancel()
	ILog.Printf("finish process(app=%s pid:%d)\n", commTaskCtx.AppName, pid)
}

type ExecCommInfo struct {
	ExecComm    string
	ExecCommEnv []string
	IsWaitDeps  bool
}

func TimingWrapper(
	execCommInfo ExecCommInfo,
	sendLcmChan chan []byte,
	waitNCountChan chan []byte,
	commTaskCtx *CommTaskContext,
	wg *sync.WaitGroup) {

	defer wg.Done()

	var subWg sync.WaitGroup
	var err error

	if execCommInfo.IsWaitDeps {
		err = ncountWorker(sendLcmChan, waitNCountChan, commTaskCtx)
		if err != nil {
			return
		}
	}

	cmd := strings.Split(execCommInfo.ExecComm, " ")
	appCmd := exec.Command(cmd[0], cmd[1:]...)
	appCmd.Stdout = os.Stdout
	appCmd.Stderr = os.Stderr
	env := make([]string, len(os.Environ()))
	copy(env, os.Environ())
	appCmd.Env = env
	appCmd.Env = append(appCmd.Env, execCommInfo.ExecCommEnv...)
	err = appCmd.Start()
	if err != nil {
		ILog.Println("appCmd Start error: ", err)
		commTaskCtx.Cancel()
		return
	}

	subWg.Add(1)
	go processExitHandler(appCmd.Process.Pid, commTaskCtx, &subWg)

	subWg.Add(1)
	go waitProcess(appCmd, commTaskCtx, &subWg)

	if !execCommInfo.IsWaitDeps {
		ncountWorker(sendLcmChan, waitNCountChan, commTaskCtx)
	}

	subWg.Wait()
	if execCommInfo.IsWaitDeps {
		ILog.Println("timingSyncLaunch finish : ", commTaskCtx.AppName)
	} else {
		ILog.Println("timingAsyncLaunch finish : ", commTaskCtx.AppName)
	}
}
