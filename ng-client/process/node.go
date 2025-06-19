package process

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	log4plus "github.com/nextGPU/include/log4go"
	"github.com/nextGPU/ng-client/header"
	ws "github.com/nextGPU/ng-client/net/websocket"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	BaseHandle = uint64(1000)
	Version    = string("1.0.0.1")
)

const (
	MeasureID  = string("00000000") //Measurement Tasks
	DispenseID = string("00000001") //Workspace Task Distribution
)

const (
	//task
	TaskPublish            = int64(1) //cloud	-> client	||| client -> cloud
	TaskStop               = int64(2) //cloud	-> client	||| client -> cloud
	TaskDelete             = int64(3) //cloud	-> client	||| client -> cloud
	TaskProgress           = int64(4) //cloud	-> client	||| client -> cloud
	TaskCompleted          = int64(5) //cloud	-> client	||| client -> cloud
	TaskStart              = int64(6) //client-> cloud
	TaskFailed             = int64(7) //client -> cloud
	TaskInferenceCompleted = int64(8) //client -> cloud

	//workspace
	DispenseStart    = int64(11) //cloud -> client
	DispenseProgress = int64(12) //client-> cloud
	DispenseComplete = int64(13) //client-> cloud
	StartupWorkSpace = int64(14) //cloud -> client	||| client -> cloud
	StartupComplete  = int64(15) //client-> cloud
	DelWorkSpace     = int64(16) //cloud ->client	||| client -> cloud
	PullFail         = int64(17) //client-> cloud
	StartupFail      = int64(18) //client-> cloud
	DeleteContainer  = int64(19) //client-> cloud

	//measure
	MeasurePublish        = int64(21) //cloud -> client	||| client -> cloud
	MeasureStart          = int64(22) //client -> cloud
	MeasureComplete       = int64(23) //client -> cloud
	MeasureFailed         = int64(24)
	ComfyUIStartupSuccess = int64(26) //test protocol

	//configure
	OSSConfigure     = int64(31)
	MonitorConfigure = int64(32)

	//monitor
	NodeExporterData = int64(41)

	//heart
	NodePing = int64(99) //client->cloud
)

type MonitorConfiguration struct {
	MonitorType string `json:"monitor_type"`
	Monitor     string `json:"monitor"`
}

type Node struct {
	wsUrl         string
	SystemUUID    string
	SlotSize      int64
	State         header.NodeStatus
	Heart         header.NodeHeart
	OSS           header.OSS
	LogOSS        header.OSS
	Monitor       []MonitorConfiguration
	nodeExporter  *NodeExporter
	dockerLock    sync.Mutex
	Dockers       []*NodeDocker
	Client        *ws.WSClient
	CloudHandle   uint64
	OfflineCh     chan bool
	ExitCh        chan struct{}
	RecvMessageCh chan []byte
	SendMessageCh chan []byte
}

var gNode *Node

// interface
func (n *Node) GetClient() *ws.WSClient {
	return n.Client
}
func (n *Node) OSSEndpoint() string {
	return n.OSS.Endpoint
}
func (n *Node) OSSAccessKeyID() string {
	return n.OSS.AccessKeyID
}
func (n *Node) OSSAccessKeySecret() string {
	return n.OSS.AccessKeySecret
}
func (n *Node) OSSBucketName() string {
	return n.OSS.BucketName
}
func (n *Node) GetSystemUUID() string {
	return n.SystemUUID
}
func (n *Node) GetMeasureID() string {
	return MeasureID
}

