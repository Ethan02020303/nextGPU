package process

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/jinzhu/copier"
	log4plus "github.com/nextGPU/include/log4go"
	"github.com/nextGPU/include/prometheus"
	"github.com/nextGPU/ng-client/configure"
	"github.com/nextGPU/ng-client/header"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"time"
)

// send message
func httpPost(url string, body []byte) (error, []byte) {
	funName := "postHttp"
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

	bodyBytes := bytes.NewReader(body)
	log4plus.Info(fmt.Sprintf("%s NewRequest Url=%s", funName, url))
	request, err := http.NewRequest("POST", url, bodyBytes)
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

func sendMessage(message []byte) {
	SingletonNode().WriteMessage(message)
}

func sendEnroll(systemUUID, os, cpu, gpu, gpuDriver string, memory uint64) (error, header.NodeEnrollResponse) {
	funName := "sendEnroll"
	type NodeEnroll struct {
		SystemUUID string `json:"systemUUID"`
		OS         string `json:"os"`
		CPU        string `json:"cpu"`
		GPU        string `json:"gpu"`
		Memory     uint64 `json:"memory"`
		GpuDriver  string `json:"gpuDriver"`
		Version    string `json:"version"`
	}
	enroll := NodeEnroll{
		SystemUUID: systemUUID,
		OS:         os,
		CPU:        cpu,
		GPU:        gpu,
		Memory:     memory,
		GpuDriver:  gpuDriver,
		Version:    Version,
	}
	data, err := json.Marshal(enroll)
	if err != nil {
		errString := fmt.Sprintf("%s json.Marshal Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return err, header.NodeEnrollResponse{}
	}
	log4plus.Info(fmt.Sprintf("%s url=[%s] systemUUID=[%s]", funName, configure.SingletonConfigure().Web.Enroll, enroll.SystemUUID))
	err, body := httpPost(configure.SingletonConfigure().Web.Enroll, data)
	if err != nil {
		errString := fmt.Sprintf("%s httpPost Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return err, header.NodeEnrollResponse{}
	}
	var res header.NodeEnrollResponse
	if err := json.Unmarshal(body, &res); err != nil {
		errString := fmt.Sprintf("%s Unmarshal Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return err, header.NodeEnrollResponse{}
	}
	if res.CodeId != 200 {
		errString := fmt.Sprintf("%s enroll result error codeId=[%d] msg=[%s]", funName, res.CodeId, res.Msg)
		log4plus.Error(errString)
		return errors.New(errString), header.NodeEnrollResponse{}
	}
	return nil, res
}

func sendValidate(systemUUID, OS, CPU, GPU, GpuDriver string, Memory uint64) (error, header.NodeValidate) {
	funName := "sendValidate"
	validate := struct {
		OS        string `json:"os,omitempty"`
		CPU       string `json:"cpu,omitempty"`
		GPU       string `json:"gpu,omitempty"`
		Memory    uint64 `json:"memory,omitempty"`
		GpuDriver string `json:"gpuDriver,omitempty"`
	}{
		OS:        OS,
		CPU:       CPU,
		GPU:       GPU,
		Memory:    Memory,
		GpuDriver: GpuDriver,
	}
	data, err := json.Marshal(validate)
	if err != nil {
		errString := fmt.Sprintf("%s json.Marshal Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return err, header.NodeValidate{}
	}
	log4plus.Info(fmt.Sprintf("%s url=[%s] systemUUID=[%s]", funName, configure.SingletonConfigure().Web.Validate, systemUUID))
	url := fmt.Sprintf("%s?systemUUID=%s", configure.SingletonConfigure().Web.Validate, systemUUID)
	err, body := httpPost(url, data)
	if err != nil {
		errString := fmt.Sprintf("%s httpPost Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return err, header.NodeValidate{}
	}
	var res header.NodeValidate
	if err := json.Unmarshal(body, &res); err != nil {
		errString := fmt.Sprintf("%s Unmarshal Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return err, header.NodeValidate{}
	}
	if res.CodeId != 200 {
		errString := fmt.Sprintf("%s enroll result error codeId=[%d] msg=[%s]", funName, res.CodeId, res.Msg)
		log4plus.Error(errString)
		return errors.New(errString), header.NodeValidate{}
	}
	return nil, res
}

type Layer struct {
	LayerId  string `json:"layerId"`
	Progress string `json:"progress"`
}

func sendDispenseProgress(workspaceID string, layers []Layer) {
	funName := "sendDispenseProgress"
	msg := struct {
		MessageID   int64   `json:"messageID"`
		WorkSpaceID string  `json:"workSpaceId"`
		Layers      []Layer `json:"layers"`
	}{
		MessageID:   DispenseProgress,
		WorkSpaceID: workspaceID,
	}
	for _, layer := range layers {
		msg.Layers = append(msg.Layers, Layer{
			LayerId:  layer.LayerId,
			Progress: layer.Progress,
		})
	}
	data, err := json.Marshal(msg)
	if err != nil {
		errString := fmt.Sprintf("%s json.Marshal Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
	}
	SingletonNode().WriteMessage(data)
}

func sendDispenseComplete(workspaceID string, layers []Layer) {
	funName := "sendDispenseComplete"
	msg := struct {
		MessageID   int64   `json:"messageID"`
		WorkSpaceID string  `json:"workSpaceId"`
		SaveDir     string  `json:"saveDir"`
		Layers      []Layer `json:"layers"`
	}{
		MessageID:   DispenseComplete,
		WorkSpaceID: workspaceID,
	}
	for _, layer := range layers {
		msg.Layers = append(msg.Layers, Layer{
			LayerId:  layer.LayerId,
			Progress: layer.Progress,
		})
	}
	data, err := json.Marshal(msg)
	if err != nil {
		errString := fmt.Sprintf("%s json.Marshal Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return
	}
	SingletonNode().WriteMessage(data)
}

func sendPulling(workspaceID string) {
	msg := struct {
		MessageID   int64  `json:"messageID"`
		WorkSpaceID string `json:"workSpaceID"`
	}{
		MessageID:   DispenseStart,
		WorkSpaceID: workspaceID,
	}
	data, _ := json.Marshal(msg)
	SingletonNode().WriteMessage(data)
}

func sendPullFail(workspaceID string, fail string) {
	msg := struct {
		MessageID   int64  `json:"messageID"`
		WorkSpaceID string `json:"workSpaceId"`
		Failure     string `json:"failure"`
	}{
		MessageID:   PullFail,
		WorkSpaceID: workspaceID,
		Failure:     fail,
	}
	data, _ := json.Marshal(msg)
	SingletonNode().WriteMessage(data)
}

func sendStartupWorkSpace(workspaceID string) {
	msg := header.StartupWorkSpaceMessage{
		MessageID:   StartupWorkSpace,
		WorkSpaceID: workspaceID,
	}
	data, _ := json.Marshal(msg)
	SingletonNode().WriteMessage(data)
}

func sendStartupComplete(workspaceID string, success bool) {
	msg := struct {
		MessageID   int64  `json:"messageID"`
		WorkSpaceID string `json:"workSpaceID"`
		Success     bool   `json:"success"`
	}{
		MessageID:   StartupComplete,
		WorkSpaceID: workspaceID,
		Success:     success,
	}
	data, _ := json.Marshal(msg)
	SingletonNode().WriteMessage(data)
}

func sendStopWorkSpace(workspaceID string) {
	msg := struct {
		MessageID   int64  `json:"messageID"`
		WorkSpaceID string `json:"workSpaceID"`
	}{
		MessageID:   DelWorkSpace,
		WorkSpaceID: workspaceID,
	}
	data, _ := json.Marshal(msg)
	SingletonNode().WriteMessage(data)
}

func sendStartupFail(workspaceID string, fail string) {
	msg := struct {
		MessageID   int64  `json:"messageID"`
		WorkSpaceID string `json:"workSpaceID"`
		Failure     string `json:"failure"`
	}{
		MessageID:   StartupFail,
		WorkSpaceID: workspaceID,
		Failure:     fail,
	}
	data, _ := json.Marshal(msg)
	SingletonNode().WriteMessage(data)
}

func sendDeleteContainer(workspaceID string, fail string) {
	msg := struct {
		MessageID   int64  `json:"messageID"`
		WorkSpaceID string `json:"workSpaceID"`
		Failure     string `json:"failure"`
	}{
		MessageID:   DeleteContainer,
		WorkSpaceID: workspaceID,
		Failure:     fail,
	}
	data, _ := json.Marshal(msg)
	SingletonNode().WriteMessage(data)
}

func sendDeleteImage(workspaceID string) {
	msg := struct {
		MessageID   int64  `json:"messageID"`
		WorkSpaceID string `json:"workSpaceID"`
	}{
		MessageID:   DelWorkSpace,
		WorkSpaceID: workspaceID,
	}
	data, _ := json.Marshal(msg)
	SingletonNode().WriteMessage(data)
}

func sendPublishTask(workspaceID, subID, session string, success bool, reason string, allSlots []header.Slot) {
	msg := struct {
		MessageID   int64              `json:"messageID"`
		WorkSpaceID string             `json:"workSpaceID"`
		SubID       string             `json:"subID"`
		Session     string             `json:"session"`
		Success     bool               `json:"success"`
		Reason      string             `json:"reason"`
		Tasks       []header.QueueTask `json:"tasks"`
	}{
		MessageID:   TaskPublish,
		WorkSpaceID: workspaceID,
		SubID:       subID,
		Session:     session,
		Success:     success,
		Reason:      reason,
	}
	if success {
		for _, slot := range allSlots {
			msg.Tasks = append(msg.Tasks, header.QueueTask{
				SubID:       slot.SubID,
				InLocalPos:  slot.InQueuePos,
				InLocalTime: slot.InQueueTime,
				CurQueuePos: slot.CurQueuePos,
				StartTime:   slot.StartTime,
				EndTime:     slot.EndTime,
			})
		}
	}
	data, _ := json.Marshal(msg)
	SingletonNode().WriteMessage(data)
}

func sendMeasureTask(workspaceID, subID, session string, success bool, reason string, allSlots []header.Slot) {
	msg := struct {
		MessageID   int64              `json:"messageID"`
		WorkSpaceID string             `json:"workSpaceID"`
		SubID       string             `json:"subID"`
		Session     string             `json:"session"`
		Success     bool               `json:"success"`
		Reason      string             `json:"reason"`
		Tasks       []header.QueueTask `json:"tasks"`
	}{
		MessageID:   MeasurePublish,
		WorkSpaceID: workspaceID,
		SubID:       subID,
		Session:     session,
		Success:     success,
		Reason:      reason,
	}
	if success {
		for _, slot := range allSlots {
			msg.Tasks = append(msg.Tasks, header.QueueTask{
				SubID:       slot.SubID,
				InLocalPos:  slot.InQueuePos,
				InLocalTime: slot.InQueueTime,
				CurQueuePos: slot.CurQueuePos,
				StartTime:   slot.StartTime,
				EndTime:     slot.EndTime,
			})
		}
	}
	data, _ := json.Marshal(msg)
	SingletonNode().WriteMessage(data)
}

func sendMeasureStart(workspaceID, subID string, allSlots []header.Slot) {
	msg := struct {
		MessageID   int64              `json:"messageID"`
		WorkSpaceID string             `json:"workSpaceID"`
		SubID       string             `json:"subID"`
		Tasks       []header.QueueTask `json:"tasks"`
	}{
		MessageID:   MeasureStart,
		WorkSpaceID: workspaceID,
		SubID:       subID,
	}
	for _, slot := range allSlots {
		msg.Tasks = append(msg.Tasks, header.QueueTask{
			SubID:       slot.SubID,
			InLocalPos:  slot.InQueuePos,
			InLocalTime: slot.InQueueTime,
			CurQueuePos: slot.CurQueuePos,
			StartTime:   slot.StartTime,
			EndTime:     slot.EndTime,
		})
	}
	data, _ := json.Marshal(msg)
	SingletonNode().WriteMessage(data)
}

func sendMeasureFailed(workspaceID, subID, failureReason string, resultData []byte, slot *header.Slot) {
	msg := struct {
		MessageID     int64  `json:"messageID"`
		WorkSpaceID   string `json:"workSpaceID"`
		SubID         string `json:"subID"`
		FailureReason string `json:"failureReason"`
		ResultData    string `json:"resultData"`
		Duration      int64  `json:"duration"`
		StartTime     int64  `json:"startTime"`
		EndTime       int64  `json:"endTime"`
	}{
		MessageID:     MeasureFailed,
		WorkSpaceID:   workspaceID,
		SubID:         subID,
		FailureReason: failureReason,
		ResultData:    string(resultData),
		Duration:      slot.EndTime - slot.StartTime,
		StartTime:     slot.StartTime,
		EndTime:       slot.EndTime,
	}
	data, _ := json.Marshal(msg)
	SingletonNode().WriteMessage(data)
}

func sendMeasureComplete(workspaceID, subID string,
	outputs []*header.OSSOutput, resultData []byte, slot *header.Slot) {
	msg := struct {
		MessageID    int64    `json:"messageID"`
		WorkSpaceID  string   `json:"workSpaceID"`
		SubID        string   `json:"subID"`
		ImageUrls    []string `json:"imageUrls"`
		OSSDurations []int64  `json:"ossDurations"`
		ImageSizes   []int64  `json:"imageSizes"`
		ResultData   string   `json:"resultData"`
		Duration     int64    `json:"duration"`
		StartTime    int64    `json:"startTime"`
		EndTime      int64    `json:"endTime"`
	}{
		MessageID:   MeasureComplete,
		WorkSpaceID: workspaceID,
		SubID:       subID,
		ResultData:  string(resultData),
		Duration:    slot.EndTime - slot.StartTime,
		StartTime:   slot.StartTime,
		EndTime:     slot.EndTime,
	}
	for _, output := range outputs {
		for _, image := range output.Images {
			msg.ImageUrls = append(msg.ImageUrls, image.OSSUrl)
			msg.OSSDurations = append(msg.OSSDurations, image.OSSDuration)
			msg.ImageSizes = append(msg.ImageSizes, image.OSSSize)
		}
	}
	data, _ := json.Marshal(msg)
	SingletonNode().WriteMessage(data)
}

func sendTaskStart(workspaceID, subID string, allSlots []header.Slot) {
	funName := "sendTaskStart"
	msg := struct {
		MessageID   int64              `json:"messageID"`
		WorkSpaceID string             `json:"workSpaceID"`
		SubID       string             `json:"subID"`
		Tasks       []header.QueueTask `json:"tasks"`
	}{
		MessageID:   TaskStart,
		WorkSpaceID: workspaceID,
		SubID:       subID,
	}
	for _, slot := range allSlots {
		msg.Tasks = append(msg.Tasks, header.QueueTask{
			SubID:       slot.SubID,
			InLocalPos:  slot.InQueuePos,
			InLocalTime: slot.InQueueTime,
			CurQueuePos: slot.CurQueuePos,
			StartTime:   slot.StartTime,
			EndTime:     slot.EndTime,
		})
	}
	data, _ := json.Marshal(msg)
	SingletonNode().WriteMessage(data)
	errString := fmt.Sprintf("%s WriteMessage subID=[%s]", funName, subID)
	log4plus.Info(errString)
}

func sendInferenceCompleted(workspaceID, subID string) {
	msg := struct {
		MessageID   int64  `json:"messageID"`
		WorkSpaceID string `json:"workSpaceID"`
		SubID       string `json:"subID"`
	}{
		MessageID:   TaskInferenceCompleted,
		WorkSpaceID: workspaceID,
		SubID:       subID,
	}
	data, _ := json.Marshal(msg)
	SingletonNode().WriteMessage(data)
}

func sendTaskCompleted(workspaceID, subID string, outputs []*header.OSSOutput, resultData []byte, allSlots []header.Slot) {
	msg := struct {
		MessageID    int64              `json:"messageID"`
		WorkSpaceID  string             `json:"workSpaceID"`
		SubID        string             `json:"subID"`
		ImageUrls    []string           `json:"imageUrls"`
		OSSDurations []int64            `json:"ossDurations"`
		ImageSizes   []int64            `json:"imageSizes"`
		ResultData   string             `json:"resultData"`
		Tasks        []header.QueueTask `json:"tasks"`
	}{
		MessageID:   TaskCompleted,
		WorkSpaceID: workspaceID,
		SubID:       subID,
		ResultData:  string(resultData),
	}
	for _, output := range outputs {
		for _, image := range output.Images {
			msg.ImageUrls = append(msg.ImageUrls, image.OSSUrl)
			msg.OSSDurations = append(msg.OSSDurations, image.OSSDuration)
			msg.ImageSizes = append(msg.ImageSizes, image.OSSSize)
		}
	}
	for _, slot := range allSlots {
		msg.Tasks = append(msg.Tasks, header.QueueTask{
			SubID:       slot.SubID,
			InLocalPos:  slot.InQueuePos,
			InLocalTime: slot.InQueueTime,
			CurQueuePos: slot.CurQueuePos,
			StartTime:   slot.StartTime,
			EndTime:     slot.EndTime,
		})
	}
	data, _ := json.Marshal(msg)
	SingletonNode().WriteMessage(data)
}

func sendTaskFailed(workspaceID, subID, failureReason string, resultData []byte, allSlots []header.Slot) {
	msg := struct {
		MessageID     int64              `json:"messageID"`
		WorkSpaceID   string             `json:"workSpaceID"`
		SubID         string             `json:"subID"`
		ResultData    string             `json:"resultData"`
		FailureReason string             `json:"failureReason"`
		Tasks         []header.QueueTask `json:"tasks"`
	}{
		MessageID:     TaskFailed,
		WorkSpaceID:   workspaceID,
		SubID:         subID,
		ResultData:    string(resultData),
		FailureReason: failureReason,
	}
	for _, slot := range allSlots {
		msg.Tasks = append(msg.Tasks, header.QueueTask{
			SubID:       slot.SubID,
			InLocalPos:  slot.InQueuePos,
			InLocalTime: slot.InQueueTime,
			CurQueuePos: slot.CurQueuePos,
			StartTime:   slot.StartTime,
			EndTime:     slot.EndTime,
		})
	}
	data, _ := json.Marshal(msg)
	SingletonNode().WriteMessage(data)
}

func sendComfyUIStartupSuccess(workSpaceID string) {
	msg := struct {
		MessageID   int64  `json:"messageID"`
		WorkSpaceID string `json:"workSpaceID"`
	}{
		MessageID:   ComfyUIStartupSuccess,
		WorkSpaceID: workSpaceID,
	}
	data, _ := json.Marshal(msg)
	SingletonNode().WriteMessage(data)
}

func sendHeart(os, cpu, gpu, gpuDriver, version string, memory uint64, docker []header.DockerArray) {
	msg := struct {
		MessageID int64                `json:"messageID"`
		OS        string               `json:"os"`
		Board     string               `json:"board"`
		CPU       string               `json:"cpu"`
		GPU       string               `json:"gpu"`
		Memory    uint64               `json:"memory"`
		Version   string               `json:"version"`
		GpuDriver string               `json:"gpuDriver"`
		Dockers   []header.DockerArray `json:"dockers"`
	}{
		MessageID: NodePing,
		OS:        os,
		CPU:       cpu,
		GPU:       gpu,
		Memory:    memory,
		Version:   version,
		GpuDriver: gpuDriver,
	}
	for _, v := range docker {
		array := header.DockerArray{
			WorkSpaceID: v.WorkSpaceID,
			RunningMode: v.RunningMode,
			State:       v.State,
			MajorCMD:    v.MajorCMD,
			MinorCMD:    v.MinorCMD,
			ComfyUIHttp: v.ComfyUIHttp,
			ComfyUIWS:   v.ComfyUIWS,
			SaveDir:     v.SaveDir,
			GPUMem:      v.GPUMem,
		}
		msg.Dockers = append(msg.Dockers, array)
	}
	data, _ := json.Marshal(msg)
	SingletonNode().WriteMessage(data)
}

func sendOSSConfigure() {
	msg := struct {
		MessageID int64 `json:"messageID"`
	}{
		MessageID: OSSConfigure,
	}
	data, _ := json.Marshal(msg)
	SingletonNode().WriteMessage(data)
}

func sendMonitorConfigure() {
	msg := struct {
		MessageID int64 `json:"messageID"`
	}{
		MessageID: MonitorConfigure,
	}
	data, _ := json.Marshal(msg)
	SingletonNode().WriteMessage(data)
}

func sendNodeExporter(msg *prometheus.ExporterData) {
	exporterData := struct {
		MessageID        int64                   `json:"messageID"`
		NodeExporterData prometheus.ExporterData `json:"exporterData"`
	}{
		MessageID: NodeExporterData,
	}
	_ = copier.CopyWithOption(&exporterData.NodeExporterData, msg, copier.Option{DeepCopy: true})
	data, _ := json.Marshal(exporterData)
	SingletonNode().WriteMessage(data)
}

func pushLogOSS(endpoint, accessKeyID, accessKeySecret, bucketName, localFilePath, ossPath, logName string) error {
	funName := "pushLogOSS"
	// 1. 创建OSS客户端
	client, err := oss.New(endpoint, accessKeyID, accessKeySecret,
		oss.Timeout(60, 120), // 连接60s，读写120s
		oss.EnableCRC(true),  // 开启CRC校验)
	)
	if err != nil {
		errString := fmt.Sprintf("%s oss.New failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return errors.New(errString)
	}
	//2. 获取Bucket
	bucket, err := client.Bucket(bucketName)
	if err != nil {
		errString := fmt.Sprintf("%s client.Bucket failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return errors.New(errString)
	}
	// 3. 检查文件是否存在
	ossFilePath := fmt.Sprintf("%s/%s", ossPath, logName)
	isExist, err := bucket.IsObjectExist(ossFilePath)
	if err != nil {
		errString := fmt.Sprintf("%s bucket.IsObjectExist failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return errors.New(errString)
	}
	if isExist {
		errString := fmt.Sprintf("%s isExist is true ossFilePath=[%s]", funName, ossFilePath)
		log4plus.Error(errString)
		return errors.New(errString)
	}
	//4. 文件不存在，执行上传
	timeDir := ""
	sessionDir := ossPath
	objectKey := filepath.Join(
		timeDir,
		sessionDir,
		filepath.Base(logName),
	)
	partSize := int64(100 * 1024 * 1024) // 100MB
	uploadOptions := []oss.Option{
		oss.Routines(3),          // 并发分片数
		oss.Checkpoint(true, ""), // 断点续传
	}
	if err = bucket.UploadFile(objectKey, localFilePath, partSize, uploadOptions...); err != nil {
		errString := fmt.Sprintf("%s PutObjectFromFile failed localFilePath=[%s] err=[%s]", funName, localFilePath, err.Error())
		log4plus.Error(errString)
		return errors.New(errString)
	}
	ossUrl := fmt.Sprintf("%s", objectKey)
	errString := fmt.Sprintf("%s UploadFile ossUrl=[%s]", funName, ossUrl)
	log4plus.Info(errString)
	return nil
}

func isLogExist(endpoint, accessKeyID, accessKeySecret, bucketName, ossPath, logName string) (error, bool) {
	funName := "pushLogOSS"
	// 1. 创建OSS客户端
	client, err := oss.New(endpoint, accessKeyID, accessKeySecret,
		oss.Timeout(60, 120), // 连接60s，读写120s
		oss.EnableCRC(true),  // 开启CRC校验)
	)
	if err != nil {
		errString := fmt.Sprintf("%s oss.New failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return errors.New(errString), false
	}
	//2. 获取Bucket
	bucket, err := client.Bucket(bucketName)
	if err != nil {
		errString := fmt.Sprintf("%s client.Bucket failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return errors.New(errString), false
	}
	// 3. 检查文件是否存在
	ossFilePath := fmt.Sprintf("%s/%s", ossPath, logName)
	isExist, err := bucket.IsObjectExist(ossFilePath)
	if err != nil {
		errString := fmt.Sprintf("%s bucket.IsObjectExist failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return errors.New(errString), false
	}
	if isExist {
		return nil, true
	}
	return nil, false
}
