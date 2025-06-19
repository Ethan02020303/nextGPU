package process

import (
	"encoding/json"
	"fmt"
	log4plus "github.com/nextGPU/include/log4go"
	"github.com/nextGPU/ng-cloud/configure"
	"github.com/nextGPU/ng-cloud/db"
	"net/http"
	"strings"
)

type (
	ResultMessage struct {
		MessageID int64  `json:"messageID"`
		CodeID    int64  `json:"codeID"`
		Msg       string `json:"msg"`
	}
	SubPublishResponse struct {
		SubID          string `json:"subID"`
		InitWaitCount  int64  `json:"initWaitCount"`
		SpareWaitCount int64  `json:"spareWaitCount"`
	}
	TaskPublishResponse struct {
		CodeID       int64                `json:"codeID"`
		Msg          string               `json:"msg"`
		Session      string               `json:"session,omitempty"`
		Connect      string               `json:"connect,omitempty"`
		TaskID       string               `json:"taskID,omitempty"`
		InitQueuePos int64                `json:"initQueuePos,omitempty"`
		ImageNum     int64                `json:"imageNum,omitempty"`
		Subs         []SubPublishResponse `json:"subs,omitempty"`
	}
	TaskDeleteResponse struct {
		CodeID int64  `json:"codeId"`
		Msg    string `json:"msg"`
	}
)

func userTaskPublish(session string, message []byte) (error, TaskPublishResponse) {
	funName := "taskPublish"
	var request TaskRequestData
	if err := json.Unmarshal(message, &request); err != nil {
		errString := fmt.Sprintf("%s json.Unmarshal Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		result := TaskPublishResponse{
			CodeID: http.StatusOK,
			Msg:    errString,
		}
		return nil, result
	}
	if user := SingletonUsers().FindSession(session); user == nil {
		errString := fmt.Sprintf("%s resources.SingletonUsers().FindSession Failed", funName)
		log4plus.Error(errString)
		result := TaskPublishResponse{
			CodeID: http.StatusOK,
			Msg:    errString,
		}
		return nil, result
	}
	task := SingletonUsers().NewTask(session, request, message)
	if task != nil {
		newListen := strings.Replace(configure.SingletonConfigure().Net.User.Listen, "0.0.0.0", configure.SingletonConfigure().Net.User.Domain, 1)
		result := TaskPublishResponse{
			CodeID:   http.StatusOK,
			Msg:      "success",
			Session:  session,
			Connect:  fmt.Sprintf("ws://%s%s/%s", newListen, configure.SingletonConfigure().Net.User.RoutePath, session),
			TaskID:   task.TaskID,
			ImageNum: task.ImageNum,
		}
		for _, subTask := range task.Tasks {
			result.Subs = append(result.Subs, SubPublishResponse{
				SubID:          subTask.SubID,
				InitWaitCount:  subTask.InitWaitCount,
				SpareWaitCount: subTask.CurWaitCount,
			})
		}
		return nil, result
	}
	result := TaskPublishResponse{
		CodeID: http.StatusOK,
		Msg:    "taskPublish Failed",
	}
	return nil, result
}

func userTaskStop(session string, message []byte) (error, ResultMessage) {
	funName := "taskStop"
	msg := struct {
		TaskID string `json:"taskID"`
	}{}
	if err := json.Unmarshal(message, &msg); err != nil {
		errString := fmt.Sprintf("%s json.Unmarshal Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		result := ResultMessage{
			CodeID: http.StatusOK,
			Msg:    errString,
		}
		return nil, result
	}
	if sessionTask := SingletonUsers().GetTask(session, msg.TaskID); sessionTask == nil {
		errString := fmt.Sprintf("%s GetTask Failed not found taskID=[%s]", funName, msg.TaskID)
		log4plus.Error(errString)
		result := ResultMessage{
			CodeID: http.StatusOK,
			Msg:    errString,
		}
		return nil, result
	} else {
		for _, task := range sessionTask.Tasks {
			SingletonNodes().DeleteTaskFromMatch(task.SystemUUID, task.SubID)
		}
		result := ResultMessage{
			CodeID: http.StatusOK,
			Msg:    "success",
		}
		return nil, result
	}
}

func userTaskDelete(session string, message []byte) (error, ResultMessage) {
	funName := "taskDelete"
	msg := struct {
		TaskID string `json:"taskID"`
	}{}
	if err := json.Unmarshal(message, &msg); err != nil {
		errString := fmt.Sprintf("%s json.Unmarshal Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		result := ResultMessage{
			CodeID: http.StatusOK,
			Msg:    errString,
		}
		return nil, result
	}
	if sessionTask := SingletonUsers().GetTask(session, msg.TaskID); sessionTask == nil {
		errString := fmt.Sprintf("%s GetTask Failed not found taskID=[%s]", funName, msg.TaskID)
		log4plus.Error(errString)
		result := ResultMessage{
			CodeID: http.StatusOK,
			Msg:    errString,
		}
		return nil, result
	} else {
		SingletonUsers().DeleteTask(msg.TaskID)
		for _, task := range sessionTask.Tasks {
			SingletonNodes().DeleteTaskFromMatch(task.SystemUUID, task.SubID)
		}
		_ = db.SingletonTaskDB().SetSubTaskDelete(msg.TaskID)
		result := ResultMessage{
			CodeID: http.StatusOK,
			Msg:    "success",
		}
		return nil, result
	}
}
