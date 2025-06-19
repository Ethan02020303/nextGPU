package process

import (
	"encoding/json"
	log4plus "github.com/nextGPU/include/log4go"
)

type TaskEvent int

const (
	SessionSync TaskEvent = iota + 1
	InGlobalQueue
	GlobalQueueChange
	InLocalQueue
	GenerateStart
	ImageGenerateSuccess
	ImageGenerateFailed
)

type (
	SubTask struct {
		SubID          string    `json:"subID"`
		State          TaskState `json:"status"`
		InitWaitCount  int64     `json:"initWaitCount"`
		SpareWaitCount int64     `json:"spareWaitCount"`
		PublishTime    string    `json:"publishTime"`
		StartTime      string    `json:"startTime"`
		EstimateMs     int64     `json:"estimateMs"`
		EndTime        string    `json:"endTime"`
		OSSUrls        []string  `json:"ossUrls"`
	}
)

func showEvent(event TaskEvent) string {
	if event == InGlobalQueue {
		return "InGlobalQueue"
	} else if event == GlobalQueueChange {
		return "GlobalQueueChange"
	} else if event == InLocalQueue {
		return "InLocalQueue"
	} else if event == GenerateStart {
		return "GenerateStart"
	} else if event == ImageGenerateSuccess {
		return "ImageGenerateSuccess"
	} else if event == ImageGenerateFailed {
		return "ImageGenerateFailed"
	} else if event == SessionSync {
		return "SessionSync"
	} else {
		return "unDefine"
	}
}

func sendSyncs(subID string, event TaskEvent) {
	funName := "sendSyncs"
	log4plus.Info("%s subID=[%s] event=[%s]", funName, subID, showEvent(event))
	tos := SingletonUsers().GetSessions()
	for _, to := range tos {
		tasks := SingletonUsers().GetSyncTasks(to.Session)
		if len(tasks) > 0 {
			for _, task := range tasks {
				sync := struct {
					MessageID int64     `json:"messageID"`
					TaskID    string    `json:"taskID"`
					Event     string    `json:"event"`
					State     TaskState `json:"status"`
					Tasks     []SubTask `json:"tasks"`
				}{
					MessageID: UserSyncTask,
					TaskID:    task.TaskID,
					State:     task.State,
					Event:     showEvent(event),
				}
				for _, subTask := range task.ShowTasks {
					showSubTask := SubTask{
						SubID:          subTask.SubID,
						State:          subTask.State,
						InitWaitCount:  subTask.InitWaitCount,
						SpareWaitCount: subTask.CurWaitCount,
						PublishTime:    time2String(subTask.PublishTime),
						StartTime:      time2String(subTask.StartTime),
						EstimateMs:     subTask.EstimateMs,
						EndTime:        time2String(subTask.CompletedTime),
					}
					for _, ossUrl := range subTask.OSSUrls {
						showSubTask.OSSUrls = append(showSubTask.OSSUrls, ossUrl)
					}
					sync.Tasks = append(sync.Tasks, showSubTask)
				}
				data, _ := json.Marshal(sync)
				session := SingletonUsers().FindSession(to.Session)
				if session != nil {
					session.Match.SendMessageCh <- data
				}
			}
		}
	}
}

func sendSyncs2(to string, event TaskEvent) {
	funName := "sendSyncs2"
	log4plus.Info("%s to=[%s] event=[%s]", funName, to, showEvent(event))
	tasks := SingletonUsers().GetSyncTasks(to)
	if len(tasks) > 0 {
		for _, task := range tasks {
			sync := struct {
				MessageID int64     `json:"messageID"`
				TaskID    string    `json:"taskID"`
				Event     string    `json:"event"`
				Tasks     []SubTask `json:"tasks"`
			}{
				MessageID: UserSyncTask,
				TaskID:    task.TaskID,
				Event:     showEvent(event),
			}
			for _, subTask := range task.ShowTasks {
				showSubTask := SubTask{
					SubID:          subTask.SubID,
					State:          subTask.State,
					InitWaitCount:  subTask.InitWaitCount,
					SpareWaitCount: subTask.CurWaitCount,
					PublishTime:    time2String(subTask.PublishTime),
					StartTime:      time2String(subTask.StartTime),
					EstimateMs:     subTask.EstimateMs,
					EndTime:        time2String(subTask.CompletedTime),
				}
				for _, ossUrl := range subTask.OSSUrls {
					showSubTask.OSSUrls = append(showSubTask.OSSUrls, ossUrl)
				}
				sync.Tasks = append(sync.Tasks, showSubTask)
			}
			data, _ := json.Marshal(sync)
			session := SingletonUsers().FindSession(to)
			if session != nil {
				session.Match.SendMessageCh <- data
			}
		}
	}
}
