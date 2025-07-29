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
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"
	. "ucl-tools/internal/ulog"
)

var MagicCode uint32 = 0x55484d49 //'UHMI' ascii code
func SendMagicCode(conn net.Conn) error {

	magicBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(magicBuf, MagicCode)
	err := ConnWrite(conn, magicBuf)
	if err != nil {
		return err
	}

	return nil
}

func WaitMagicCode(conn net.Conn) error {

	recvMagicBuf, err := ConnRead(conn, 4)
	if err != nil {
		return err
	}

	recvCode := binary.BigEndian.Uint32(recvMagicBuf)
	if recvCode != MagicCode {
		return errors.New("magicCode mismatch!")
	}

	return nil
}

func ReadCommand(conn net.Conn) ([]byte, []byte, error) {

	err := WaitMagicCode(conn)
	if err != nil {
		return nil, nil, errors.New(fmt.Sprintf("WaitMagicCode : %s\n", err))
	}

	commType, err := ConnReadWithSize(conn)
	if err != nil {
		return nil, nil, errors.New(fmt.Sprintf("read command type err"))
	}

	command, err := ConnReadWithSize(conn)
	if err != nil {
		return nil, nil, errors.New(fmt.Sprintf("read command data err"))
	}

	return commType, command, nil
}

func SendCommand(conn net.Conn, commType string, command string) error {

	err := SendMagicCode(conn)
	if err != nil {
		return err
	}

	err = ConnWriteWithSize(conn, []byte(commType))
	if err != nil {
		return err
	}

	err = ConnWriteWithSize(conn, []byte(command))
	if err != nil {
		return err
	}

	return nil
}

func ReadStatus(conn net.Conn) ([]byte, error) {

	recvStatus, err := ConnReadWithSize(conn)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("read status err"))
	}

	return recvStatus, nil
}

func ResponseStatus(conn net.Conn, status string) error {

	err := ConnWriteWithSize(conn, []byte(status))
	if err != nil {
		return err
	}

	return nil
}

func ConnWrite(conn net.Conn, message []byte) error {
	n, err := conn.Write(message)
	if err != nil || n == 0 {
		if err != nil {
			ELog.Printf("Write error: %s \n", err)
		} else {
			ELog.Printf("Write error \n")
		}
		return errors.New(fmt.Sprintf("Write error %v\n", conn.(*net.TCPConn).RemoteAddr()))
	}
	return nil
}

func ConnWriteWithSize(conn net.Conn, message []byte) error {

	bufLen := uint32(len(message))
	szBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(szBuf, bufLen)
	err := ConnWrite(conn, szBuf)
	if err != nil {
		return err
	}
	err = ConnWrite(conn, message)
	if err != nil {
		return err
	}

	return err
}

func ConnReadLoop(conn net.Conn, rcvChan chan []byte) {
	for {
		buf, err := ConnReadWithSize(conn)
		if err != nil {
			rcvChan <- nil
			return
		}

		rcvChan <- buf
	}
}

func ConnRead(conn net.Conn, bufSize int) ([]byte, error) {

	buf := make([]byte, bufSize)
	total := 0
	for total < bufSize {
		readCount, err := conn.Read(buf[total:])
		if err != nil {
			return nil, errors.New("disconnected from " + conn.(*net.TCPConn).RemoteAddr().String())
		}
		if readCount == 0 {
			return nil, errors.New("message is EOF")
		}
		total += readCount
	}
	return buf, nil
}

func ConnReadWithSize(conn net.Conn) ([]byte, error) {

	szBuf, err := ConnRead(conn, 4)
	if err != nil {
		return nil, errors.New("Command Size Read Fail: " + err.Error())
	}

	recvSize := binary.BigEndian.Uint32(szBuf)
	if recvSize == 0 {
		return nil, errors.New("zero byte read(meaningless)")
	} else if recvSize > 0x10000 {
		return nil, errors.New("recvSize: " + strconv.Itoa(int(recvSize)) + " is too huge as command size")
	}
	recvBuf, err := ConnRead(conn, int(recvSize))
	if err != nil {
		return nil, errors.New("Command Read Fail: " + err.Error())
	}

	return recvBuf[:recvSize], nil
}

func retryConnectTarget(ctx context.Context, sockChan chan net.Conn, addr string) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			conn, err := net.Dial("tcp", addr)
			if err == nil {
				sockChan <- conn
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func ConnectTarget(addr string) (net.Conn, error) {
	var conn net.Conn

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	sockChan := make(chan net.Conn, 1)

	go retryConnectTarget(ctx, sockChan, addr)

	select {
	case <-ctx.Done():
		return nil, errors.New("Dial cannot connect " + addr)
	case conn = <-sockChan:
		return conn, nil
	}
}
