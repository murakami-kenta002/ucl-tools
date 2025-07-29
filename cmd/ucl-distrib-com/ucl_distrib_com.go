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
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"time"
	"ucl-tools/internal/ucl"
	. "ucl-tools/internal/ulog"
)

func readStdinJson() string {

	dataChan := make(chan []byte, 1)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	go func() {
		data, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			ELog.Printf("app.json ReadAll error: %s \n", err)
			os.Exit(1)
		}
		var checkData interface{}
		err = json.Unmarshal(data, &checkData)
		if err != nil {
			ELog.Printf("Invalid app.json: %s\n", err)
			os.Exit(1)
		}
		dataChan <- data
		DLog.Println(string(data))
	}()

	select {
	case appJson := <-dataChan:
		return string(appJson)
	case <-ctx.Done():
		ELog.Printf("No app.json path specified in stdin\n")
		os.Exit(1)
	}

	return ""
}

func handleLcmConnection(targetAddr string) {

	appJson := readStdinJson()
	if appJson == "" {
		ELog.Println("readStdinJson err")
		os.Exit(1)
	}

	conn, err := ucl.ConnectTarget(targetAddr)
	if err != nil {
		ELog.Println("ConnectTarget: ", err)
		os.Exit(1)
	}
	ILog.Println("Dial connected to ", targetAddr)
	defer conn.Close()

	err = ucl.SendCommand(conn, ucl.CMD_DistribComm, string(appJson))
	if err != nil {
		ELog.Printf("sendCommand: ", err)
		os.Exit(1)
	}

	//wait for end message from LifeCycleManager.
	readBuf, err := ucl.ConnReadWithSize(conn)
	if err != nil {
		ILog.Printf("%s\n", err)
	} else {
		ILog.Printf("Notification from  LifeCycleManager : %s \n", readBuf)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage:\n")
	fmt.Fprintf(os.Stderr, " cat <app.json Path> | %s [OPTIONS]\n\n", os.Args[0])

	fmt.Fprintf(os.Stderr, "Description:\n")
	fmt.Fprintf(os.Stderr, " <app.json> must be supplied through stdin. \n\n")

	fmt.Fprintf(os.Stderr, "Options:\n")
	flag.PrintDefaults()
}

func main() {

	flag.Usage = printUsage

	var (
		verbose      bool
		debug        bool
		vScrnDefFile string
	)

	flag.BoolVar(&verbose, "v", true, "verbose info log")
	flag.BoolVar(&debug, "d", false, "verbose debug log")
	flag.StringVar(&vScrnDefFile, "f", ucl.VSCRNDEF_FILE, "virtual-screen-def.json file Path")
	flag.Parse()

	if verbose == true {
		ILog.SetOutput(os.Stderr)
	}

	if debug == true {
		DLog.SetOutput(os.Stderr)
	}

	vscrnDef, err := ucl.ReadVScrnDef(vScrnDefFile)
	if err != nil {
		ELog.Println("ReadVScrnDef error : ", err)
		os.Exit(1)
	}

	lcmAddr, err := ucl.GetLcmNodeAddr(vscrnDef)
	if err != nil {
		ELog.Println("GetLcmNodeAddr error : ", err)
		os.Exit(1)
	}

	handleLcmConnection(lcmAddr)
}
