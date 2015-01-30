package runner

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func kill(path string) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		// no process anymore, nothing to do
		return
	}
	oldPid, e := strconv.Atoi(string(b))
	if e != nil {
		panic("can't parse old pid " + e.Error())
	}

	cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("kill -s TERM %v", oldPid))
	_, errr := cmd.CombinedOutput()
	if errr == nil {
		//fmt.Println("old instance killed")
		log.Println("old instance killed")
		time.Sleep(time.Millisecond * 10)
	}
}

func write(path string) {
	pid := os.Getpid()
	err := ioutil.WriteFile(path, []byte(fmt.Sprintf("%v", pid)), os.FileMode(0644))
	if err != nil {
		panic("could not write PID file: " + err.Error())
	}
	//  fmt.Println("new pid file written")
}

func handleInterrupt(fn func()) {
	cmd := exec.Command("which", "go")
	out, err := cmd.Output()
	gobin := ""
	if err == nil {
		gobin = strings.TrimRight(string(out), "\n")
	} else {
		log.Println("can't find go executable, won't be able to sighup")
		return
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)
	signal.Notify(c, syscall.SIGHUP)
	go func() {
		for vl := range c {
			if vl == syscall.SIGHUP {
				if gobin == "" {
					log.Println("asked to reload, but can't find go binary")
				} else {
					log.Println("kindly asked to reload, cleaning up...")
					fn()
					log.Println("...finished cleaning, now restarting")
					er := syscall.Exec(gobin, []string{"go", "run", MainFile}, os.Environ())
					if er != nil {
						log.Println(er.Error())
					}
				}
				//fmt.Println("sighupped")

			} else {
				log.Println("kindly asked to shutdown, cleaning up...")
				fn()
				log.Println("...finished cleaning")
				os.Exit(0)

			}
		}
	}()
}