func (n *Node) SetState(state header.NodeStatus) {
	n.State = state
}
func (n *Node) SendPublishTask(workspaceID, subID, session string, success bool, reason string, allSlots []header.Slot) {
	sendPublishTask(workspaceID, subID, session, success, reason, allSlots)
}
func (n *Node) SendMeasureTask(workspaceID, subID, session string, success bool, reason string, allSlots []header.Slot) {
	sendMeasureTask(workspaceID, subID, session, success, reason, allSlots)
}
func (n *Node) SendMeasureFailed(workspaceID, subID, failureReason string, resultData []byte, slot *header.Slot) {
	sendMeasureFailed(workspaceID, subID, failureReason, resultData, slot)
}
func (n *Node) SendMeasureComplete(workspaceID, subID string, outputs []*header.OSSOutput, resultData []byte, slot *header.Slot) {
	sendMeasureComplete(workspaceID, subID, outputs, resultData, slot)
}
func (n *Node) SendTaskFailed(workspaceID, subID, failureReason string, resultData []byte, allSlots []header.Slot) {
	sendTaskFailed(workspaceID, subID, failureReason, resultData, allSlots)
}
func (n *Node) SendTaskCompleted(workspaceID, subID string, outputs []*header.OSSOutput, resultData []byte, allSlots []header.Slot) {
	sendTaskCompleted(workspaceID, subID, outputs, resultData, allSlots)
}
func (n *Node) SendComfyUIStartupSuccess(workSpaceID string) {
	sendComfyUIStartupSuccess(workSpaceID)
}
func (n *Node) SendTaskStart(workspaceID, subID string, allSlots []header.Slot) {
	sendTaskStart(workspaceID, subID, allSlots)
}
func (n *Node) SendMeasureStart(workspaceID, subID string, allSlots []header.Slot) {
	sendMeasureStart(workspaceID, subID, allSlots)
}
func (n *Node) SendInferenceCompleted(workspaceID, subID string) {
	sendInferenceCompleted(workspaceID, subID)
}

