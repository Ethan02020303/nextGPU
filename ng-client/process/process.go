package process

import (
	"errors"
	"fmt"
	log4plus "github.com/nextGPU/include/log4go"
	"os"
	"os/exec"
	"strings"
	"time"
)

type Process struct {
	ExitCh chan struct{}
	node   *Node
}

var gProcess *Process

func (p *Process) uploadOSSLog() {
	funName := "uploadOSSLog"
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 1, 0, 0, 0, now.Location())
	end := start.Add(10 * time.Minute)
	if !now.Before(start) && now.Before(end) {
		systemUUID := SingletonNode().Heart.Board
		log4plus.Info(fmt.Sprintf("%s log---->>>>OSS systemUUID=[%s] now=[%s]", funName, systemUUID, now.Format("2006-01-02 15:04:05")))
		if strings.Trim(systemUUID, " ") != "" {
			curDir, _ := os.Getwd()
			yesterday := time.Now().AddDate(0, 0, -1).Format("20060102")
			localDir := fmt.Sprintf("%s/log", curDir)
			localFileName := fmt.Sprintf("%s/%s.log", localDir, yesterday)
			ossFileName := fmt.Sprintf("%s.log", yesterday)
			err, exist := isLogExist(SingletonNode().LogOSS.Endpoint,
				SingletonNode().LogOSS.AccessKeyID,
				SingletonNode().LogOSS.AccessKeySecret,
				SingletonNode().LogOSS.BucketName,
				systemUUID, ossFileName)
			if err != nil {
				errString := fmt.Sprintf("%s isLogExist failed systemUUID=[%s] ossFileName=[%s] err=[%s]",
					funName, systemUUID, ossFileName, err.Error())
				log4plus.Error(errString)
				return
			}
			if exist {
				return
			}
			if err = pushLogOSS(SingletonNode().LogOSS.Endpoint,
				SingletonNode().LogOSS.AccessKeyID,
				SingletonNode().LogOSS.AccessKeySecret,
				SingletonNode().LogOSS.BucketName,
				localFileName, systemUUID, ossFileName); err != nil {
				errString := fmt.Sprintf("%s pushLogOSS failed systemUUID=[%s] localFileName=[%s] err=[%s]", funName, systemUUID, localFileName, err.Error())
				log4plus.Error(errString)
				return
			}
			log4plus.Info(fmt.Sprintf("%s pushLogOSS success systemUUID=[%s] localFileName=[%s]", funName, systemUUID, localFileName))
		}
	}
}

func (p *Process) checkUploadOSS() {
	uploadOSSTicker := time.NewTicker(5 * time.Minute)
	defer uploadOSSTicker.Stop()

	for {
		select {
		case <-uploadOSSTicker.C:
			p.uploadOSSLog()
		case <-p.ExitCh:
			return
		}
	}
}

func checkWSL() error {
	funName := "checkWSL"
	cmd := exec.Command("wsl", "--list", "--verbose")
	output, err := cmd.CombinedOutput()
	if err != nil {
		errString := fmt.Sprintf("%s exec.Command Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return errors.New(errString)
	}
	if !strings.Contains(string(output), "Running") {
		errString := fmt.Sprintf("%s wsl not running", funName)
		log4plus.Error(errString)
		return errors.New(errString)
	}
	return nil
}

func checkDocker() error {
	funName := "checkDocker"
	cmd := exec.Command("wsl", "docker", "info")
	if err := cmd.Run(); err != nil {
		errString := fmt.Sprintf("%s docker not found", funName)
		log4plus.Error(errString)
		return errors.New(errString)
	}
	return nil
}

func SingletonProcess() *Process {
	funName := "SingletonProcess"
	//if checkWSL() != nil || checkDocker() != nil {
	//	log4plus.Error(fmt.Sprintf("%s Runtime environment check failed. Please verify that the WSL or Docker environment is functioning properly ", funName))
	//	return nil
	//}

	if gProcess == nil {
		gProcess = &Process{}
		log4plus.Info(fmt.Sprintf("%s New Node", funName))

		gProcess.node = SingletonNode()
		go gProcess.checkUploadOSS()
	}
	return gProcess
}
