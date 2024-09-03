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
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"ucl-tools/internal/ucl"
	. "ucl-tools/internal/ulog"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

type aliasIp struct {
	Alias map[string]ucl.UclNode `json:"alias"`
}

func retryConnectTarget(sockChan chan net.Conn, addr string) {
	for {
		conn, err := net.Dial("tcp", addr)
		if err == nil {
			sockChan <- conn
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func connectTarget(addr string) net.Conn {
	var conn net.Conn

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	sockChan := make(chan net.Conn, 1)

	go retryConnectTarget(sockChan, addr)

	select {
	case <-ctx.Done():
		ELog.Println("Dial cannot connect master")
		os.Exit(1)
	case conn = <-sockChan:
		ILog.Println("Dial connected to ", addr)
	}

	return conn
}

func readStdinJson(myAddr string) map[string]interface{} {

	bs, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		ELog.Printf("Json Command ReadAll error: %s \n", err)
		os.Exit(1)
	}

	jsonComm := string(bs)
	DLog.Println(jsonComm)

	mJson := make(map[string]interface{})
	err = json.Unmarshal([]byte(jsonComm), &mJson)
	if err != nil {
		ELog.Printf("Unmarshal json command error: %s \n", err)
		os.Exit(1)
	}

	return mJson
}

func readAliasIp(vscrnDef *ucl.VScrnDef) (aip aliasIp) {

	aip.Alias = make(map[string]ucl.UclNode)
	for _, r1 := range vscrnDef.Nodes {
		var uclNode ucl.UclNode
		uclNode.Ip = r1.Ip
		for _, r2 := range vscrnDef.DistrubutedWindowSystem.FrameworkNode {
			if r1.NodeId == r2.NodeId {
				uclNode.Port = r2.Ucl.Port
				break
			}
		}
		aip.Alias[r1.HostName] = uclNode
	}

	DLog.Printf("%+v\n", aip)

	return
}

func readAliasIpByFlag(args ucl.MultiFlag) (aip aliasIp) {

	aip.Alias = make(map[string]ucl.UclNode)

	for _, arg := range args {
		nodeInfo := strings.Split(arg, ":")
		var uclNode ucl.UclNode
		if len(nodeInfo) == 3 {
			hostname := nodeInfo[0]
			uclNode.Ip = nodeInfo[1]
			uclNode.Port, _ = strconv.Atoi(nodeInfo[2])
			aip.Alias[hostname] = uclNode
		}
	}

	return
}

func getMyIpAddrByVScrnDef(vscrnDef *ucl.VScrnDef) string {
	hostname, err := os.Hostname()
	if err != nil {
		ELog.Println("os.Hostname fail", err)
		os.Exit(1)
	}
	ipAddrs, err := ucl.GetIpAddrsOfAllInterfaces()
	if err != nil {
		ELog.Printf("GetIpAddrsOfAllInterfaces %s\n", err)
		os.Exit(1)
	}
	_, myAddr, err := ucl.GetNodeIdAndIpAddr(ipAddrs, hostname, vscrnDef)
	if err != nil {
		ELog.Println("GetNodeIdAndIpAddr error : ", err)
		os.Exit(1)
	}

	return myAddr
}

func isUsableIPAddress(ip net.IP) bool {
	if ip.IsLoopback() {
		return false
	}
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return false
	}
	if ip.IsMulticast() {
		return false
	}
	if ip.IsUnspecified() {
		return false
	}
	return true
}

func sendCommand(conn net.Conn, command_type string, command string) {
	magicBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(magicBuf, ucl.MagicCode)
	ucl.ConnWrite(conn, magicBuf)

	ucl.ConnWriteWithSize(conn, []byte(command_type))
	ucl.ConnWriteWithSize(conn, []byte(command))
}

func waitCommand(conn net.Conn) {
	readBuf, err := ucl.ConnRead(conn, 4)
	if err != nil {
		ELog.Println("ConnRead error: ", err)
		conn.Close()
		os.Exit(1)
	}
	readCode := binary.BigEndian.Uint32(readBuf)
	if readCode != ucl.MagicCode {
		ELog.Println("magicCode mismatch!")
		conn.Close()
		os.Exit(1)
	}
}

func chkConnection(localIp string, remoteAddr string) bool {

	localAddr := &net.TCPAddr{
		IP:   net.ParseIP(localIp),
		Port: 0,
	}
	dialer := net.Dialer{
		LocalAddr: localAddr,
		Timeout:   100 * time.Millisecond,
	}

	chkConn, err := dialer.Dial("tcp", remoteAddr)
	if err != nil {
		DLog.Printf("cannot connect to remoteIP:%s from localIP:%s\n", remoteAddr, localIp)
		return false
	}
	chkConn.Close()

	return true
}

func monitorTimeout(ctx context.Context, listener net.Listener) {
	select {
	case <-ctx.Done():
		listener.Close()
	}
}

func waitConnectionWithTimeout(listenAddr string) bool {

	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return false
	}
	defer listener.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	go monitorTimeout(ctx, listener)

	listenConn, err := listener.Accept()
	if err != nil {
		listener.Close()
		return false
	}
	listenConn.Close()

	return true
}

