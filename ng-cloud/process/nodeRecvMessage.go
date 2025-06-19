package process

import (
	"encoding/json"
	"fmt"
	log4plus "github.com/nextGPU/include/log4go"
	"github.com/nextGPU/include/monitor"
	"github.com/nextGPU/ng-cloud/configure"
	"github.com/nextGPU/ng-cloud/db"
	"sort"
	"strings"
	"time"
)

type QueueTask struct {
	SubID       string `json:"subID"`
	InLocalPos  int64  `json:"inLocalPos"`
	InLocalTime int64  `json:"inLocalTime"`
	CurQueuePos int64  `json:"curQueuePos"`
	StartTime   int64  `json:"startTime"`
	EndTime     int64  `json:"endTime"`
}

// 队列排序并同步给用户
func syncUserQueue(subID string, event TaskEvent) {
	SingletonUsers().WaitSort()
	SingletonUsers().ProcessSort()
	sendSyncs(subID, event)
}

// 设置算力节点测量未通过
func setNodeUnPass(systemUUID, workspaceID, subID string) {
	SingletonNodes().SetDockerStatus(systemUUID, workspaceID, db.UnPassed)
	_ = db.SingletonNodeBaseDB().SetDockerStatus(systemUUID, workspaceID, db.UnPassed)
	SingletonNodes().ClearIdle(systemUUID, subID)
}

// 设置算力节点测量未通过
func setMeasureNodeUnPass(systemUUID, workspaceID, subID string) {
	SingletonNodes().SetMeasureUnPassed(systemUUID)
	SingletonNodes().SetDockerStatus(systemUUID, workspaceID, db.UnPassed)
	SingletonNodes().ClearIdle(systemUUID, subID)
	_ = db.SingletonNodeBaseDB().SetDockerStatus(systemUUID, workspaceID, db.UnPassed)
}

// 设置算力节点测量通过
func setMeasureNodePass(systemUUID, workspaceID, subID string) {
	SingletonNodes().SetDockerStatus(systemUUID, workspaceID, db.Passed)
	SingletonNodes().ClearIdle(systemUUID, subID)
	SingletonNodes().SetMeasureCompletion(systemUUID)
	_ = db.SingletonNodeBaseDB().SetDockerStatus(systemUUID, workspaceID, db.Passed)
}

