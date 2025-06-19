package process

import (
	log4plus "github.com/nextGPU/include/log4go"
	"time"
)

type Process struct {
}

var gProcess *Process

func (p *Process) pollShow() {
	for {
		time.Sleep(time.Duration(10) * time.Second)
		log4plus.Info("--------------------------------------")
	}
}

func SingletonProcess() *Process {
	funName := "SingletonProcess"
	if gProcess == nil {
		gProcess = &Process{}
		log4plus.Info("%s start pollShow", funName)
		go gProcess.pollShow()
	}
	return gProcess
}