func chkBidirectConnection(localIp string, remoteAddr string) bool {

	if !chkConnection(localIp, remoteAddr) {
		return false
	}

	conn, err := net.Dial("tcp", remoteAddr)
	if err != nil {
		DLog.Printf("cannot connect from localIP:%s to remoteIP:%s\n", localIp, remoteAddr)
		return false
	}

	defer conn.Close()

	listenAddr := localIp + ":" + "8080"
	sendCommand(conn, "checkSocket", listenAddr)

	if !waitConnectionWithTimeout(listenAddr) {
		DLog.Printf("localIP:%s cannot get connection from remoteIP:%s", localIp, remoteAddr)
		return false
	}

	readBuf, err := ucl.ConnRead(conn, 4)
	if err != nil {
		return false
	}

	readCode := binary.BigEndian.Uint32(readBuf)
	if readCode != ucl.MagicCode {
		return false
	}

	return true
}

func findCandidateIps(localIps []string, remoteAddrs []string) []string {

	var candidateIps []string

	for _, localIp := range localIps {
		ip := net.ParseIP(localIp)
		if isUsableIPAddress(ip) {
			allConnected := true
			for _, remoteAddr := range remoteAddrs {
				if !chkBidirectConnection(localIp, remoteAddr) {
					allConnected = false
					break
				}
			}
			if allConnected {
				DLog.Printf("localIP:%s can communicate with all remoteIPs\n", localIp)
				candidateIps = append(candidateIps, localIp)
			}
		}
	}
	return candidateIps
}

func getMyIpAddr(args ucl.MultiFlag) string {

	var myAddr string

	localIps, err := ucl.GetIpAddrsOfAllInterfaces()
	if err != nil {
		ELog.Printf("GetIpAddrsOfAllInterfaces %s \n", err)
		os.Exit(1)
	}

	var remoteAddrs []string
	for _, arg := range args {
		nodeInfo := strings.Split(arg, ":")
		if len(nodeInfo) == 3 {
			remoteIp := nodeInfo[1]
			remotePort := nodeInfo[2]
			remoteAddrs = append(remoteAddrs, remoteIp+":"+remotePort)
		} else {
			ELog.Printf("the usage of -n option is incorrect\n")
			flag.Usage()
			os.Exit(1)
		}

	}

	candidateIps := findCandidateIps(localIps, remoteAddrs)

	if len(candidateIps) == 0 {
		ELog.Printf("cannot find my IP from interfaces\n")
		os.Exit(1)
	} else {
		myAddr = candidateIps[0]
	}
	ILog.Printf("my IP address is %s\n", myAddr)

	return myAddr
}

func handleTargetConnection(addr string, command string, wg *sync.WaitGroup) {
	defer wg.Done()

	conn := connectTarget(addr)
	defer conn.Close()

	sendCommand(conn, "launchApp", command)
	waitCommand(conn)
}

func chkConnectableNode(node ucl.UclNode, cChan chan bool) {

	addr := node.Ip + ":" + strconv.Itoa(node.Port)

	conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
	if err != nil {
		ELog.Printf("error:%s occured. %s not connectable\n", err, addr)

		cChan <- false
		return
	}

	cChan <- true

	conn.Close()
}

func getConnectableNode(dNodes []ucl.UclNode) (cNodes []ucl.UclNode) {

	numNodes := len(dNodes)

	cChans := make([]chan bool, numNodes)
	for i := range cChans {
		cChans[i] = make(chan bool, 1)
	}

	for i := 0; i < numNodes; i++ {
		go chkConnectableNode(dNodes[i], cChans[i])
	}

	for i := 0; i < numNodes; i++ {
		connectable := <-cChans[i]
		if connectable {
			cNodes = append(cNodes, dNodes[i])
		}
	}

	return
}

