package process

import (
	"encoding/json"
	"fmt"
	"github.com/jinzhu/copier"
	log4plus "github.com/nextGPU/include/log4go"
	"github.com/nextGPU/ng-cloud/common"
	"github.com/nextGPU/ng-cloud/configure"
	"github.com/nextGPU/ng-cloud/db"
	"time"
)

func sendMeasure(systemUUID, workspaceID string, docker *DockerArray) {
	funName := "sendMeasure"
	node := SingletonNodes().FindNode(systemUUID)
	if node == nil {
		errString := fmt.Sprintf("%s not found node systemUUID=[%s]", funName, systemUUID)
		log4plus.Error(errString)
		return
	}
	batchSize := configure.SingletonConfigure().Measure.MeasureCount
	if batchSize <= 0 {
		batchSize = 1
	}
	docker.MeasureTaskID = fmt.Sprintf("%s.%s", MeasureID, common.TaskID())
	msg := struct {
		Title      string `json:"title"`
		ImageCount int64  `json:"imageCount"`
	}{
		Title:      docker.MeasureTitle,
		ImageCount: batchSize,
	}
	data, _ := json.Marshal(msg)
	err := db.SingletonTaskDB().InsertMainTask(docker.MeasureTaskID, MeasureID, docker.MeasureTitle, batchSize, db.AdminTask,
		docker.Measure, workspaceID, json.RawMessage(docker.Measure), string(data))
	if err != nil {
		errString := fmt.Sprintf("%s InsertMainTask Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return
	}
	SingletonNodes().CleanIdleSlot(systemUUID)
	for i := 1; i <= int(batchSize); i++ {
		subID := fmt.Sprintf("%s_%04d", docker.MeasureTaskID, i)
		subTask := &MeasureTaskSlot{
			WorkSpaceID: workspaceID,
			Title:       docker.MeasureTitle,
			TaskID:      docker.MeasureTaskID,
			SubID:       subID,
			Index:       int64(i),
			SystemUUID:  systemUUID,
			PublishTime: time.Now(),
			State:       PublishingTask,
			Duration:    0,
		}
		subTask.TaskData.Title = docker.MeasureTitle
		subTask.TaskData.SystemUUID = systemUUID
		newData, errData := SingletonWorkflows().SetMeasureSeed(json.RawMessage(docker.Measure), json.RawMessage(docker.MeasureConfiguration))
		if errData != nil {
			errString := fmt.Sprintf("%s SetSeed Failed errData=[%s]", funName, errData.Error())
			log4plus.Error(errString)
			return
		}
		_ = copier.CopyWithOption(&subTask.TaskData.Data, &newData, copier.Option{DeepCopy: true})
		docker.MeasureSubTasks = append(docker.MeasureSubTasks, subTask)
		msg := struct {
			MessageID   int64           `json:"messageID"`
			TaskID      string          `json:"taskID"`
			SubID       string          `json:"subID"`
			WorkSpaceID string          `json:"workSpaceID"`
			Data        json.RawMessage `json:"data"`
		}{
			MessageID:   MeasurePublish,
			TaskID:      subTask.TaskID,
			SubID:       subID,
			WorkSpaceID: workspaceID,
		}
		_ = copier.CopyWithOption(&msg.Data, &newData, copier.Option{DeepCopy: true})
		_ = db.SingletonTaskDB().InsertSubTask(subTask.TaskID, subTask.SubID, string(newData), workspaceID)
		message, errRes := json.Marshal(msg)
		if errRes != nil {
			errString := fmt.Sprintf("%s json.Marshal Failed err=[%s]", funName, errRes.Error())
			log4plus.Error(errString)
			return
		}
		slot := &Slot{
			SystemUUID: systemUUID,
			TaskID:     docker.MeasureTaskID,
			SlotID:     1,
			SubID:      subID,
			State:      StatusIdle,
		}
		if node.Match != nil {
			node.Match.SendMessageCh <- message
			SingletonNodes().PushMeasureSlot(slot)
			node.MeasureTime = time.Now()
		}
	}
	node.State = db.Measuring
}

func sendDispenseStart(workSpace WorkSpace, node *Node) {
	funName := "sendDispenseStart"
	taskID := common.TaskID()
	log4plus.Info("----------------------->>>>>>> %s", taskID)
	dispense := struct {
		MessageID   int64            `json:"messageID"`
		SubID       string           `json:"subID"`
		WorkSpaceID string           `json:"workSpaceID"`
		RunningMode db.RunMode       `json:"runMode"`
		State       db.DockerStatus  `json:"state"`
		MajorCMD    string           `json:"majorCMD"`
		MinorCMD    string           `json:"minorCMD"`
		ComfyUIHttp string           `json:"comfyUIHttp"`
		ComfyUIWS   string           `json:"comfyUIWS"`
		SaveDir     string           `json:"saveDir"`
		Deleted     db.DockerDeleted `json:"deleted"`
	}{
		MessageID:   DispenseStart,
		SubID:       DispenseID,
		WorkSpaceID: workSpace.Base.WorkspaceID,
		RunningMode: workSpace.Base.RunningMode,
		MajorCMD:    workSpace.Base.MajorCMD,
		MinorCMD:    workSpace.Base.MinorCMD,
		ComfyUIHttp: workSpace.Base.ComfyUIHttp,
		ComfyUIWS:   workSpace.Base.ComfyUIWS,
		SaveDir:     workSpace.Base.SavePath,
		Deleted:     workSpace.Base.Deleted,
	}
	message, err := json.Marshal(dispense)
	if err != nil {
		errString := fmt.Sprintf("%s json.Marshal Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return
	}
	slot := &Slot{
		SystemUUID: node.SystemUUID,
		TaskID:     taskID,
		SlotID:     1,
		SubID:      dispense.SubID,
		State:      StatusIdle,
	}
	node.Base.Dockers = append(node.Base.Dockers, &DockerArray{
		WorkSpaceID:          workSpace.Base.WorkspaceID,
		RunningMode:          workSpace.Base.RunningMode,
		State:                db.Pulling,
		MajorCMD:             workSpace.Base.MajorCMD,
		MinorCMD:             workSpace.Base.MinorCMD,
		ComfyUIHttp:          workSpace.Base.ComfyUIHttp,
		ComfyUIWS:            workSpace.Base.ComfyUIWS,
		SaveDir:              workSpace.Base.SavePath,
		Deleted:              workSpace.Base.Deleted,
		Measure:              workSpace.Base.Measure,
		MeasureConfiguration: workSpace.Base.MeasureConfiguration,
	})
	if node.Match != nil {
		node.Match.SendMessageCh <- message
		SingletonNodes().PushDispenseSlot(slot)
	}
}

func sendOSSConfigure(systemUUID string) {
	if node := SingletonNodes().FindNode(systemUUID); node != nil {
		type OSSConfig struct {
			Endpoint        string `json:"endpoint"`
			AccessKeyID     string `json:"accessKeyID"`
			AccessKeySecret string `json:"accessKeySecret"`
			BucketName      string `json:"bucketName"`
		}
		oss := struct {
			MessageID int64     `json:"messageID"`
			SubID     string    `json:"subID"`
			AliOSS    OSSConfig `json:"aliOSS"`
			AliLogOSS OSSConfig `json:"aliLogOSS"`
		}{
			MessageID: OSSConfigure,
			SubID:     ConfigureID,
		}
		//Ali OSS
		oss.AliOSS.Endpoint = configure.SingletonConfigure().AliOSS.Endpoint
		oss.AliOSS.AccessKeyID = configure.SingletonConfigure().AliOSS.AccessKeyID
		oss.AliOSS.AccessKeySecret = configure.SingletonConfigure().AliOSS.AccessKeySecret
		oss.AliOSS.BucketName = configure.SingletonConfigure().AliOSS.BucketName
		//AliLogOSS
		oss.AliLogOSS.Endpoint = configure.SingletonConfigure().AliLogOSS.Endpoint
		oss.AliLogOSS.AccessKeyID = configure.SingletonConfigure().AliLogOSS.AccessKeyID
		oss.AliLogOSS.AccessKeySecret = configure.SingletonConfigure().AliLogOSS.AccessKeySecret
		oss.AliLogOSS.BucketName = configure.SingletonConfigure().AliLogOSS.BucketName
		message, _ := json.Marshal(oss)
		if node.Match != nil {
			node.Match.SendMessageCh <- message
		}
	}
}

func sendMonitorConfigure(systemUUID string) {
	funName := "sendMonitorConfigure"
	if node := SingletonNodes().FindNode(systemUUID); node != nil {
		config := struct {
			MessageID int64                     `json:"messageID"`
			Monitors  []db.MonitorConfiguration `json:"monitors"`
		}{
			MessageID: MonitorConfigure,
		}
		if err, monitors := db.SingletonNodeBaseDB().Configuration(); err != nil {
			errString := fmt.Sprintf("%s json.Marshal Failed err=[%s]", funName, err.Error())
			log4plus.Error(errString)
			return
		} else {
			for _, monitor := range monitors.Monitors {
				config.Monitors = append(config.Monitors, db.MonitorConfiguration{
					MonitorType: monitor.MonitorType,
					Monitor:     monitor.Monitor,
				})
			}
		}
		message, _ := json.Marshal(config)
		if node.Match != nil {
			node.Match.SendMessageCh <- message
		}
	}
}

func sendPong(systemUUID string) {
	if node := SingletonNodes().FindNode(systemUUID); node != nil {
		pong := struct {
			MessageID int64         `json:"messageID"`
			Dockers   []DockerArray `json:"dockers"`
		}{
			MessageID: NodePing,
		}
		for _, docker := range node.Base.Dockers {
			pong.Dockers = append(pong.Dockers, DockerArray{
				WorkSpaceID: docker.WorkSpaceID,
				State:       docker.State,
				MajorCMD:    docker.MajorCMD,
				MinorCMD:    docker.MinorCMD,
				ComfyUIHttp: docker.ComfyUIHttp,
				ComfyUIWS:   docker.ComfyUIWS,
				Deleted:     docker.Deleted,
			})
		}
		message, _ := json.Marshal(pong)
		if node.Match != nil {
			node.Match.SendMessageCh <- message
		}
	}
}
