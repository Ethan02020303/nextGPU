package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/beevik/ntp"
	"github.com/nextGPU/common"
	log4plus "github.com/nextGPU/include/log4go"
	"github.com/nextGPU/ng-update/configure"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
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

func fileMD5(filePath string) (string, error) {
	funName := "fileMD5"
	file, err := os.Open(filePath)
	if err != nil {
		log4plus.Error(fmt.Sprintf("%s Open Failed filePath=[%s] err=[%s]", funName, filePath, err.Error()))
		return "", err
	}
	defer file.Close()
	hash := md5.New()
	if _, errCopy := io.Copy(hash, file); errCopy != nil {
		log4plus.Error(fmt.Sprintf("%s Copy Success err=[%s]", funName, errCopy.Error()))
		return "", errCopy
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func PathExist(fullPath string) bool {
	_, err := os.Stat(fullPath)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}

func setLog() {
	logJson := "log.json"
	set := false
	if bExist := PathExist(logJson); bExist {
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

func syncTimeWithNTP() {
	funName := "syncTimeWithNTP"
	ntpServers := []string{"time.google.com", "pool.ntp.org", "ntp.aliyun.com"}
	var resp *ntp.Response
	var err error
	for _, server := range ntpServers {
		resp, err = ntp.QueryWithOptions(server, ntp.QueryOptions{Timeout: 3 * time.Second})
		if err == nil {
			break
		}
	}
	if err != nil {
		errString := fmt.Sprintf("%s All NTP servers are unavailable err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return
	}
	adjustedTime := time.Now().Add(resp.ClockOffset)
	log4plus.Info("%s NTP current time=[%s]", funName, adjustedTime.Format("2006-01-02 15:04:05.000"))
}

func httpPost(url string) (error, []byte) {
	funName := "httpPost"
	log4plus.Info(fmt.Sprintf("%s parse Url=%s", funName, url))
	client := &http.Client{
		Transport: &http.Transport{
			Dial: func(netw, addr string) (net.Conn, error) {
				c, err := net.DialTimeout(netw, addr, time.Minute*10)
				if err != nil {
					log4plus.Error(fmt.Sprintf("%s dail timeout err=%s", funName, err.Error()))
					return nil, err
				}
				return c, nil
			},
			MaxIdleConnsPerHost:   10,
			ResponseHeaderTimeout: time.Minute * 10,
		},
	}
	defer client.CloseIdleConnections()
	log4plus.Info(fmt.Sprintf("%s NewRequest Url=%s", funName, url))
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log4plus.Error(fmt.Sprintf("%s http.NewRequest Failed url=%s err=%s", funName, url, err.Error()))
		return err, []byte{}
	}
	response, err := client.Do(request)
	if err != nil {
		log4plus.Error(fmt.Sprintf("%s client.Do Failed url=%s err=%s", funName, url, err.Error()))
		return err, []byte{}
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		log4plus.Error(fmt.Sprintf("%s ioutil.ReadAll Failed url=%s err=%s", funName, url, err.Error()))
		return err, []byte{}
	}
	log4plus.Info(fmt.Sprintf("%s Check StatusCode response.StatusCode=%d", funName, response.StatusCode))
	if response.StatusCode != 200 {
		log4plus.Error(fmt.Sprintf("%s client.Do url=%s response.StatusCode=%d responseBody=%s", funName, url, response.StatusCode, string(responseBody)))
		return err, []byte{}
	}
	return nil, responseBody
}

type Md5FileContext struct {
	Url       string `json:"url"`
	CloudPath string `json:"cloudPath"`
	LocalPath string `json:"localPath"`
	MD5       string `json:"md5"`
	Start     bool   `json:"start"`
}

type Md5Files struct {
	Md5File []Md5FileContext `json:"files"`
}

type Response struct {
	CodeId     int64            `json:"codeId"`
	Msg        string           `json:"msg"`
	Version    string           `json:"version"`
	CreateTime string           `json:"createTime"`
	Md5File    []Md5FileContext `json:"files"`
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	return !os.IsNotExist(err)
}

func getMD5(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hasher := md5.New()
	buffer := make([]byte, 8*1024*1024) // 8MB 缓冲区
	for {
		bytesRead, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return "", err
		}
		if bytesRead == 0 {
			break
		}
		hasher.Write(buffer[:bytesRead])
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func parseFiles(body string) (error, []Md5FileContext) {
	funName := "parseFiles"
	var response Response
	err := json.Unmarshal([]byte(body), &response)
	if err != nil {
		log4plus.Error(fmt.Sprintf("%s json.Unmarshal error=[%s]", funName, err.Error()))
		return err, []Md5FileContext{}
	}
	if response.CodeId != 200 {
		log4plus.Error(fmt.Sprintf("%s response.CodeId is not 200 response.CodeId=[%d]", funName, response.CodeId))
		return err, []Md5FileContext{}
	}
	//获取当前目录
	var filePaths []Md5FileContext
	curDir, _ := os.Getwd()
	for index, file := range response.Md5File {
		log4plus.Info(fmt.Sprintf("%s index=[%d] filePath=[%s] md5=[%s]", funName, index, file.LocalPath, file.MD5))
		filePath := fmt.Sprintf("%s/%s", curDir, file.LocalPath)
		if !fileExists(filePath) {
			filePaths = append(filePaths, Md5FileContext{
				Url:       file.Url,
				CloudPath: file.CloudPath,
				LocalPath: file.LocalPath,
				MD5:       file.MD5,
				Start:     file.Start,
			})
		} else {
			md5File, errMd5 := getMD5(filePath)
			if errMd5 != nil {
				log4plus.Error(fmt.Sprintf("%s ioutil.ReadAll Failed filePath=[%s] errMd5=[%s]", funName, filePath, errMd5.Error()))
				filePaths = append(filePaths, Md5FileContext{
					Url:       file.Url,
					CloudPath: file.CloudPath,
					LocalPath: file.LocalPath,
					MD5:       file.MD5,
					Start:     file.Start,
				})
				if errMove := os.Remove(filePath); errMove != nil {
					log4plus.Error(fmt.Sprintf("%s Remove Failed filePath=[%s] errMove=[%s]", funName, filePath, errMove.Error()))
				}
				continue
			}
			if !strings.EqualFold(md5File, file.MD5) {
				filePaths = append(filePaths, Md5FileContext{
					Url:       file.Url,
					CloudPath: file.CloudPath,
					LocalPath: file.LocalPath,
					MD5:       file.MD5,
					Start:     file.Start,
				})
				if errMove := os.Remove(filePath); errMove != nil {
					log4plus.Error(fmt.Sprintf("%s Remove Failed filePath=[%s] errMove=[%s]", funName, filePath, errMove.Error()))
				}
			} else {
				filePaths = append(filePaths, Md5FileContext{
					Url:       file.Url,
					CloudPath: file.CloudPath,
					LocalPath: file.LocalPath,
					MD5:       file.MD5,
					Start:     file.Start,
				})
			}
		}
	}
	return nil, filePaths
}

func setWritable(path string) error {
	funName := "setWritable"
	// 修改权限
	if err := os.Chmod(path, 0777); err != nil {
		errString := fmt.Sprintf("%s Chmod Failed path=[%s] err=[%s]", funName, path, err.Error())
		log4plus.Error(errString)
		return errors.New(errString)
	}
	// 验证权限
	if info, err := os.Stat(path); err == nil {
		errString := fmt.Sprintf("%s Stat mode=[%o]", funName, info.Mode().Perm())
		log4plus.Info(errString)
	}
	return nil
}

func downloadFile(url, filePath string) error {
	funName := "downloadFile"
	resp, err := http.Get(url)
	if err != nil {
		log4plus.Error(fmt.Sprintf("%s httpPost url=[%s] err=[%s]", funName, url, err.Error()))
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		errString := fmt.Sprintf(fmt.Sprintf("%s httpPost resp.StatusCode=[%d]", funName, resp.StatusCode))
		log4plus.Error(errString)
		return errors.New(errString)
	}
	curDir, _ := os.Getwd()
	absFilePath := fmt.Sprintf("%s/%s", curDir, filePath)
	dirPath := filepath.Dir(absFilePath)
	// 递归创建目录（自动处理多级目录）
	if err = os.MkdirAll(dirPath, 0777); err != nil {
		errString := fmt.Sprintf(fmt.Sprintf("%s MkdirAll filePath=[%s] err=[%s]", funName, absFilePath, err.Error()))
		log4plus.Error(errString)
		return err
	}
	outFile, err := os.OpenFile(absFilePath, os.O_CREATE|os.O_WRONLY, 0777)
	if err != nil {
		errString := fmt.Sprintf(fmt.Sprintf("%s Create filePath=[%s] err=[%s]", funName, absFilePath, err.Error()))
		log4plus.Error(errString)
		return err
	}
	defer outFile.Close()
	_, err = io.Copy(outFile, resp.Body)
	if err = os.Chmod(absFilePath, 0777); err != nil {
		log.Fatal("权限设置失败:", err)
	}
	return nil
}

func check() {
	funName := "check"

	configure.SingletonConfigure()
	url := configure.SingletonConfigure().Application.UpdateUrl
	err, body := httpPost(url)
	if err != nil {
		log4plus.Error(fmt.Sprintf("%s httpPost url=[%s] err=[%s]", funName, configure.SingletonConfigure().Application.UpdateUrl, err.Error()))
		return
	}
	errParse, fileArray := parseFiles(string(body))
	if errParse != nil {
		log4plus.Error(fmt.Sprintf("%s parseFiles url=[%s] err=[%s]", funName, configure.SingletonConfigure().Application.UpdateUrl, errParse.Error()))
		return
	}
	success := true
	for _, file := range fileArray {
		err = downloadFile(file.Url, file.LocalPath)
		if err != nil {
			log4plus.Error(fmt.Sprintf("%s downloadFile url=[%s] err=[%s]", funName, file.Url, err.Error()))
			success = false
		}
	}
	curDir, _ := os.Getwd()
	log4plus.Info(fmt.Sprintf("%s os.Getwd curDir=[%s] success=[%t]", funName, curDir, success))
	if success {
		log4plus.Info(fmt.Sprintf("%s os.Getwd curDir=[%s] len(fileArray)=[%t]", funName, curDir, len(fileArray)))
		for _, file := range fileArray {
			log4plus.Info(fmt.Sprintf("%s os.Getwd LocalPath=[%s] file.Start=[%t]", funName, file.LocalPath, file.Start))
			if file.Start {
				filePath := fmt.Sprintf("%s/%s", curDir, file.LocalPath)
				log4plus.Info(fmt.Sprintf("%s exec.Command filePath=[%s]", funName, filePath))

				//方法1：使用exec库
				argv := []string{filePath}
				// 进程属性配置
				attr := &os.ProcAttr{
					Files: []*os.File{
						os.Stdin,
						os.Stdout,
						os.Stderr,
					},
					Sys: &syscall.SysProcAttr{
						// 可选：设置子进程组或会话特性[4](@ref)
					},
				}
				process, errProcess := os.StartProcess(filePath, argv, attr)
				if errProcess != nil {
					log4plus.Error(fmt.Sprintf("%s os.StartProcess err=[%s]", funName, errProcess.Error()))
					os.Exit(1)
				}
				log4plus.Info(fmt.Sprintf("%s os.StartProcess success pID=[%d]", funName, process.Pid))
				_ = process.Release()
			}
		}
	}
}

func main() {
	defer common.CrashDump()
	needReturn := flags.Check()
	if needReturn {
		return
	}
	setLog()
	//同步时间
	syncTimeWithNTP()
	configure.SingletonConfigure()
	//启动算力端后退出
	check()
	time.Sleep(time.Duration(10) * time.Second)
}