// 发布测量任务
func measurePublish(systemUUID string, message []byte) {
	funName := "measure"
	publish := struct {
		MessageID   int64       `json:"messageID"`
		WorkSpaceID string      `json:"workSpaceID"`
		SubID       string      `json:"subID"`
		Session     string      `json:"session"`
		Success     bool        `json:"success"`
		Reason      string      `json:"reason"`
		Tasks       []QueueTask `json:"tasks"`
	}{}
	if err := json.Unmarshal(message, &publish); err != nil {
		errString := fmt.Sprintf("%s json.Unmarshal Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return
	}
	node := SingletonNodes().FindNode(systemUUID)
	if node == nil {
		errString := fmt.Sprintf("%s FindNode Failed not found systemUUID=[%s]", funName, systemUUID)
		log4plus.Error(errString)
		return
	}
	for _, docker := range node.Base.Dockers {
		for _, dockerMeasure := range docker.MeasureSubTasks {
			if strings.EqualFold(dockerMeasure.SubID, publish.SubID) {
				taskID := dockerMeasure.TaskID
				if publish.Success {
					//设置测量状态
					SingletonNodes().SetNodeStatus(systemUUID, db.Measuring)
					//设置测量发布时间
					_ = db.SingletonTaskDB().SetPublishTime(taskID, time.Now().Format("2006-01-02 15:04:05"))
					_ = db.SingletonTaskDB().SetSubTaskPublish(taskID, publish.SubID, systemUUID, time.Now().Format("2006-01-02 15:04:05"))
				} else {
					SingletonNodes().SetNodeStatus(systemUUID, db.MeasurementFailed)
				}
			}
		}
	}
}

func time2String(curTime int64) string {
	sec := curTime / 1000
	nsec := (curTime % 1000) * int64(time.Millisecond)
	tmpTime := time.Unix(sec, nsec)
	curTimeString := tmpTime.Format("2006-01-02 15:04:05.000")
	return curTimeString
}

// 测量任务开始执行
func measureStart(systemUUID string, message []byte) {
	funName := "measureStart"
	start := struct {
		MessageID   int64       `json:"messageID"`
		WorkSpaceID string      `json:"workSpaceID"`
		SubID       string      `json:"subID"`
		Tasks       []QueueTask `json:"tasks"`
	}{}
	if err := json.Unmarshal(message, &start); err != nil {
		errString := fmt.Sprintf("%s json.Unmarshal Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return
	}
	node := SingletonNodes().FindNode(systemUUID)
	if node == nil {
		errString := fmt.Sprintf("%s FindNode Failed not found systemUUID=[%s]", funName, systemUUID)
		log4plus.Error(errString)
		return
	}
	//获取测量任务信息
	var measureTask QueueTask
	for _, v := range start.Tasks {
		if strings.EqualFold(v.SubID, start.SubID) {
			measureTask = v
		}
	}
	for _, docker := range node.Base.Dockers {
		for _, dockerMeasure := range docker.MeasureSubTasks {
			if strings.EqualFold(dockerMeasure.SubID, start.SubID) {
				taskID := dockerMeasure.TaskID
				startTimer := time2String(measureTask.StartTime)
				_ = db.SingletonTaskDB().SetStartTime(taskID, startTimer)
				//预估参数信息
				err, completionTime, familyWeight := SingletonConsumptions().CalcEstimates(dockerMeasure.Title,
					systemUUID,
					configure.SingletonConfigure().Measure.MeasureWidth,
					configure.SingletonConfigure().Measure.MeasureHeight)
				if err != nil {
					errString := fmt.Sprintf("%s CalcEstimates Failed not found systemUUID=[%s]", funName, systemUUID)
					log4plus.Error(errString)
					return
				}
				//计算预估时间
				estimate := int64(float64(completionTime) * familyWeight)
				estimateMS := estimate + int64(configure.SingletonConfigure().Measure.MaxUnPassPercent*float64(estimate))
				dockerMeasure.EstimatesTime = estimateMS
				_ = db.SingletonTaskDB().SetSubTaskStart(taskID, start.SubID, systemUUID, estimateMS)
				log4plus.Info("%s subID=[%s] estimateMS=[%d]", funName, start.SubID, estimateMS)
				return
			}
		}
	}
}

// 测量任务失败
func measureFailed(systemUUID string, message []byte) {
	funName := "measureFailed"
	failed := struct {
		MessageID     int64  `json:"messageID"`
		WorkSpaceID   string `json:"workSpaceID"`
		SubID         string `json:"subID"`
		FailureReason string `json:"failureReason"`
		ResultData    string `json:"resultData"`
		Duration      int64  `json:"duration"`
		StartTime     int64  `json:"startTime"`
		EndTime       int64  `json:"endTime"`
	}{}
	if err := json.Unmarshal(message, &failed); err != nil {
		errString := fmt.Sprintf("%s json.Unmarshal Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return
	}
	SingletonNodes().ClearTaskFromMatch(systemUUID)
	SingletonNodes().ClearIdle(systemUUID, failed.SubID)
	node := SingletonNodes().FindNode(systemUUID)
	if node == nil {
		errString := fmt.Sprintf("%s FindNode Failed not found systemUUID=[%s]", funName, systemUUID)
		log4plus.Error(errString)
		return
	}
	var taskID string
	for _, docker := range node.Base.Dockers {
		for _, dockerMeasure := range docker.MeasureSubTasks {
			if strings.EqualFold(dockerMeasure.SubID, failed.SubID) {
				taskID = dockerMeasure.TaskID
				duration := failed.EndTime - failed.StartTime
				db.SingletonTaskDB().SetMeasureTaskFailed(taskID, failed.SubID, time2String(failed.StartTime), time2String(failed.EndTime),
					failed.ResultData, duration, failed.FailureReason)
				log4plus.Info("%s SetSubTaskFailed taskID=[%s] subID=[%s]", funName, taskID, failed.SubID)
				break
			}
		}
	}
	db.SingletonTaskDB().SetTaskFailed(taskID, time2String(failed.EndTime))
	SingletonNodes().Workspace2Node(systemUUID)
	setMeasureNodeUnPass(systemUUID, failed.WorkSpaceID, failed.SubID)
}

// 测量任务完成
func measureComplete(systemUUID string, message []byte) {
	funName := "measureComplete"
	completed := struct {
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
	}{}
	if err := json.Unmarshal(message, &completed); err != nil {
		errString := fmt.Sprintf("%s json.Unmarshal Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return
	}
	SingletonNodes().ClearTaskFromMatch(systemUUID)
	node := SingletonNodes().FindNode(systemUUID)
	if node == nil {
		errString := fmt.Sprintf("%s FindNode Failed not found systemUUID=[%s]", funName, systemUUID)
		log4plus.Error(errString)
		return
	}
	if completed.StartTime > completed.EndTime {
		errString := fmt.Sprintf("%s >>>>>>> startTime=[%d] endTime=[%d]", funName, completed.StartTime, completed.EndTime)
		log4plus.Error(errString)
	}
	var taskID string
	for _, docker := range node.Base.Dockers {
		for _, dockerMeasure := range docker.MeasureSubTasks {
			if strings.EqualFold(dockerMeasure.SubID, completed.SubID) {
				taskID = dockerMeasure.TaskID
				//set task result
				startTimer := time2String(completed.StartTime)
				endTimer := time2String(completed.EndTime)

				var measureOSSUrls []string
				for _, ossUrl := range completed.ImageUrls {
					ossAllUrl := fmt.Sprintf("https://%s.%s/%s", configure.SingletonConfigure().AliOSS.BucketName, configure.SingletonConfigure().AliOSS.Endpoint, ossUrl)
					measureOSSUrls = append(measureOSSUrls, ossAllUrl)
				}
				db.SingletonTaskDB().SubTaskResult(taskID, completed.SubID, startTimer, endTimer, completed.ResultData,
					completed.Duration, measureOSSUrls, completed.OSSDurations, completed.ImageSizes)
				break
			}
		}
	}
	for _, docker := range node.Base.Dockers {
		if strings.EqualFold(docker.WorkSpaceID, completed.WorkSpaceID) {
			//set duration
			for _, measureSubTask := range docker.MeasureSubTasks {
				if measureSubTask.SubID == completed.SubID {
					if len(completed.ImageUrls) == len(completed.ImageSizes) && len(completed.ImageSizes) > 0 {
						measureSubTask.State = CompletedTask
						measureSubTask.Duration = completed.Duration
						break
					} else {
						measureSubTask.State = FailedTask
						measureSubTask.Duration = completed.Duration
						break
					}
				}
			}
			//Check whether the measurement task is completed
			isCompleted := true
			for _, measureSubTask := range docker.MeasureSubTasks {
				if measureSubTask.State != CompletedTask && measureSubTask.State != FailedTask {
					isCompleted = false
				}
			}
			if !isCompleted {
				SingletonNodes().PushIdle(systemUUID, completed.SubID)
				return
			}
			//Check if it exceeds the estimated time
			isUnPass := false
			failedIndex := 0
			sort.Slice(docker.MeasureSubTasks, func(p, q int) bool { return docker.MeasureSubTasks[p].Index < docker.MeasureSubTasks[q].Index })
			for index, measureSubTask := range docker.MeasureSubTasks {
				if measureSubTask.Index != 1 {
					if measureSubTask.EstimatesTime < measureSubTask.Duration {
						isUnPass = true
						failedIndex = index
					}
					if measureSubTask.State == FailedTask {
						isUnPass = true
						failedIndex = index
					}
				}
			}
			//Set the computing power node measurement pass status
			if isUnPass {
				log4plus.Info("%s Measure UnPass -->> systemUUID=[%s] index=[%d] estimatesMS=[%d] duration=[%d]",
					funName, systemUUID, failedIndex, docker.MeasureSubTasks[failedIndex].EstimatesTime, docker.MeasureSubTasks[failedIndex].Duration)
				setMeasureNodeUnPass(systemUUID, completed.WorkSpaceID, completed.SubID)
			} else {
				log4plus.Info("%s Measure Pass -->> systemUUID=[%s] ", funName, systemUUID)
				setMeasureNodePass(systemUUID, completed.WorkSpaceID, completed.SubID)
			}
			_ = db.SingletonTaskDB().SetTaskResult(taskID, time2String(completed.EndTime))
		}
	}
}

// 任务发布
func taskPublish(message []byte) {
	funName := "taskPublish"
	publish := struct {
		MessageID   int64       `json:"messageID"`
		WorkSpaceID string      `json:"workSpaceID"`
		SubID       string      `json:"subID"`
		Session     string      `json:"session"`
		Success     bool        `json:"success"`
		Reason      string      `json:"reason"`
		Tasks       []QueueTask `json:"tasks"`
	}{}
	if err := json.Unmarshal(message, &publish); err != nil {
		errString := fmt.Sprintf("%s json.Unmarshal Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return
	}
	if strings.EqualFold(publish.SubID, MeasureID) || strings.EqualFold(publish.SubID, DispenseID) {
		errString := fmt.Sprintf("%s EqualFold Failed start.SubId=[%s]", funName, publish.SubID)
		log4plus.Error(errString)
		return
	}
	subTask := SingletonUsers().FindProcessing(publish.SubID)
	if subTask == nil {
		errString := fmt.Sprintf("%s FindProcessing Failed subID not found subID=[%s]", funName, publish.SubID)
		log4plus.Error(errString)
		return
	}
	subTask.State = PublishedTask
	for _, v := range publish.Tasks {
		if strings.EqualFold(v.SubID, subTask.SubID) {
			subTask.InLocalTime = v.InLocalTime
		}
	}
	//write db
	_ = db.SingletonTaskDB().SetPublishTime(subTask.TaskID, time.Now().Format("2006-01-02 15:04:05"))
	_ = db.SingletonTaskDB().SetSubTaskPublish(subTask.TaskID, subTask.SubID, subTask.SystemUUID, time.Now().Format("2006-01-02 15:04:05"))
	//排序并同步
	syncUserQueue(publish.SubID, InLocalQueue)
}

// 任务开始执行
func taskStart(message []byte) {
	funName := "taskStart"
	start := struct {
		MessageID   int64       `json:"messageID"`
		WorkSpaceID string      `json:"workSpaceID"`
		SubID       string      `json:"subID"`
		Tasks       []QueueTask `json:"tasks"`
	}{}
	if err := json.Unmarshal(message, &start); err != nil {
		errString := fmt.Sprintf("%s json.Unmarshal Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return
	}
	if strings.EqualFold(start.SubID, MeasureID) || strings.EqualFold(start.SubID, DispenseID) {
		errString := fmt.Sprintf("%s EqualFold Failed start.SubId=[%s]", funName, start.SubID)
		log4plus.Error(errString)
		return
	}
	subTask := SingletonUsers().FindProcessing(start.SubID)
	if subTask == nil {
		errString := fmt.Sprintf("%s FindProcessing Failed subID not found ", funName)
		log4plus.Error(errString)
		return
	}
	subTask.State = RunningTask
	if subTask.EstimateTime != nil {
		subTask.EstimateTime.Stop()
	}
	for _, v := range start.Tasks {
		if strings.EqualFold(v.SubID, subTask.SubID) {
			subTask.InLocalTime = v.InLocalTime
			subTask.StartTime = v.StartTime
		}
	}
	/*预估时间*/
	err, completionTime, familyWeight := SingletonConsumptions().CalcEstimates(subTask.WorkflowTitle, subTask.SystemUUID, subTask.Width, subTask.Height)
	if err != nil {
		errString := fmt.Sprintf("%s CalcEstimates Failed not found title=[%s] width=[%d] height=[%d]", funName, subTask.WorkflowTitle, subTask.Width, subTask.Height)
		log4plus.Error(errString)
		subTask.EstimateMs = 0
	} else {
		estimate := int64(float64(completionTime) * float64(subTask.InputImageScaledRatio) * familyWeight)
		subTask.EstimateMs = int64(estimate) + int64(configure.SingletonConfigure().Measure.MaxUnPassPercent*float64(estimate))
		subTask.EstimateTime = time.AfterFunc(time.Duration(subTask.EstimateMs)*time.Millisecond, func() {
			taskEstimateTimeout(subTask.SubID)
		})
	}
	//写DB
	startTime := time2String(subTask.StartTime)
	_ = db.SingletonTaskDB().SetStartTime(subTask.TaskID, startTime)
	_ = db.SingletonTaskDB().SetSubTaskStart(subTask.TaskID, subTask.SubID, subTask.SystemUUID, subTask.EstimateMs)
	log4plus.Info("%s SetSubTaskStart taskID=[%s] subID=[%s] startTime=[%d] estimateMS=[%d]", funName, subTask.TaskID, subTask.SubID, subTask.StartTime, subTask.EstimateMs)
	//排序并同步
	syncUserQueue(start.SubID, GenerateStart)
}

// 任务重新调度
func taskReScheduling(subTask *TaskSlot) {
	funName := "taskReScheduling"
	//retry count
	err, retryCount, errorNodes := db.SingletonTaskDB().GetSubTaskFailed(subTask.TaskID, subTask.SubID)
	if err != nil {
		retryCount = configure.SingletonConfigure().AITask.MaxRetryCount
	}
	if retryCount < configure.SingletonConfigure().AITask.MaxRetryCount {
		//判断是否还能进行处理
		log4plus.Info("%s retry publish subTask subID=[%s]", funName, subTask.SubID)
		if SingletonUsers().ExistOtherNodes(subTask) {
			subTask.State = WaitingTask
			subTask.PushQueueTime = time.Now().UnixMilli()
			subTask.ErrorNodes = subTask.ErrorNodes[:0]
			parts := strings.Split(errorNodes, ",")
			subTask.ErrorNodes = append(subTask.ErrorNodes, parts...)
			SingletonUsers().PushWaitHead(subTask)
			return
		}
	}
	subTask.State = FailedTask
	subTask.PushQueueTime = time.Now().UnixMilli()
	subTask.ErrorNodes = subTask.ErrorNodes[:0]
	parts := strings.Split(errorNodes, ",")
	subTask.ErrorNodes = append(subTask.ErrorNodes, parts...)
	//写DB
	SingletonUsers().FailedSubTask(subTask.SubID, subTask)
}

// 任务预估超时
func taskEstimateTimeout(subID string) {
	funName := "taskEstimateTimeout"
	log4plus.Info("%s --------------- subID=[%s]", funName, subID)
	//超时重新进行调度
	subTask := SingletonUsers().FindProcessing(subID)
	if subTask == nil {
		errString := fmt.Sprintf("%s FindProcessing Failed subID not found ", funName)
		log4plus.Error(errString)
		return
	}
	if subTask.EstimateTime != nil {
		subTask.EstimateTime.Stop()
	}
	subTask.CompletedTime = time.Now().UnixMilli()
	db.SingletonTaskDB().SetSubTaskFailed(subTask.SystemUUID,
		subTask.TaskID,
		subTask.SubID,
		time2String(subTask.StartTime),
		time2String(subTask.CompletedTime),
		subTask.CompletedTime-subTask.StartTime,
		"estimateTimeout")
	//设置算力节点不通过
	setNodeUnPass(subTask.SystemUUID, subTask.WorkSpaceID, subTask.SubID)
	//重新调度
	taskReScheduling(subTask)
	//排序并同步
	syncUserQueue(subTask.SubID, ImageGenerateFailed)
}

// 任务推理完成
func taskInferenceCompleted(message []byte) {
	funName := "taskInferenceCompleted"
	inferenceCompleted := struct {
		MessageID   int64  `json:"messageID"`
		WorkSpaceID string `json:"workSpaceID"`
		SubID       string `json:"subID"`
	}{}
	if err := json.Unmarshal(message, &inferenceCompleted); err != nil {
		errString := fmt.Sprintf("%s json.Unmarshal Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return
	}
	if strings.EqualFold(inferenceCompleted.SubID, MeasureID) || strings.EqualFold(inferenceCompleted.SubID, DispenseID) {
		errString := fmt.Sprintf("%s EqualFold Failed inferenceCompleted.SubId=[%s]", funName, inferenceCompleted.SubID)
		log4plus.Error(errString)
		return
	}
	subTask := SingletonUsers().FindProcessing(inferenceCompleted.SubID)
	if subTask == nil {
		errString := fmt.Sprintf("%s FindProcessing Failed subID not found ", funName)
		log4plus.Error(errString)
		return
	}
	if subTask.EstimateTime != nil {
		subTask.EstimateTime.Stop()
	}
}

// 任务完成
func taskCompleted(systemUUID string, message []byte) {
	funName := "taskCompleted"
	completed := struct {
		MessageID    int64       `json:"messageID"`
		WorkSpaceID  string      `json:"workSpaceID"`
		SubID        string      `json:"subID"`
		ImageUrls    []string    `json:"imageUrls"`
		OSSDurations []int64     `json:"ossDurations"`
		ImageSizes   []int64     `json:"imageSizes"`
		ResultData   string      `json:"resultData"`
		Tasks        []QueueTask `json:"tasks"`
	}{}
	if err := json.Unmarshal(message, &completed); err != nil {
		errString := fmt.Sprintf("%s json.Unmarshal Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return
	}
	subTask := SingletonUsers().FindProcessing(completed.SubID)
	if subTask == nil {
		errString := fmt.Sprintf("%s FindProcessing Failed subID not found SubId=[%s]", funName, completed.SubID)
		log4plus.Error(errString)
		return
	}
	if !strings.EqualFold(subTask.SystemUUID, systemUUID) {
		errString := fmt.Sprintf("%s EqualFold Failed subID=[%s] sub Task systemUUID=[%s] result systemUUID=[%s]",
			funName, completed.SubID, subTask.SystemUUID, systemUUID)
		log4plus.Error(errString)
		return
	}
	if subTask.EstimateTime != nil {
		subTask.EstimateTime.Stop()
	}
	for _, v := range completed.Tasks {
		if strings.EqualFold(v.SubID, subTask.SubID) {
			subTask.InLocalTime = v.InLocalTime
			subTask.StartTime = v.StartTime
			subTask.CompletedTime = v.EndTime
		}
	}
	for _, ossUrl := range completed.ImageUrls {
		ossAllUrl := fmt.Sprintf("https://%s.%s/%s", configure.SingletonConfigure().AliOSS.BucketName, configure.SingletonConfigure().AliOSS.Endpoint, ossUrl)
		subTask.OSSUrls = append(subTask.OSSUrls, ossAllUrl)
	}
	log4plus.Info("%s ---->>>> subID=[%s] EstimateMS=[%d] DealMS=[%d] ImageUrls=[%+v] ImageSizes=[%+v] OSSDurations=[%+v]",
		funName, completed.SubID, subTask.EstimateMs, subTask.CompletedTime-subTask.StartTime, completed.ImageUrls, completed.ImageSizes, completed.OSSDurations)
	SingletonNodes().DeleteTaskFromMatch(systemUUID, completed.SubID)
	dockerState := SingletonNodes().GetDockerStatus(systemUUID, completed.WorkSpaceID)
	if dockerState == db.UnPassed {
		log4plus.Info(fmt.Sprintf("%s --->>>subID=[%s] width=[%d] height=[%d] endTime-startTime=[%d] estimateMs=[%d]",
			funName, completed.SubID, subTask.Width, subTask.Height, subTask.CompletedTime-subTask.StartTime, subTask.EstimateMs))
		//设置算力节点不通过
		setNodeUnPass(subTask.SystemUUID, subTask.WorkSpaceID, completed.SubID)
		//设置任务失败
		db.SingletonTaskDB().SetSubTaskFailed(systemUUID,
			subTask.TaskID,
			subTask.SubID,
			time2String(subTask.StartTime),
			time2String(subTask.CompletedTime),
			subTask.CompletedTime-subTask.StartTime,
			"The computing power node failed the measurement")
		//重新调度
		taskReScheduling(subTask)
		//排序并同步
		syncUserQueue(completed.SubID, ImageGenerateSuccess)
	} else if dockerState == db.Passed {
		//判断任务是否超时
		if subTask.CompletedTime-subTask.StartTime > subTask.EstimateMs {
			subTask.State = CompletedTask
			duration := subTask.CompletedTime - subTask.StartTime
			db.SingletonTaskDB().SubTaskResult(subTask.TaskID, subTask.SubID, time2String(subTask.StartTime), time2String(subTask.CompletedTime), completed.ResultData,
				duration, subTask.OSSUrls, completed.OSSDurations, completed.ImageSizes)
			//设置算力节点不通过
			setNodeUnPass(subTask.SystemUUID, subTask.WorkSpaceID, completed.SubID)
			//重新将任务放入队列头部等待处理(处理逻辑与失败相同) ???
			db.SingletonTaskDB().SetSubTaskFailed(systemUUID, subTask.TaskID, subTask.SubID, time2String(subTask.StartTime), time2String(subTask.CompletedTime),
				subTask.CompletedTime-subTask.StartTime, "The computing node image generation timeout")
			//重新调度
			taskReScheduling(subTask)
			//排序并同步
			syncUserQueue(completed.SubID, ImageGenerateSuccess)
		} else {
			//失败需要重新调度
			if len(completed.ImageUrls) != len(completed.ImageSizes) || len(completed.ImageSizes) == 0 {
				subTask.State = CompletedTask
				duration := subTask.CompletedTime - subTask.StartTime
				db.SingletonTaskDB().SubTaskResult(subTask.TaskID, subTask.SubID, time2String(subTask.StartTime), time2String(subTask.CompletedTime), completed.ResultData,
					duration, subTask.OSSUrls, completed.OSSDurations, completed.ImageSizes)
				//c
				setNodeUnPass(subTask.SystemUUID, subTask.WorkSpaceID, completed.SubID)
				//重新将任务放入队列头部等待处理(处理逻辑与失败相同) ???
				db.SingletonTaskDB().SetSubTaskFailed(systemUUID, subTask.TaskID, subTask.SubID, time2String(subTask.StartTime), time2String(subTask.CompletedTime),
					subTask.CompletedTime-subTask.StartTime, "The computing node image generation timeout")
				//重新调度
				taskReScheduling(subTask)
				//排序并同步
				syncUserQueue(completed.SubID, ImageGenerateSuccess)
			} else {
				subTask.State = CompletedTask
				duration := subTask.CompletedTime - subTask.StartTime
				db.SingletonTaskDB().SubTaskResult(subTask.TaskID, subTask.SubID, time2String(subTask.StartTime), time2String(subTask.CompletedTime), completed.ResultData,
					duration, subTask.OSSUrls, completed.OSSDurations, completed.ImageSizes)
				SingletonNodes().PushIdle(systemUUID, completed.SubID)
				offline := SingletonUsers().CompletedSubTask(completed.SubID, subTask.CompletedTime)
				//排序并同步
				syncUserQueue(completed.SubID, ImageGenerateSuccess)
				//断开用户
				if offline {
					SingletonUsers().DeleteSession(subTask.Session)
				}
			}
		}
	}
}

// 任务失败
func taskFailed(systemUUID string, message []byte) {
	funName := "taskFailed"
	failed := struct {
		MessageID     int64       `json:"messageID"`
		WorkSpaceID   string      `json:"workSpaceID"`
		SubID         string      `json:"subID"`
		ResultData    string      `json:"resultData"`
		FailureReason string      `json:"failureReason"`
		Tasks         []QueueTask `json:"tasks"`
	}{}
	if err := json.Unmarshal(message, &failed); err != nil {
		errString := fmt.Sprintf("%s json.Unmarshal Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return
	}
	subTask := SingletonUsers().FindProcessing(failed.SubID)
	if subTask == nil {
		errString := fmt.Sprintf("%s FindProcessing Failed subID not found SubId=[%s]", funName, failed.SubID)
		log4plus.Error(errString)
		return
	}
	if !strings.EqualFold(subTask.SystemUUID, systemUUID) {
		errString := fmt.Sprintf("%s EqualFold Failed subID=[%s] sub Task systemUUID=[%s] result systemUUID=[%s]", funName, failed.SubID, subTask.SystemUUID, systemUUID)
		log4plus.Error(errString)
		return
	}
	if subTask.EstimateTime != nil {
		subTask.EstimateTime.Stop()
	}
	for _, v := range failed.Tasks {
		if strings.EqualFold(v.SubID, subTask.SubID) {
			subTask.InLocalTime = v.InLocalTime
			subTask.StartTime = v.StartTime
			subTask.CompletedTime = v.EndTime
		}
	}
	SingletonNodes().DeleteTaskFromMatch(systemUUID, failed.SubID)
	duration := subTask.CompletedTime - subTask.StartTime
	db.SingletonTaskDB().SetSubTaskFailed(systemUUID,
		subTask.TaskID,
		subTask.SubID,
		time2String(subTask.StartTime),
		time2String(subTask.CompletedTime),
		duration,
		failed.FailureReason)
	//设置算力节点测量通过
	setNodeUnPass(subTask.SystemUUID, subTask.WorkSpaceID, failed.SubID)
	//重新调度
	taskReScheduling(subTask)
	//排序并同步
	syncUserQueue(failed.SubID, ImageGenerateFailed)
}

// 发布Docker开始
func dispenseStart(systemUUID string, message []byte) {
	funName := "dispenseStart"
	start := struct {
		MessageID   int64  `json:"messageID"`
		WorkSpaceID string `json:"workSpaceID"`
	}{}
	if err := json.Unmarshal(message, &start); err != nil {
		errString := fmt.Sprintf("%s json.Unmarshal Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return
	}
	log4plus.Info(fmt.Sprintf("%s ---->>>>[%#v]", funName, start))
	SingletonNodes().SetNodeDockerStatus(systemUUID, start.WorkSpaceID, db.Pulling)
	_ = db.SingletonNodeBaseDB().DispenseStart(systemUUID, start.WorkSpaceID, db.Pulling)
}

// 发布Docker进度
func dispenseProgress(systemUUID string, message []byte) {
	funName := "dispenseProgress"
	progress := struct {
		MessageID   int64   `json:"messageID"`
		WorkSpaceID string  `json:"workSpaceID"`
		Layers      []Layer `json:"layers"`
	}{}
	if err := json.Unmarshal(message, &progress); err != nil {
		errString := fmt.Sprintf("%s json.Unmarshal Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return
	}
	var layers []string
	for _, v := range progress.Layers {
		layers = append(layers, fmt.Sprintf("%s : %s", v.LayerId, v.Progress))
	}
	result := strings.Join(layers, "\n")
	SingletonNodes().SetNodeDockerStatus(systemUUID, progress.WorkSpaceID, db.Pulling)
	_ = db.SingletonNodeBaseDB().DispenseProgress(systemUUID, progress.WorkSpaceID, result, db.Pulling)
}

// 发布Docker完成
func dispenseComplete(systemUUID string, message []byte) {
	funName := "dispenseComplete"
	complete := struct {
		MessageID   int64   `json:"messageID"`
		WorkSpaceID string  `json:"workSpaceID"`
		SaveDir     string  `json:"saveDir"`
		Layers      []Layer `json:"layers"`
	}{}
	if err := json.Unmarshal(message, &complete); err != nil {
		errString := fmt.Sprintf("%s json.Unmarshal Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return
	}
	var layers []string
	for _, v := range complete.Layers {
		layers = append(layers, fmt.Sprintf("%s : %s", v.LayerId, v.Progress))
	}
	result := strings.Join(layers, "\n")
	SingletonNodes().SetNodeDockerStatus(systemUUID, complete.WorkSpaceID, db.PullCompleted)
	_ = db.SingletonNodeBaseDB().DispenseComplete(systemUUID, complete.WorkSpaceID, complete.SaveDir, result, db.PullCompleted)
}

// 启动工作空间
func startupWorkSpace(systemUUID string, message []byte) {
	funName := "startupWorkSpace"
	start := struct {
		MessageID   int64  `json:"messageID"`
		WorkSpaceID string `json:"workSpaceID"`
	}{}
	if err := json.Unmarshal(message, &start); err != nil {
		errString := fmt.Sprintf("%s json.Unmarshal Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return
	}
	SingletonNodes().SetNodeDockerStatus(systemUUID, start.WorkSpaceID, db.Startuping)
	_ = db.SingletonNodeBaseDB().StartWorkSpace(systemUUID, start.WorkSpaceID, db.Startuping)
}

// 启动完成
func startupComplete(systemUUID string, message []byte) {
	funName := "startupComplete"
	startup := struct {
		MessageID   int64  `json:"messageID"`
		WorkSpaceID string `json:"workSpaceID"`
		Success     bool   `json:"success"`
	}{}
	if err := json.Unmarshal(message, &startup); err != nil {
		errString := fmt.Sprintf("%s json.Unmarshal Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return
	}
	_ = db.SingletonNodeBaseDB().DispenseStart(systemUUID, startup.WorkSpaceID, db.Pulling)
	//重新加载此算力节点下的所有docker信息
	node := SingletonNodes().FindNode(systemUUID)
	if node == nil {
		errString := fmt.Sprintf("%s FindNode Failed not found node systemUUID=[%s]", funName, systemUUID)
		log4plus.Error(errString)
		return
	}
	node.Base.LoadDocker(systemUUID)
	//根据返回信息设置状态
	if startup.Success {
		SingletonNodes().SetNodeDockerStatus(systemUUID, startup.WorkSpaceID, db.StartupCompleted)
		_ = db.SingletonNodeBaseDB().StartupWorkSpace(systemUUID, startup.WorkSpaceID, db.StartupCompleted)
	} else {
		SingletonNodes().SetNodeDockerStatus(systemUUID, startup.WorkSpaceID, db.StartupFailed)
		_ = db.SingletonNodeBaseDB().StartupWorkSpace(systemUUID, startup.WorkSpaceID, db.StartupFailed)
	}
}

func nodePing(systemUUID, ip string, message []byte) {
	funName := "nodePing"
	heart := struct {
		MessageID int64         `json:"messageID"`
		OS        string        `json:"os"`
		CPU       string        `json:"cpu"`
		GPU       string        `json:"gpu"`
		Memory    uint64        `json:"memory"`
		Version   string        `json:"version"`
		GpuDriver string        `json:"gpuDriver"`
		Dockers   []DockerArray `json:"dockers"`
	}{}
	if err := json.Unmarshal(message, &heart); err != nil {
		errString := fmt.Sprintf("%s json.Unmarshal Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return
	}
	SingletonNodes().SetNodeHeart(systemUUID, ip, heart.OS, heart.CPU, heart.GPU, heart.Version, heart.GpuDriver, heart.Memory)
	if node := SingletonNodes().FindNode(systemUUID); node != nil {
		for _, docker := range node.Base.Dockers {
			deleteSame := false
			for _, clientDocker := range heart.Dockers {
				if strings.EqualFold(docker.WorkSpaceID, clientDocker.WorkSpaceID) {
					if docker.Deleted == clientDocker.Deleted {
						deleteSame = true
						break
					}
				}
			}
			if !deleteSame {
				log4plus.Warn(fmt.Sprintf("%s The database record status is inconsistent with the client-side state workSpaceID=[%s] docker.Deleted=[%d]",
					funName, docker.WorkSpaceID, docker.Deleted))
			}
		}
	}
	_ = db.SingletonNodeBaseDB().NodeHeart(systemUUID, ip, heart.OS, heart.CPU, heart.GPU, heart.GpuDriver, heart.Version, heart.Memory)
	sendPong(systemUUID)
}

func pullFail(systemUUID string, message []byte) {
	funName := "pullFail"
	fail := struct {
		MessageID   int64  `json:"messageID"`
		WorkSpaceID string `json:"workSpaceID"`
		Failure     string `json:"failure"`
	}{}
	if err := json.Unmarshal(message, &fail); err != nil {
		errString := fmt.Sprintf("%s json.Unmarshal Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return
	}
	SingletonNodes().SetNodeDockerStatus(systemUUID, fail.WorkSpaceID, db.PullFailed)
	_ = db.SingletonNodeBaseDB().DispenseFail(systemUUID, fail.WorkSpaceID, db.PullFailed)
}

func startupFail(systemUUID string, message []byte) {
	funName := "startupFail"
	fail := struct {
		MessageID   int64  `json:"messageID"`
		WorkSpaceID string `json:"workSpaceID"`
		Failure     string `json:"failure"`
	}{}
	if err := json.Unmarshal(message, &fail); err != nil {
		errString := fmt.Sprintf("%s json.Unmarshal Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return
	}
	SingletonNodes().SetNodeDockerStatus(systemUUID, fail.WorkSpaceID, db.StartupFailed)
	_ = db.SingletonNodeBaseDB().DispenseFail(systemUUID, fail.WorkSpaceID, db.StartupFailed)
}

func deleteContainer(systemUUID string, message []byte) {
	funName := "deleteContainer"
	container := struct {
		MessageID   int64  `json:"messageID"`
		WorkSpaceID string `json:"workSpaceID"`
		Failure     string `json:"failure"`
	}{}
	if err := json.Unmarshal(message, &container); err != nil {
		errString := fmt.Sprintf("%s json.Unmarshal Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return
	}
	SingletonNodes().SetNodeDockerStatus(systemUUID, container.WorkSpaceID, db.PullCompleted)
	_ = db.SingletonNodeBaseDB().SetDockerStatus(systemUUID, container.WorkSpaceID, db.PullCompleted)
}

func deleteDocker(systemUUID string, message []byte) {
	funName := "delWorkSpace"
	del := struct {
		MessageID   int64  `json:"messageID"`
		WorkSpaceID string `json:"workSpaceID"`
	}{}
	if err := json.Unmarshal(message, &del); err != nil {
		errString := fmt.Sprintf("%s json.Unmarshal Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return
	}
	_ = db.SingletonNodeBaseDB().DelWorkSpace(systemUUID, del.WorkSpaceID)
	SingletonNodes().DeleteDocker(systemUUID, del.WorkSpaceID)
}

func comfyUIStartupSuccess(systemUUID string, message []byte) {
	funName := "comfyUIStartupSuccess"
	comfyUI := struct {
		MessageID   int64  `json:"messageID"`
		WorkSpaceID string `json:"workSpaceID"`
	}{}
	if err := json.Unmarshal(message, &comfyUI); err != nil {
		errString := fmt.Sprintf("%s json.Unmarshal Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return
	}
	if node := SingletonNodes().FindNode(systemUUID); node == nil {
		errString := fmt.Sprintf("%s FindNode Failed not found systemUUID=[%s]", funName, systemUUID)
		log4plus.Error(errString)
		return
	} else {
		for _, docker := range node.Base.Dockers {
			if strings.EqualFold(docker.WorkSpaceID, comfyUI.WorkSpaceID) {
				if docker.State == db.StartupCompleted {
					docker.State = db.StartupSuccess
					_ = db.SingletonNodeBaseDB().SetDockerStatus(systemUUID, comfyUI.WorkSpaceID, db.StartupSuccess)
					sendMeasure(systemUUID, comfyUI.WorkSpaceID, docker)
					return
				}
			}
		}
	}
}

type MonitorData struct {
	SystemUUID       string               `json:"systemUUID"`
	NodeExporterData monitor.ExporterData `json:"exporterData"`
}

func nodeExporterData(systemUUID string, message []byte) {
	funName := "nodeExporterData"
	exporterData := struct {
		MessageID        int64                `json:"messageID"`
		NodeExporterData monitor.ExporterData `json:"exporterData"`
	}{}
	if err := json.Unmarshal(message, &exporterData); err != nil {
		errString := fmt.Sprintf("%s json.Unmarshal Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return
	}
	if node := SingletonNodes().FindNode(systemUUID); node == nil {
		errString := fmt.Sprintf("%s FindNode Failed not found systemUUID=[%s]", funName, systemUUID)
		log4plus.Error(errString)
		return
	} else {
		monitorData := MonitorData{
			SystemUUID:       systemUUID,
			NodeExporterData: exporterData.NodeExporterData,
		}
		SingletonNodes().ExporterDataCh <- monitorData
		log4plus.Info("%s monitor Data systemUUID=[%s] len=[%d]", funName, systemUUID, len(message))
	}
}