func removeDisconnectNode(mJson map[string]interface{}, cNodes []ucl.UclNode) {

	switch mJson["format_v1"].(map[string]interface{})["command_type"] {
	case "remote_virtio_gpu", "transport":
		sender := mJson["format_v1"].(map[string]interface{})["sender"].(map[string]interface{})
		chk := sender["launcher"].(ucl.UclNode)
		if !isExistNode(cNodes, chk) {
			ELog.Println("sender is not connectable")
			os.Exit(1)
		}

		var Recvs []interface{}

		if mJson["format_v1"].(map[string]interface{})["receivers"] != nil {
			receivers := mJson["format_v1"].(map[string]interface{})["receivers"].([]interface{})
			for _, r := range receivers {
				chk := r.(map[string]interface{})["launcher"].(ucl.UclNode)
				if isExistNode(cNodes, chk) {
					Recvs = append(Recvs, r)
				}
			}

			if len(Recvs) == 0 {
				ELog.Println("not exist connectable receivers")
				os.Exit(1)
			}

			mJson["format_v1"].(map[string]interface{})["receivers"] = Recvs
		}

	case "local":
		local := mJson["format_v1"].(map[string]interface{})["local"].(map[string]interface{})
		chk := local["launcher"].(ucl.UclNode)
		if !isExistNode(cNodes, chk) {
			ELog.Println("local is not connectable")
			os.Exit(1)
		}

	default:
		ELog.Println("not exist connectable receivers")
		os.Exit(1)
	}

	return
}

func isExistNode(dNodes []ucl.UclNode, chk ucl.UclNode) bool {
	for _, node := range dNodes {
		if node == chk {
			return true
		}
	}
	return false
}

func getDistribNode(mJson map[string]interface{}) (dNodes []ucl.UclNode) {

	switch mJson["format_v1"].(map[string]interface{})["command_type"] {
	case "remote_virtio_gpu", "transport":
		sender := mJson["format_v1"].(map[string]interface{})["sender"].(map[string]interface{})
		dNodes = append(dNodes, sender["launcher"].(ucl.UclNode))

		if mJson["format_v1"].(map[string]interface{})["receivers"] != nil {
			receivers := mJson["format_v1"].(map[string]interface{})["receivers"].([]interface{})
			for _, r := range receivers {
				chk := r.(map[string]interface{})["launcher"].(ucl.UclNode)
				if !isExistNode(dNodes, chk) {
					dNodes = append(dNodes, chk)
				}
			}
		}

	case "local":
		local := mJson["format_v1"].(map[string]interface{})["local"].(map[string]interface{})
		dNodes = append(dNodes, local["launcher"].(ucl.UclNode))

	default:
		ELog.Printf("command_type: %s is not supported \n", mJson["format_v1"].(map[string]interface{})["command_type"])
		os.Exit(1)
	}

	return
}

func getSenderReceiverNum(mJson map[string]interface{}) int {

	num := 0

	/* add for sender */
	num += 1

	if mJson["format_v1"].(map[string]interface{})["receivers"] != nil {
		receivers := mJson["format_v1"].(map[string]interface{})["receivers"].([]interface{})
		for range receivers {
			num += 1
		}
	}
	return num
}

func expandAliasNode(mJson map[string]interface{}, alias map[string]ucl.UclNode) {
	/* expand alias */
	var key string

	switch mJson["format_v1"].(map[string]interface{})["command_type"] {
	case "remote_virtio_gpu", "transport":
		sender := mJson["format_v1"].(map[string]interface{})["sender"].(map[string]interface{})
		switch sender["launcher"].(type) {
		case string:
			key = sender["launcher"].(string)
			sender["launcher"] = alias[key]
		}

		if mJson["format_v1"].(map[string]interface{})["receivers"] != nil {
			receivers := mJson["format_v1"].(map[string]interface{})["receivers"].([]interface{})
			for _, r := range receivers {
				receiver := r.(map[string]interface{})
				switch receiver["launcher"].(type) {
				case string:
					key = receiver["launcher"].(string)
					receiver["launcher"] = alias[key]
				}
			}
		}

	case "local":
		local := mJson["format_v1"].(map[string]interface{})["local"].(map[string]interface{})
		switch local["launcher"].(type) {
		case string:
			key = local["launcher"].(string)
			node, ok := alias[key]
			if ok {
				local["launcher"] = node
			} else {
				ELog.Printf("not found hostname %s from alias[hostname:{IP, Port}] \n", key)
				os.Exit(1)
			}
		}

	default:
		ELog.Printf("command_type: %s is not supported \n", mJson["format_v1"].(map[string]interface{})["command_type"])
		os.Exit(1)
	}

}

func launchNCountMaster(listenIp string, listenPort int, numWorker int, appName string) {
	var err error
	var nCountCmd *exec.Cmd

	nCountCmd = exec.Command("ucl-ncount-master", listenIp, strconv.Itoa(listenPort), strconv.Itoa(numWorker), appName)
	nCountCmd.Stdout = os.Stdout
	nCountCmd.Stderr = os.Stderr
	err = nCountCmd.Run()
	if err != nil {
		ELog.Println("cmd.Run fail", err)
		syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
	}
}

