package ucl

import (
	"context"
	"encoding/binary"
	"errors"
	. "ucl-tools/internal/ulog"
	"io"
	"net"
	"os"
	"strconv"
	"time"
)

var MagicCode uint32 = 0x55484d49 //'UHMI' ascii code

func retryConnectMaster(sockChan chan net.Conn, addr string) {
	for {
		conn, err := net.Dial("tcp", addr)
		if err == nil {
			sockChan <- conn
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func ConnWrite(conn net.Conn, message []byte) {
	_, err := conn.Write(message)
	if err != nil {
		ELog.Printf("Write error: %s \n", err)
		os.Exit(1)
	}
}

func ConnWriteWithSize(conn net.Conn, message []byte) {

	bufLen := uint32(len(message))
	szBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(szBuf, bufLen)
	ConnWrite(conn, szBuf)

	ConnWrite(conn, message)
}

func ConnRead(conn net.Conn, bufSize int) ([]byte, error) {
	buf := make([]byte, bufSize)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, err
	}
	return buf[:n], nil
}

func ConnReadWithSize(cbio io.Reader) ([]byte, error) {

	szBuf := make([]byte, 4)
	_, err := io.ReadFull(cbio, szBuf)
	if err != nil {
		return nil, errors.New("Command Size Read Fail: " + err.Error())
	}

	recvSize := binary.BigEndian.Uint32(szBuf)
	if recvSize == 0 {
		return nil, errors.New("zero byte read(meaningless)")
	} else if recvSize > 0x10000 {
		return nil, errors.New("recvSize: " + string(recvSize) + " is too huge as command size")
	}

	recvBuf := make([]byte, recvSize)
	_, err = io.ReadFull(cbio, recvBuf)
	if err != nil {
		return nil, errors.New("Command Read Fail: " + err.Error())
	}

	return recvBuf[:recvSize], nil
}

func WorkerConnWatchDog(waitTime time.Duration, regist chan int, numWorker int) {
	connected := 0

	t := time.NewTicker(waitTime * time.Second)
	defer t.Stop()

LOOP:
	for {
		select {
		case <-regist:
			connected++
			if connected >= numWorker {
				DLog.Printf("stop watch dog")
				break LOOP
			}
		case <-t.C:
			ELog.Printf("WatchDog Worker is insufficient(%d < %d)", connected, numWorker)
			os.Exit(1)
		}
	}
}

func ConnectMaster(masterIp string, masterPort int, magicCode []byte) net.Conn {
	var conn net.Conn

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	addr := masterIp + ":" + strconv.Itoa(masterPort)
	sockChan := make(chan net.Conn, 1)

	go retryConnectMaster(sockChan, addr)

	select {
	case <-ctx.Done():
		ELog.Println("Dial cannot connect master")
		os.Exit(1)
	case conn = <-sockChan:
		if magicCode != nil {
			ConnWriteWithSize(conn, magicCode)
		}
		ILog.Println("Dial connected to ", addr)
	}

	return conn
}