func getFileSize(path string) (int64, error) {
	funName := "getFileSize"
	fi, err := os.Stat(path)
	if err != nil {
		errString := fmt.Sprintf("%s os.Stat failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return 0, errors.New(errString)
	}
	return fi.Size(), nil
}

func fileExists(fileName string) bool {
	_, err := os.Stat(fileName)
	if err == nil {
		return true
	}
	if errors.Is(err, os.ErrNotExist) {
		return false
	}
	return false
}

func (n *Node) RenameFile(saveDir, subFolder, fileName string) (error, string) {
	funName := "renameFile"
	imageUrl := ""
	if subFolder != "" {
		imageUrl = saveDir + subFolder + "/" + fileName
	} else {
		imageUrl = saveDir + "/" + fileName
	}
	log4plus.Info(fmt.Sprintf("%s --->>>[%s]", funName, imageUrl))
	ext := filepath.Ext(imageUrl)
	if fileExists(imageUrl) {
		newFileName := fmt.Sprintf("%s%s%s", time.Now().Format("20060102150405"), fmt.Sprintf("%06d", time.Now().Nanosecond()/1e3), ext)
		newFile := ""
		if subFolder != "" {
			newFile = saveDir + subFolder + "/" + newFileName
		} else {
			newFile = saveDir + "/" + newFileName
		}
		log4plus.Info(fmt.Sprintf("%s newFile=[%s]", funName, newFile))
		if errRename := os.Rename(imageUrl, newFile); errRename != nil {
			errString := fmt.Sprintf("%s Rename failed imageUrl=[%s] newFile=[%s] err=[%s]", funName, imageUrl, newFile, errRename.Error())
			log4plus.Error(errString)
			return errRename, ""
		}
		return nil, newFile
	}
	return errors.New("not found file"), ""
}

func (n *Node) PushOSS(ossDir, session, imagePath, filePath string) (string, int64, int64, error) {
	funName := "pushOSS"
	ossStartupTime := time.Now().UnixMilli()
	client, err := oss.New(n.OSS.Endpoint, n.OSS.AccessKeyID, n.OSS.AccessKeySecret,
		oss.Timeout(60, 120), // 连接60s，读写120s
		oss.EnableCRC(true),  // 开启CRC校验)
	)
	if err != nil {
		errString := fmt.Sprintf("%s oss.New failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return "", 0, 0, errors.New(errString)
	}
	bucket, err := client.Bucket(n.OSS.BucketName)
	if err != nil {
		errString := fmt.Sprintf("%s client.Bucket failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return "", 0, 0, errors.New(errString)
	}
	timeDir := ""
	sessionDir := imagePath
	objectKey := filepath.Join(
		timeDir,
		sessionDir,
		filepath.Base(filePath),
	)
	partSize := int64(100 * 1024 * 1024) // 100MB
	uploadOptions := []oss.Option{
		oss.Routines(3),          // 并发分片数
		oss.Checkpoint(true, ""), // 断点续传
	}
	if err = bucket.UploadFile(objectKey, filePath, partSize, uploadOptions...); err != nil {
		errString := fmt.Sprintf("%s PutObjectFromFile failed filePath=[%s] err=[%s]", funName, filePath, err.Error())
		log4plus.Error(errString)
		return "", 0, 0, errors.New(errString)
	}
	ossUrl := fmt.Sprintf("%s", objectKey)
	errString := fmt.Sprintf("%s UploadFile ossUrl=[%s]", funName, ossUrl)
	log4plus.Info(errString)
	ossEndTime := time.Now().UnixMilli()
	stampMilli := ossEndTime - ossStartupTime
	if fileSize, errFileSize := getFileSize(filePath); errFileSize != nil {
		log4plus.Error(fmt.Sprintf("%s getFileSize failed err=[%s]", funName, errFileSize.Error()))
		return ossUrl, stampMilli, 0, nil
	} else {
		return ossUrl, stampMilli, fileSize, nil
	}
}

// OnEvent
func (n *Node) OnConnect(linker *ws.Linker, param string, Ip string, port string) {
	n.CloudHandle = linker.Handle()
	n.State = header.WebSocketConnected
	sendOSSConfigure()
	sendMonitorConfigure()
}

func (n *Node) OnRead(linker *ws.Linker, param string, message []byte) error {
	n.RecvMessageCh <- message
	return nil
}

func (n *Node) OnDisconnect(linker *ws.Linker, param string, Ip string, port string) {
	n.State = header.Unavailable
	n.OfflineCh <- true
}

func (n *Node) Exist(param string) bool {
	return true
}

func (n *Node) findDocker(workspaceID string) *NodeDocker {
	n.dockerLock.Lock()
	defer n.dockerLock.Unlock()
	for _, v := range n.Dockers {
		if strings.EqualFold(v.WorkSpaceID, workspaceID) {
			return v
		}
	}
	return nil
}

func parseBase(data []byte) (header.BaseMessage, error) {
	var base header.BaseMessage
	err := json.Unmarshal(data, &base)
	return base, err
}

// recv message
func (n *Node) startCloudWS() {
	funName := "startCloudWS"
	n.State = header.WebSocketConnecting
	if linker := n.Client.StartClient(n.wsUrl, n.SystemUUID, n); linker != nil {
		log4plus.Info(fmt.Sprintf("%s StartClient success handle=[%d]", funName, linker.Handle()))
		n.State = header.WebSocketConnected
	} else {
		errString := fmt.Sprintf("%s StartClient Failed wsUrl=[%s] SystemUUID=[%s]", funName, n.wsUrl, n.SystemUUID)
		log4plus.Error(errString)
	}
}

func (n *Node) startWS() {
	funName := "startWS"
	n.State = header.WebSocketConnecting
	if cli := ws.NewWSClient(BaseHandle); cli != nil {
		n.Client = cli
	} else {
		errString := fmt.Sprintf("%s NewWSClient Failed", funName)
		log4plus.Error(errString)
	}
}

func (n *Node) reconnect() {
	if n.State == header.Unavailable {
		n.start()
	}
}

func (n *Node) offline() {
	funName := "offline"
	if n.Client != nil {
		if n.CloudHandle != 0 {
			log4plus.Info(fmt.Sprintf("%s n.CloudLinker.Close handle=[%d]", funName, n.CloudHandle))
			n.Client.Close(n.CloudHandle)
			n.State = header.Unavailable
			n.Monitor = n.Monitor[:0]
			for _, docker := range n.Dockers {
				docker.Stop()
			}
			n.Dockers = n.Dockers[:0]
		}
	}
}

func (n *Node) dispenseStart(message []byte) {
	funName := "dispenseStart"
	var msg header.DispenseMessage
	if err := json.Unmarshal(message, &msg); err != nil {
		errString := fmt.Sprintf("%s json.Unmarshal Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return
	}
	log4plus.Info(fmt.Sprintf("%s msg=[%#v]", funName, msg))
	array := header.DockerArray{
		WorkSpaceID: msg.WorkSpaceID,
		RunningMode: msg.RunningMode,
		MajorCMD:    msg.MajorCMD,
		MinorCMD:    msg.MinorCMD,
		ComfyUIHttp: msg.ComfyUIHttp,
		ComfyUIWS:   msg.ComfyUIWS,
		SaveDir:     msg.SaveDir,
		GPUMem:      "",
		Deleted:     msg.Deleted,
	}
	if docker := n.findDocker(msg.WorkSpaceID); docker != nil {
		docker.SetDocker(array)
	} else {
		docker = NewNodeDocker(msg.WorkSpaceID, n)
		n.Dockers = append(n.Dockers, docker)
		docker.SetDocker(array)
	}
}

func (n *Node) measure(message []byte) {
	funName := "measure"
	log4plus.Info(fmt.Sprintf("%s -------->>>>>>>> ", funName))
	var msg header.MeasureMessage
	if err := json.Unmarshal(message, &msg); err != nil {
		errString := fmt.Sprintf("%s json.Unmarshal Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return
	}
	log4plus.Info(fmt.Sprintf("-------->>>>>>>>%s WorkSpaceID=[%s] subID=[%s] data=[%s]", funName, msg.WorkSpaceID, msg.SubID, string(msg.Data)))
	if docker := n.findDocker(msg.WorkSpaceID); docker != nil {
		log4plus.Info(fmt.Sprintf("%s in findDocker WorkSpaceID=[%s]", funName, msg.WorkSpaceID))
		docker.MeasureCh <- &msg
	}
}

func (n *Node) publish(message []byte) {
	funName := "publish"
	var msg header.PublishMessage
	if err := json.Unmarshal(message, &msg); err != nil {
		errString := fmt.Sprintf("%s json.Unmarshal Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return
	}
	log4plus.Info(fmt.Sprintf("-------->>>>>>>>%s WorkSpaceID=[%s] subID=[%s] session=[%s]", funName, msg.WorkSpaceID, msg.SubID, msg.Session))
	if docker := n.findDocker(msg.WorkSpaceID); docker != nil {
		log4plus.Info(fmt.Sprintf("%s in findDocker WorkSpaceID=[%s]", funName, msg.WorkSpaceID))
		docker.TaskCh <- &msg
	}
}

func (n *Node) startupDocker(message []byte) {
	funName := "startupDocker"
	var msg header.StartupWorkSpaceMessage
	if err := json.Unmarshal(message, &msg); err != nil {
		errString := fmt.Sprintf("%s json.Unmarshal Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return
	}
	if docker := n.findDocker(msg.WorkSpaceID); docker != nil {
		docker.StartupDockerCh <- struct{}{}
		sendStartupWorkSpace(msg.WorkSpaceID)
	}
}

func (n *Node) deleteDocker(message []byte) {
	funName := "deleteDocker"
	msg := struct {
		MessageID   int64  `json:"messageID"`
		WorkSpaceID string `json:"workSpaceId"`
	}{}
	if err := json.Unmarshal(message, &msg); err != nil {
		errString := fmt.Sprintf("%s json.Unmarshal Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return
	}
	if docker := n.findDocker(msg.WorkSpaceID); docker != nil {
		docker.DelDockerCh <- msg.WorkSpaceID
		sendStopWorkSpace(msg.WorkSpaceID)
	}
}

func (n *Node) ossConfigure(message []byte) {
	funName := "ossConfigure"

	type OSSConfig struct {
		Endpoint        string `json:"endpoint"`
		AccessKeyID     string `json:"accessKeyID"`
		AccessKeySecret string `json:"accessKeySecret"`
		BucketName      string `json:"bucketName"`
	}
	oss := struct {
		MessageID int64     `json:"messageID"`
		SubId     string    `json:"subID"`
		AliOSS    OSSConfig `json:"aliOSS"`
		AliLogOSS OSSConfig `json:"aliLogOSS"`
	}{}
	if err := json.Unmarshal(message, &oss); err != nil {
		errString := fmt.Sprintf("%s json.Unmarshal Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return
	}
	n.OSS.Endpoint = oss.AliOSS.Endpoint
	n.OSS.AccessKeyID = oss.AliOSS.AccessKeyID
	n.OSS.AccessKeySecret = oss.AliOSS.AccessKeySecret
	n.OSS.BucketName = oss.AliOSS.BucketName

	n.LogOSS.Endpoint = oss.AliLogOSS.Endpoint
	n.LogOSS.AccessKeyID = oss.AliLogOSS.AccessKeyID
	n.LogOSS.AccessKeySecret = oss.AliLogOSS.AccessKeySecret
	n.LogOSS.BucketName = oss.AliLogOSS.BucketName
}

func (n *Node) monitorConfigure(message []byte) {
	funName := "monitorConfigure"
	monitor := struct {
		MessageID int64                  `json:"messageID"`
		Monitors  []MonitorConfiguration `json:"monitors"`
	}{}
	if err := json.Unmarshal(message, &monitor); err != nil {
		errString := fmt.Sprintf("%s json.Unmarshal Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return
	}
	log4plus.Info(fmt.Sprintf("%s monitor=[%#v]", funName, monitor))

	//var nodeExports []string
	//var gpuExports []string
	//for _, v := range monitor.Monitors {
	//	n.Monitor = append(n.Monitor, v)
	//	if strings.EqualFold(v.MonitorType, "node") {
	//		nodeExports = append(nodeExports, v.Monitor)
	//	} else if strings.EqualFold(v.MonitorType, "gpu") {
	//		gpuExports = append(gpuExports, v.Monitor)
	//	}
	//}
	//if n.nodeExporter == nil {
	//	n.nodeExporter = NewNodeExporter(nodeExports, n)
	//}
}

func (n *Node) nodePong(message []byte) {
	funName := "nodePong"
	pong := struct {
		MessageID int64                `json:"messageID"`
		Dockers   []header.DockerArray `json:"dockers"`
	}{}
	if err := json.Unmarshal(message, &pong); err != nil {
		errString := fmt.Sprintf("%s json.Unmarshal Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return
	}
	for _, v := range pong.Dockers {
		docker := n.findDocker(v.WorkSpaceID)
		if docker != nil {
			if v.Deleted == header.Deleted {
				if docker.Base.Deleted != header.Deleted {
					docker.DelDockerCh <- v.WorkSpaceID
					sendStopWorkSpace(v.WorkSpaceID)
				}
			}
		}
	}
}

func (n *Node) dispense(message []byte) {
	funName := "dispense"
	baseMessage, err := parseBase(message)
	if err != nil {
		errString := fmt.Sprintf("%s parseBase Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return
	}
	switch baseMessage.MessageID {
	case MeasurePublish:
		n.measure(message)
	case TaskPublish:
		n.publish(message)
	case DispenseStart:
		n.dispenseStart(message)
	case StartupWorkSpace:
		n.startupDocker(message)
	case DelWorkSpace:
		n.deleteDocker(message)
	case OSSConfigure:
		n.ossConfigure(message)
	case MonitorConfigure:
		n.monitorConfigure(message)
	case NodePing:
		n.nodePong(message)
	}
}

func (n *Node) recv() {
	for {
		select {
		case msg := <-n.SendMessageCh:
			sendMessage(msg)
		case msg := <-n.RecvMessageCh:
			n.dispense(msg)
		case <-n.OfflineCh:
			n.offline()
		case <-n.ExitCh:
			n.offline()
			return
		}
	}
}

func (n *Node) heart() {
	funName := "heart"
	if strings.Trim(n.Heart.Board, " ") == "" {
		_, n.Heart.Board = BoardInfo()
	}
	if strings.Trim(n.Heart.OS, " ") == "" {
		_, n.Heart.OS = OSInfo()
	}
	if strings.Trim(n.Heart.CPU, " ") == "" {
		_, n.Heart.CPU = CPUInfo()
	}
	_, n.Heart.Memory = MemoryInfo()
	gpus, err := GetGPUInfo()
	if err != nil {
		errString := fmt.Sprintf("%s GetGPUInfo Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		n.Heart.GPU = ""
		n.Heart.GpuDriver = ""
	} else {
		var gpuUUIDs []string
		for index, gpu := range gpus {
			gpuUUIDs = append(gpuUUIDs, fmt.Sprintf("GPU %d: %s (UUID: GPU-%s)", index, gpu.Model, gpu.UUID))
		}
		if len(gpuUUIDs) > 1 {
			n.Heart.GPU = strings.Join(gpuUUIDs, "\n")
		} else {
			n.Heart.GPU = gpuUUIDs[0]
		}
		n.Heart.GpuDriver = gpus[0].Driver
	}
	var arrays []header.DockerArray
	for _, docker := range n.Dockers {
		array := header.DockerArray{
			WorkSpaceID: docker.WorkSpaceID,
			RunningMode: docker.Base.RunningMode,
			State:       docker.State,
			ComfyUIHttp: docker.Base.ComfyUIHttp,
			ComfyUIWS:   docker.Base.ComfyUIWS,
			SaveDir:     docker.Base.SaveDir,
			Deleted:     docker.Base.Deleted,
		}
		arrays = append(arrays, array)
	}
	sendHeart(n.Heart.OS, n.Heart.CPU, n.Heart.GPU, n.Heart.GpuDriver, n.Heart.Version, n.Heart.Memory, arrays)
}

func (n *Node) sendTicker() {
	heartTicker := time.NewTicker(5 * time.Second)
	defer heartTicker.Stop()

	reconnectTicker := time.NewTicker(30 * time.Second)
	defer reconnectTicker.Stop()

	for {
		select {
		case <-heartTicker.C:
			if n.State == header.MeasurementComplete ||
				n.State == header.WebSocketConnected {
				n.heart()
			}
		case <-reconnectTicker.C:
			n.reconnect()
		case <-n.ExitCh:
			n.offline()
			return
		}
	}
}

func (n *Node) start() {
	funName := "start"
	time.Sleep(time.Duration(5) * time.Second)

	//检测
	_, os := OSInfo()
	_, cpu := CPUInfo()
	_, memory := MemoryInfo()
	gpus, err := GetGPUInfo()
	var gpu, gpuDriver string
	if err != nil {
		errString := fmt.Sprintf("%s GetGPUInfo Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		gpu = ""
		gpuDriver = ""
	} else {
		var gpuUUIDs []string
		for index, singleGPU := range gpus {
			gpuUUIDs = append(gpuUUIDs, fmt.Sprintf("GPU %d: %s (UUID: GPU-%s)", index, singleGPU.Model, singleGPU.UUID))
		}
		if len(gpuUUIDs) > 1 {
			gpu = strings.Join(gpuUUIDs, "\n")
		} else {
			gpu = gpuUUIDs[0]
		}
		gpuDriver = gpus[0].Driver
	}
	err, validateRes := sendValidate(n.SystemUUID, os, cpu, gpu, gpuDriver, memory)
	if err != nil {
		errString := fmt.Sprintf("%s sendValidate Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return
	}
	if !validateRes.AllowInstall {
		errString := fmt.Sprintf("%s sendValidate allowInstall=[%t]", funName, validateRes.AllowInstall)
		log4plus.Error(errString)
		return
	}

	//登录
	n.State = header.Enrolling
	err, res := sendEnroll(n.SystemUUID, os, cpu, gpu, gpuDriver, memory)
	if err != nil {
		n.State = header.Unavailable
		errString := fmt.Sprintf("%s sendEnroll Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return
	}
	n.State = header.Enrolled
	n.wsUrl = res.Connect
	n.startCloudWS()
	for _, v := range res.Dockers {
		docker := NewNodeDocker(v.WorkSpaceID, n)
		docker.SetDocker(v)
		n.Dockers = append(n.Dockers, docker)
	}
}

func (n *Node) Stop() {
	n.dockerLock.Lock()
	defer n.dockerLock.Unlock()
	for _, v := range n.Dockers {
		v.Stop()
	}
	close(n.ExitCh)
}

func (n *Node) WriteMessage(msg []byte) {
	if n.Client != nil {
		_ = n.Client.SendMessage(n.CloudHandle, msg)
	}
}

func SingletonNode() *Node {
	if gNode == nil {
		gNode = &Node{
			State:         header.Unavailable,
			ExitCh:        make(chan struct{}),
			OfflineCh:     make(chan bool),
			SendMessageCh: make(chan []byte, 1024),
			RecvMessageCh: make(chan []byte, 1024),
		}
		gNode.SystemUUID = GenerateEnvironmentID()
		go gNode.startWS()
		go gNode.recv()
		go gNode.sendTicker()
		go gNode.start()
	}
	return gNode
}