func launchNKeepMaster(masterIp string, masterPort int, numWorker int, appName string) {
	var wg sync.WaitGroup
	var err error
	var nKeepCmd *exec.Cmd

	sigChan := make(chan os.Signal, 1)
	pidChan := make(chan int, 2)
	go ucl.SignalHandler(sigChan, pidChan, false)
	signal.Notify(sigChan,
		syscall.SIGINT,
		syscall.SIGTERM)

	nKeepCmd = exec.Command("ucl-nkeep-master", masterIp, strconv.Itoa(masterPort), strconv.Itoa(numWorker), appName)
	nKeepCmd.Stdout = os.Stdout
	nKeepCmd.Stderr = os.Stderr
	err = nKeepCmd.Start()
	if err != nil {
		ELog.Println("cmd.Start fail:", err)
		/* kill my self */
		syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
		goto WAIT_ALL
	}
	pidChan <- nKeepCmd.Process.Pid
	wg.Add(1)
	go ucl.WaitApp(nKeepCmd, &wg, appName)

WAIT_ALL:
	wg.Wait()
	ILog.Println("I'm finish")
}

func mainLoop(myAddr string, aip aliasIp, force bool) {

	nfport, err := ucl.GetFreePort()
	if err != nil {
		ELog.Printf("GetFreePort %s\n", err)
		os.Exit(1)
	}

	tfport, err := ucl.GetFreePort()
	if err != nil {
		ELog.Printf("GetFreePort %s\n", err)
		os.Exit(1)
	}

	mJson := readStdinJson(myAddr)

	mJson["format_v1"].(map[string]interface{})["consistency_keep"] = ucl.UclNode{Ip: myAddr, Port: nfport}
	mJson["format_v1"].(map[string]interface{})["timing_control"] = ucl.UclNode{Ip: myAddr, Port: tfport}

	expandAliasNode(mJson, aip.Alias)

	if force {
		dNodes := getDistribNode(mJson)
		cNodes := getConnectableNode(dNodes)
		removeDisconnectNode(mJson, cNodes)
	}
	dNodes := getDistribNode(mJson)
	DLog.Println("connection target nodes:", dNodes)

	newJson, err := json.Marshal(&mJson)
	if err != nil {
		ELog.Printf("Marshal error: %s \n", err)
		os.Exit(1)
	}

	var targetAddrs []string
	for _, d := range dNodes {
		targetAddr := d.Ip + ":" + strconv.Itoa(d.Port)
		targetAddrs = append(targetAddrs, targetAddr)
	}

	var wg sync.WaitGroup

	for _, targetAddr := range targetAddrs {
		wg.Add(1)
		go handleTargetConnection(targetAddr, string(newJson), &wg)
	}

	wg.Wait()

	var nKeepNum int
	switch mJson["format_v1"].(map[string]interface{})["command_type"] {
	case "remote_virtio_gpu", "transport":
		nKeepNum = getSenderReceiverNum(mJson)
	default:
		nKeepNum = 1
	}

	appName, _ := mJson["format_v1"].(map[string]interface{})["appli_name"].(string)
	go launchNCountMaster(myAddr, tfport, nKeepNum, appName)

	launchNKeepMaster(myAddr, nfport, nKeepNum, appName)
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])

	fmt.Fprintf(os.Stderr, "%s [option] vScrnDefFile < json command\n", os.Args[0])

	fmt.Fprintf(os.Stderr, "[option]\n")
	flag.PrintDefaults()
}

func main() {

	flag.Usage = printUsage

	var (
		myAddr   string
		nodeInfo ucl.MultiFlag
		aip      aliasIp
		force    bool
		verbose  bool
		debug    bool
	)

	flag.BoolVar(&force, "f", false, "force the execution of the application even if some nodes are not alive.")
	flag.BoolVar(&verbose, "v", true, "verbose info log")
	flag.BoolVar(&debug, "d", false, "verbose debug log")
	flag.StringVar(&myAddr, "ip", "", "IP address of the device executing this command")
	flag.Var(&nodeInfo, "n", "node information (<hostname>:<ip>:<port>)")
	flag.Parse()

	if verbose == true {
		ILog.SetOutput(os.Stderr)
	}

	if debug == true {
		DLog.SetOutput(os.Stderr)
	}

	DLog.Printf("ARG0:%s\n", flag.Arg(0))

	if len(nodeInfo) != 0 {
		aip = readAliasIpByFlag(nodeInfo)
		if myAddr == "" {
			myAddr = getMyIpAddr(nodeInfo)
		}

	} else {
		vScrnDefFile := flag.Arg(0)
		if len(vScrnDefFile) == 0 {
			vScrnDefFile = ucl.VSCRNDEF_FILE
		}

		vscrnDef, err := ucl.ReadVScrnDef(vScrnDefFile)
		if err != nil {
			ELog.Println("ReadVScrnDef error : ", err)
			os.Exit(1)
		}

		aip = readAliasIp(vscrnDef)
		myAddr = getMyIpAddrByVScrnDef(vscrnDef)
	}

	mainLoop(myAddr, aip, force)
}
