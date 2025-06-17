package main

import (
	"flag"
	"fmt"
	"github.com/nextGPU/common"
	log4plus "github.com/nextGPU/include/log4go"
	"github.com/nextGPU/ng-backend/configure"
	"github.com/nextGPU/ng-backend/db"
	"github.com/nextGPU/ng-backend/net/web"
	"github.com/nextGPU/ng-backend/process"
	"os"
	"path/filepath"
	"time"
)

const (
	Version = "1.0.0"
)

type Flags struct {
	Help    bool
	Version bool
}

func (f *Flags) Init() {
	flag.BoolVar(&f.Help, "h", false, "help")
	flag.BoolVar(&f.Version, "v", false, "show version")
}

func (f *Flags) Check() (needReturn bool) {
	flag.Parse()
	if f.Help {
		flag.Usage()
		needReturn = true
	} else if f.Version {
		verString := configure.SingletonConfigure().Application.Comment + " Version: " + Version + "\r\n"
		fmt.Println(verString)
		needReturn = true
	}
	return needReturn
}

var flags *Flags = &Flags{}

func init() {
	flags.Init()
}

func getExeName() string {
	ret := ""
	ex, err := os.Executable()
	if err == nil {
		ret = filepath.Base(ex)
	}
	return ret
}

func setLog() {
	logJson := "log.json"
	set := false
	if bExist := common.PathExist(logJson); bExist {
		if err := log4plus.SetupLogWithConf(logJson); err == nil {
			set = true
		}
	}
	if !set {
		fileWriter := log4plus.NewFileWriter()
		exeName := getExeName()
		_ = fileWriter.SetPathPattern("./log/" + exeName + "-%Y%M%D.log")
		log4plus.Register(fileWriter)
		log4plus.SetLevel(log4plus.DEBUG)
	}
}

func main() {
	defer common.CrashDump()
	needReturn := flags.Check()
	if needReturn {
		return
	}
	setLog()
	defer log4plus.Close()

	configure.SingletonConfigure()
	web.SingletonWeb()

	db.SingletonNodeBaseDB()
	db.SingletonTaskDB()
	process.SingletonProcess()

	for {
		time.Sleep(time.Duration(10) * time.Second)
	}
}
