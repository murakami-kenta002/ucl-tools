package ucl

import (
	. "ucl-tools/internal/ulog"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

func CheckProcessAlive(pid int) *os.Process {
	process, err := os.FindProcess(pid)
	if err != nil {
		DLog.Println("cannot find process: ", err)
		return nil
	}
	err = process.Signal(syscall.Signal(0))
	if err != nil {
		DLog.Printf("PID(%d) does not exist!\n", pid)
		return nil
	}

	return process
}

func killWhenAlive(pid int, sig os.Signal, sync bool) {
	process := CheckProcessAlive(pid)
	if process == nil {
		return
	} else {
		process.Signal(sig)
	}

	if sync {
		for {
			if CheckProcessAlive(pid) == nil {
				break
			}
		}
	}
}

func watchDogKill(pid int, sync bool) {
	killWhenAlive(pid, syscall.SIGTERM, sync)
	if !sync {
		time.Sleep(4 * time.Second)
	}
	killWhenAlive(pid, syscall.SIGKILL, sync)
}

func killAllChildren(children []int, sync bool) {
	for _, pid := range children {
		if sync {
			watchDogKill(pid, sync)
		} else {
			go watchDogKill(pid, sync)
		}
	}
}

func SignalHandler(sigChan chan os.Signal, pidChan chan int, sync bool) {

	var children []int

	for {
		select {
		case pid := <-pidChan:
			ILog.Printf("Append child[pid=%d]\n", pid)
			children = append(children, pid)
		case s := <-sigChan:
			switch s {
			case syscall.SIGINT:
				DLog.Println("SIGINT")
				killAllChildren(children, sync)
				children = children[:0]
			case syscall.SIGTERM:
				DLog.Println("SIGTERM")
				killAllChildren(children, sync)
				children = children[:0]
			}
		}
	}
}

func WaitApp(cmd *exec.Cmd, wg *sync.WaitGroup, appName string) {
	defer wg.Done()

	pid := cmd.Process.Pid
	DLog.Printf("wait process(app=%s pid:%d)\n", appName, pid)
	cmd.Wait()
	ILog.Printf("finish process(app=%s pid:%d)\n", appName, pid)

	/* kill my self */
	syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
}
