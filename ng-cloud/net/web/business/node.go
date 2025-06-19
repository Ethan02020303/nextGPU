package business

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/copier"
	log4plus "github.com/nextGPU/include/log4go"
	"github.com/nextGPU/ng-cloud/common"
	"github.com/nextGPU/ng-cloud/configure"
	"github.com/nextGPU/ng-cloud/db"
	"github.com/nextGPU/ng-cloud/header"
	"github.com/nextGPU/ng-cloud/process"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	NodeBase             = header.Base + 350
	RegisterError        = NodeBase + 1
	EnrollDBError        = NodeBase + 2
	NewNodeError         = NodeBase + 3
	NotFoundNodeError    = NodeBase + 4
	ConfigurationDBError = NodeBase + 5
	ValidateFailed       = NodeBase + 6
)

type NodeValidate struct {
	OS        string `json:"os,omitempty"`
	CPU       string `json:"cpu,omitempty"`
	GPU       string `json:"gpu,omitempty"`
	Memory    uint64 `json:"memory,omitempty"`
	GpuDriver string `json:"gpuDriver,omitempty"`
}

type Node struct {
	monitorConfigurations *db.MonitorConfigurations
}

var gNode *Node

func (w *Node) register(c *gin.Context) {
	funName := "register"
	clientIp := common.ClientIP(c)
	now := time.Now().UnixMilli()
	defer func() {
		log4plus.Info("%s clientIp=[%s] consumption time=%d(ms)", funName, clientIp, time.Now().UnixMilli()-now)
	}()
	request := struct {
		SystemUUID string `json:"systemUUID"`
		Account    string `json:"account"`
	}{}
	if err := c.BindJSON(&request); err != nil {
		errString := fmt.Sprintf("%s BindJSON Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		common.SendError(c, header.JsonParseError, errString)
		return
	}
	log4plus.Info("%s request=[%+v]", funName, request)
	if err := db.SingletonNodeBaseDB().NodeRegister(request.SystemUUID, request.Account, ""); err != nil {
		errString := fmt.Sprintf("%s CreateNode Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		common.SendError(c, RegisterError, errString)
		return
	}
	common.SendSuccess(c)
}

func (w *Node) enroll(c *gin.Context) {
	funName := "enroll"
	clientIp := common.ClientIP(c)
	request := struct {
		SystemUUID string `json:"systemUUID"`
		OS         string `json:"os"`
		CPU        string `json:"cpu"`
		GPU        string `json:"gpu"`
		Memory     uint64 `json:"memory"`
		GpuDriver  string `json:"gpuDriver"`
		Version    string `json:"version"`
	}{}
	var err error
	if err = c.BindJSON(&request); err != nil {
		errString := fmt.Sprintf("%s BindJSON Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		common.SendError(c, header.JsonParseError, errString)
		return
	}
	if strings.Trim(request.SystemUUID, " ") == "" {
		errString := fmt.Sprintf("%s request.SystemUuid is Empty", funName)
		log4plus.Error(errString)
		common.SendError(c, header.JsonParamNil, errString)
		return
	}
	log4plus.Info("%s request=[%+v]", funName, request)
	wsListen := strings.Replace(configure.SingletonConfigure().Net.Node.Listen, "0.0.0.0", configure.SingletonConfigure().Net.Node.Domain, 1)
	response := struct {
		CodeId  int64                 `json:"codeId"`
		Msg     string                `json:"msg"`
		Connect string                `json:"connect"`
		Dockers []process.DockerArray `json:"dockers"`
	}{
		CodeId:  http.StatusOK,
		Msg:     "success",
		Connect: fmt.Sprintf("ws://%s%s/%s", wsListen, configure.SingletonConfigure().Net.Node.RoutePath, request.SystemUUID),
	}
	node := process.SingletonNodes().FindNode(request.SystemUUID)
	if node == nil {
		err, _ = db.SingletonNodeBaseDB().WriteNode(request.SystemUUID, request.OS, request.CPU, request.GPU, request.GpuDriver, request.Memory)
		if err != nil {
			errString := fmt.Sprintf("%s LoadNode is not found SystemUuid=[%s]", funName, request.SystemUUID)
			log4plus.Error(errString)
			common.SendError(c, header.JsonParamNil, errString)
			return
		}
		if err = db.SingletonNodeBaseDB().NodeEnroll(request.SystemUUID, clientIp, db.Enrolling); err != nil {
			errString := fmt.Sprintf("%s SetNode Failed err=[%s]", funName, err.Error())
			log4plus.Error(errString)
			common.SendError(c, EnrollDBError, errString)
			return
		}
		err, node = process.SingletonNodes().AddNode(request.SystemUUID, clientIp)
		if err != nil {
			errString := fmt.Sprintf("%s NewNode Failed err=[%s]", funName, err.Error())
			log4plus.Error(errString)
			common.SendError(c, NewNodeError, errString)
			return
		}
		for _, v := range node.Base.Dockers {
			docker := process.DockerArray{
				WorkSpaceID:          v.WorkSpaceID,
				RunningMode:          v.RunningMode,
				State:                v.State,
				MajorCMD:             v.MajorCMD,
				MinorCMD:             v.MinorCMD,
				ComfyUIHttp:          v.ComfyUIHttp,
				ComfyUIWS:            v.ComfyUIWS,
				SaveDir:              v.SaveDir,
				GPUMem:               v.GPUMem,
				Deleted:              v.Deleted,
				MeasureTitle:         v.MeasureTitle,
				Measure:              v.Measure,
				MeasureConfiguration: v.MeasureConfiguration,
			}
			docker.Workflows = append(docker.Workflows, v.Workflows...)
			//Set the node's workspace state based on the workspace's overall status.
			workSpaces := process.SingletonWorkSpaces().AllWorkSpaces()
			for _, workSpace := range workSpaces {
				if strings.EqualFold(workSpace.Base.WorkspaceID, docker.WorkSpaceID) {
					if workSpace.Base.Deleted == db.Deleted {
						docker.Deleted = workSpace.Base.Deleted
						break
					}
				}
			}
			response.Dockers = append(response.Dockers, docker)
		}
	} else {
		node.State = db.Enrolled
		node.Base.HeartTime = time.Now()
		if node.Net != nil {
			_ = node.Net.Close()
			node.Net = nil
		}
		for _, v := range node.Base.Dockers {
			docker := process.DockerArray{
				WorkSpaceID:          v.WorkSpaceID,
				RunningMode:          v.RunningMode,
				State:                v.State,
				MajorCMD:             v.MajorCMD,
				MinorCMD:             v.MinorCMD,
				ComfyUIHttp:          v.ComfyUIHttp,
				ComfyUIWS:            v.ComfyUIWS,
				SaveDir:              v.SaveDir,
				GPUMem:               v.GPUMem,
				Deleted:              v.Deleted,
				Measure:              v.Measure,
				MeasureConfiguration: v.MeasureConfiguration,
			}
			response.Dockers = append(response.Dockers, docker)
		}
	}
	c.JSON(http.StatusOK, response)
}

func (w *Node) disconnect(c *gin.Context) {
	funName := "disconnect"
	clientIp := common.ClientIP(c)
	now := time.Now().UnixMilli()
	defer func() {
		log4plus.Info("%s clientIp=[%s] consumption time=%d(ms)", funName, clientIp, time.Now().UnixMilli()-now)
	}()
	systemUUID := c.DefaultQuery("systemUUID", "")
	log4plus.Info("%s DefaultQuery systemUUID=[%s]", funName, systemUUID)
	process.SingletonNodes().DelNode(systemUUID)
	common.SendSuccess(c)
}

func getGPU(checkGPU, gpu string) bool {
	rtxIndex := strings.Index(checkGPU, fmt.Sprintf(" %s ", gpu))
	if rtxIndex == -1 {
		return false
	}
	return true
}

func getOS(checkOS, os string) bool {
	rtxIndex := strings.Index(checkOS, os)
	if rtxIndex == -1 {
		return false
	}
	return true
}

func getCPU(checkCPU, cpu string) bool {
	rtxIndex := strings.Index(checkCPU, cpu)
	if rtxIndex == -1 {
		return false
	}
	return true
}

func getCPUCore(cpu string) int64 {
	keyword := "physical cores:"
	index := strings.Index(cpu, keyword)
	if index == -1 {
		return 0
	}
	subStr := cpu[index+len(keyword):]
	var numStr strings.Builder
	for _, c := range subStr {
		if c >= '0' && c <= '9' {
			numStr.WriteRune(c)
		} else if numStr.Len() > 0 {
			break // 遇到非数字时终止
		}
	}
	physicalCores, _ := strconv.Atoi(numStr.String())
	return int64(physicalCores)
}

func (w *Node) validate(c *gin.Context) {
	funName := "validate"
	clientIp := common.ClientIP(c)
	now := time.Now().UnixMilli()
	defer func() {
		log4plus.Info("%s clientIp=[%s] consumption time=%d(ms)", funName, clientIp, time.Now().UnixMilli()-now)
	}()
	systemUUID := c.DefaultQuery("systemUUID", "")
	log4plus.Info("%s DefaultQuery nodeAddr=[%s]", funName, systemUUID)
	var request NodeValidate
	if err := c.BindJSON(&request); err != nil {
		errString := fmt.Sprintf("%s BindJSON Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		common.SendError(c, header.JsonParseError, errString)
		return
	}
	log4plus.Info("%s request=[%v]", funName, request)

	response := struct {
		CodeId       int64  `json:"codeId"`
		Msg          string `json:"msg"`
		AllowInstall bool   `json:"allowInstall"`
	}{}
	//检测gpu是否符合要求
	if strings.Trim(request.GPU, " ") != "" {
		err, gpus := db.SingletonNodeBaseDB().GetAvailableGPU()
		if err != nil {
			errString := fmt.Sprintf("%s GetAvailableGPUs Failed err=[%s]", funName, err.Error())
			log4plus.Error(errString)
			common.SendError(c, header.DBError, errString)
			return
		}
		if len(gpus) > 0 {
			exist := false
			for _, gpu := range gpus {
				if getGPU(request.GPU, gpu) {
					exist = true
					break
				}
			}
			if !exist {
				errString := fmt.Sprintf("%s not found gpu=[%s]", funName, request.GPU)
				log4plus.Error(errString)
				response.CodeId = ValidateFailed
				response.Msg = errString
				response.AllowInstall = false
				c.JSON(http.StatusOK, response)
				return
			}
		}
	}
	//检测memory是否符合要求
	if request.Memory != 0 {
		if request.Memory < configure.SingletonConfigure().MinRequest.Memory {
			errString := fmt.Sprintf("%s Memory is less than required request.Memory=[%d] minRequest.Memory=[%d]",
				funName, request.Memory, configure.SingletonConfigure().MinRequest.Memory)
			log4plus.Error(errString)
			response.CodeId = ValidateFailed
			response.Msg = errString
			response.AllowInstall = false
			c.JSON(http.StatusOK, response)
			return
		}
	}
	//检测cpu是否符合要求
	if strings.Trim(request.CPU, " ") != "" {
		coreNum := getCPUCore(request.CPU)
		if coreNum < configure.SingletonConfigure().MinRequest.CoreNum {
			errString := fmt.Sprintf("%s Core Num is less than required request.CPU=[%s] coreNum=[%d] minRequest.CoreNum=[%d]",
				funName, request.CPU, coreNum, configure.SingletonConfigure().MinRequest.CoreNum)
			log4plus.Error(errString)
			response.CodeId = ValidateFailed
			response.Msg = errString
			response.AllowInstall = false
			c.JSON(http.StatusOK, response)
			return
		}
	}
	response.CodeId = http.StatusOK
	response.Msg = "success"
	response.AllowInstall = true
	c.JSON(http.StatusOK, response)
}

func (w *Node) monitorConfiguration(c *gin.Context) {
	funName := "monitorConfiguration"
	clientIp := common.ClientIP(c)
	now := time.Now().UnixMilli()
	defer func() {
		log4plus.Info("%s clientIp=[%s] consumption time=%d(ms)", funName, clientIp, time.Now().UnixMilli()-now)
	}()
	systemUUID := c.DefaultQuery("systemUUID", "")
	log4plus.Info("%s DefaultQuery nodeAddr=[%s]", funName, systemUUID)
	response := struct {
		CodeId       int64    `json:"codeId"`
		Msg          string   `json:"msg"`
		SystemUUID   string   `json:"systemUUID"`
		NodeMonitors []string `json:"nodeMonitors"`
		GPUMonitors  []string `json:"gpuMonitors"`
	}{
		CodeId:     http.StatusOK,
		Msg:        "success",
		SystemUUID: systemUUID,
	}
	if w.monitorConfigurations == nil {
		var err error
		err, w.monitorConfigurations = db.SingletonNodeBaseDB().Configuration()
		if err != nil {
			errString := fmt.Sprintf("%s monitorDB.Configuration Failed err=[%s]", funName, err.Error())
			log4plus.Error(errString)
			common.SendError(c, ConfigurationDBError, errString)
			return
		}
	}
	for _, v := range w.monitorConfigurations.Monitors {
		if strings.EqualFold(v.MonitorType, "node") {
			response.NodeMonitors = append(response.NodeMonitors, v.Monitor)
		} else if strings.EqualFold(v.MonitorType, "gpu") {
			response.GPUMonitors = append(response.GPUMonitors, v.Monitor)
		}
	}
	c.JSON(http.StatusOK, response)
}

func (w *Node) unPass(c *gin.Context) {
	funName := "unPass"
	clientIp := common.ClientIP(c)
	now := time.Now().UnixMilli()
	defer func() {
		log4plus.Info("%s clientIp=[%s] consumption time=%d(ms)", funName, clientIp, time.Now().UnixMilli()-now)
	}()
	systemUUID := c.DefaultQuery("systemUUID", "")
	log4plus.Info("%s DefaultQuery systemUUID=[%s]", funName, systemUUID)
	if strings.Trim(systemUUID, " ") != "" {
		if node := process.SingletonNodes().FindNode(systemUUID); node != nil {
			node.MeasureTime = time.Now()
			process.SingletonNodes().CleanIdleSlot(systemUUID)
			node.Match.ClearTasks()
			for _, v := range node.Base.Dockers {
				process.SingletonNodes().SetDockerStatus(systemUUID, v.WorkSpaceID, db.UnPassed)
				_ = db.SingletonNodeBaseDB().SetDockerStatus(systemUUID, v.WorkSpaceID, db.UnPassed)
			}
		}
		common.SendSuccess(c)
		return
	}
	common.SendError(c, header.JsonParamNil, "please check your parameters")
}

type (
	Md5FileContext struct {
		Url       string `json:"url"`
		CloudPath string `json:"cloudPath"`
		LocalPath string `json:"localPath"`
		MD5       string `json:"md5"`
		Start     bool   `json:"start"`
	}
	UpdateFiles struct {
		Version    string           `json:"version"`
		CreateTime string           `json:"createTime"`
		Md5File    []Md5FileContext `json:"files"`
	}
)

func (w *Node) update(c *gin.Context) {
	funName := "update"
	clientIp := common.ClientIP(c)
	now := time.Now().UnixMilli()
	defer func() {
		log4plus.Info("%s clientIp=[%s] consumption time=%d(ms)", funName, clientIp, time.Now().UnixMilli()-now)
	}()
	file, err := os.Open(configure.SingletonConfigure().ClientVersion.UpdateFile)
	if err != nil {
		errString := fmt.Sprintf("%s os.Open Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		common.SendError(c, header.OpenFileError, errString)
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		errString := fmt.Sprintf("%s io.ReadAll Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		common.SendError(c, header.OpenFileError, errString)
		return
	}
	var clientUpdate UpdateFiles
	if err = json.Unmarshal(content, &clientUpdate); err != nil {
		errString := fmt.Sprintf("%s json.Unmarshal Failed Error=[%s] content=[%s]", funName, err.Error(), string(content))
		log4plus.Error(errString)
		common.SendError(c, header.JsonParamNil, errString)
		return
	}
	response := struct {
		CodeId     int64            `json:"codeId"`
		Msg        string           `json:"msg"`
		Version    string           `json:"version"`
		CreateTime string           `json:"createTime"`
		Md5File    []Md5FileContext `json:"files"`
	}{
		CodeId:     http.StatusOK,
		Msg:        "success",
		Version:    clientUpdate.Version,
		CreateTime: clientUpdate.CreateTime,
	}
	_ = copier.Copy(&response.Md5File, &clientUpdate.Md5File)
	c.JSON(http.StatusOK, response)
}

func getAllFiles(rootPath string) ([]string, error) {
	funName := "getAllFiles"
	var files []string
	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			errString := fmt.Sprintf("%s filepath.Walk Failed err=[%s]", funName, err.Error())
			log4plus.Error(errString)
			return err
		}
		if !info.IsDir() { // 仅处理文件
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func computeMD5(filePath string) (string, error) {
	funName := "computeMD5"
	file, err := os.Open(filePath)
	if err != nil {
		errString := fmt.Sprintf("%s io.Open Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, errMD5 := io.Copy(hash, file); errMD5 != nil {
		errString := fmt.Sprintf("%s io.Copy Failed err=[%s]", funName, errMD5.Error())
		log4plus.Error(errString)
		return "", errMD5
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func getRelativePath(rootPath, absolutePath string) (string, error) {
	return filepath.Rel(rootPath, absolutePath)
}

func saveResults(results UpdateFiles, outputPath string) error {
	funName := "saveResults"
	file, err := os.Create(outputPath)
	if err != nil {
		errString := fmt.Sprintf("%s os.Create Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return err
	}
	defer file.Close()

	data, err := json.MarshalIndent(results, "", "  ") // 美化输出
	if err != nil {
		errString := fmt.Sprintf("%s json.MarshalIndent Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return err
	}
	if err = os.WriteFile(outputPath, data, 0644); err != nil {
		errString := fmt.Sprintf("%s os.WriteFile Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return err
	}
	return nil
}

func extractFilename(path string) string {
	return filepath.Base(path)
}

func createClientUpdate(version, createTime, md5FilePath string) (error, UpdateFiles) {
	funName := "createClientUpdate"
	// 获取所有文件
	files, err := getAllFiles(md5FilePath)
	if err != nil {
		errString := fmt.Sprintf("%s getAllFiles Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return errors.New(errString), UpdateFiles{}
	}
	// 计算MD5并保存结果
	results := make(map[string]string)
	for _, absolutePath := range files {
		// 计算相对路径
		relativePath, errPath := filepath.Rel(md5FilePath, absolutePath)
		if errPath != nil {
			errString := fmt.Sprintf("%s filepath.Rel Failed absolutePath=[%s] err=[%s]", funName, absolutePath, errPath.Error())
			log4plus.Error(errString)
			return errors.New(errString), UpdateFiles{}
		}
		// 计算MD5
		fileMD5, errMd5 := computeMD5(absolutePath)
		if errMd5 != nil {
			errString := fmt.Sprintf("%s computeMD5 Failed absolutePath=[%s] err=[%s]", funName, absolutePath, errMd5.Error())
			log4plus.Error(errString)
			return errors.New(errString), UpdateFiles{}
		}
		results[relativePath] = fileMD5
	}
	//生成内容
	var clientUpdate UpdateFiles
	clientUpdate.Version = version
	clientUpdate.CreateTime = createTime
	for key, fileMD5 := range results {
		fileContext := Md5FileContext{
			Url:       fmt.Sprintf("%s/%s/%s", configure.SingletonConfigure().ClientVersion.Domain, md5FilePath, key),
			CloudPath: fmt.Sprintf("%s/%s", md5FilePath, key),
			LocalPath: fmt.Sprintf("%s", key),
			MD5:       fileMD5,
			Start:     false,
		}
		fileName := extractFilename(fileContext.LocalPath)
		if strings.EqualFold(fileName, "ng_client") {
			fileContext.Start = true
		}
		clientUpdate.Md5File = append(clientUpdate.Md5File, fileContext)
	}
	return nil, clientUpdate
}

func (w *Node) generateUpdate(c *gin.Context) {
	funName := "generateUpdate"
	clientIp := common.ClientIP(c)
	now := time.Now().UnixMilli()
	defer func() {
		log4plus.Info("%s clientIp=[%s] consumption time=%d(ms)", funName, clientIp, time.Now().UnixMilli()-now)
	}()
	_, data := createClientUpdate(configure.SingletonConfigure().ClientVersion.Version, configure.SingletonConfigure().ClientVersion.CreateTime, "files")
	_ = saveResults(data, configure.SingletonConfigure().ClientVersion.UpdateFile)
	common.SendSuccess(c)
}

func (w *Node) Start(nodeGroup *gin.RouterGroup) {
	nodeGroup.POST("/register", w.register)
	nodeGroup.POST("/enroll", w.enroll)
	nodeGroup.POST("/validate", w.validate)
	nodeGroup.GET("/monitorConfiguration", w.monitorConfiguration)
	nodeGroup.GET("/disconnect", w.disconnect)
	nodeGroup.GET("/unPass", w.unPass)
	nodeGroup.GET("/update", w.update)
	nodeGroup.GET("/generateUpdate", w.generateUpdate)
}

func (w *Node) download(c *gin.Context) {
	funName := "download"
	clientIp := common.ClientIP(c)
	now := time.Now().UnixMilli()
	defer func() {
		log4plus.Info("%s clientIp=[%s] consumption time=%d(ms)", funName, clientIp, time.Now().UnixMilli()-now)
	}()
	requestedPath := strings.TrimPrefix(c.Param("filepath"), "/")
	cleanPath := filepath.Clean(requestedPath)
	curPath, _ := os.Getwd()
	fullPath := fmt.Sprintf("%s/files/%s", curPath, cleanPath)
	// 检查文件是否存在
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
		return
	}
	// 设置响应头并发送文件
	fileName := filepath.Base(fullPath)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", url.QueryEscape(fileName)))
	c.Header("Content-Type", "application/octet-stream")
	c.File(fullPath)
}

func (w *Node) FilesStart(filesGroup *gin.RouterGroup) {
	filesGroup.GET("/*filepath", w.download)
}

func SingletonNode() *Node {
	if gNode == nil {
		gNode = &Node{}
	}
	return gNode
}
