package process

import (
	"encoding/json"
	"fmt"
	"github.com/jinzhu/copier"
	log4plus "github.com/nextGPU/include/log4go"
	"github.com/nextGPU/ng-cloud/configure"
	"github.com/nextGPU/ng-cloud/db"
	ws "github.com/nextGPU/ng-cloud/net/websocket"
	"strings"
	"sync"
	"time"
)

type TaskState int

const (
	WaitingTask    TaskState = iota //任务处于全局队列等待状态
	PublishingTask                  //任务正在分发
	PublishedTask
	RunningTask
	CompletedTask
	FailedTask
)

type TaskSlot struct {
	//任务本身
	Session               string          //任务发起session
	WorkSpaceID           string          //工作空间ID
	WorkflowTitle         string          //工作流Title
	TaskID                string          //任务ID
	MaxSystemSlotSize     int64           //系统中最大的槽数量
	SubID                 string          //子任务ID
	ImageCount            int64           //图片数量（一直是：1）
	ImagePath             string          //OSS保存图片的路径
	InputImageScaledRatio int64           //尺寸比例
	Width                 int64           //生成图宽度
	Height                int64           //生成图高度
	TaskData              TaskRequestData //任务数据
	//匹配算力
	SystemUUID string //算力节点
	//错误算力
	ErrorNodes []string //错误节点
	//任务进度情况
	State          TaskState   //任务状态
	PublishTime    int64       //任务发起时间
	PushQueueTime  int64       //任务放入队列时间（和发起时间有区别，当任务出错时入队列时间不等于发起时间）
	InitQueuePos   int64       //入队列时在全局队列中的位置（应剔除已经分发了的任务）
	CurQueuePos    int64       //当前在全局队列中的位置
	InLocalTime    int64       //入本地队列的时间
	InLocalInitPos int64       //入本地队列的位置
	CurLocalPos    int64       //当前本地队列中的位置
	StartTime      int64       //开始执行时间
	EstimateMs     int64       //预估完成时间
	EstimateTime   *time.Timer //预估计时器
	CompletedTime  int64       //完成时间
	//任务进度显示
	InitWaitCount int64 //初始等待队列数量（此数值一点计算完成不再发生变化）
	CurWaitCount  int64 //当前等待队列数量
	//任务结束后的OSS图片保存
	OSSUrls []string
}

type WaitingSlots struct {
	lock  sync.Mutex
	slots []*TaskSlot
}

func (n *WaitingSlots) Sort() {
	n.lock.Lock()
	defer n.lock.Unlock()
	for index, v := range n.slots {
		tmpCurQueuePos := int64(index)
		if v.InitQueuePos != 0 && tmpCurQueuePos <= v.InitQueuePos {
			v.CurQueuePos = tmpCurQueuePos
		}
		tmpCurWaitCount := v.CurQueuePos + v.MaxSystemSlotSize
		if tmpCurWaitCount <= v.CurWaitCount {
			v.CurWaitCount = tmpCurWaitCount
		}
	}
}

func (n *WaitingSlots) PushSlot(slot *TaskSlot) {
	n.lock.Lock()
	slot.SystemUUID = ""
	slot.InLocalInitPos = 0
	slot.CurLocalPos = 0
	slot.State = WaitingTask
	n.slots = append(n.slots, slot)
	n.lock.Unlock()
	//通知
	SingletonNodes().TaskDispense()
	//排序
	n.Sort()
}

func (n *WaitingSlots) CleanSlot(subID string) {
	n.lock.Lock()
	var newSlots []*TaskSlot
	for _, slot := range n.slots {
		if !strings.EqualFold(slot.SubID, subID) {
			slot.CurQueuePos = 0
			newSlots = append(newSlots, slot)
		}
	}
	n.slots = newSlots
	n.lock.Unlock()
	//排序
	n.Sort()
}

func (n *WaitingSlots) SlotSize() int {
	n.lock.Lock()
	defer n.lock.Unlock()
	return len(n.slots)
}

func isExist(workspaceID string, workspaceIDs []string) bool {
	for _, v := range workspaceIDs {
		if strings.EqualFold(workspaceID, v) {
			return true
		}
	}
	return false
}

func (n *WaitingSlots) PopSlot(systemUUID string, workspaceIDs []string) *TaskSlot {
	n.lock.Lock()
	defer n.lock.Unlock()
	for _, slot := range n.slots {
		if isExist(slot.WorkSpaceID, workspaceIDs) {
			//滤过出错的算力节点
			isErrNode := false
			for _, errNode := range slot.ErrorNodes {
				if strings.EqualFold(systemUUID, errNode) {
					isErrNode = true
					break
				}
			}
			if !isErrNode {
				return slot
			}
		}
	}
	return nil
}

func (n *WaitingSlots) SlotPos(subID string) (bool, int64) {
	n.lock.Lock()
	defer n.lock.Unlock()
	for index, slot := range n.slots {
		if strings.EqualFold(slot.SubID, subID) {
			return true, int64(index)
		}
	}
	return false, 0
}

func (n *WaitingSlots) PopSlotFromSystemUUID(systemUUID string) *TaskSlot {
	n.lock.Lock()
	defer n.lock.Unlock()
	for _, slot := range n.slots {
		if strings.EqualFold(slot.SystemUUID, systemUUID) {
			return slot
		}
	}
	return nil
}

func (n *WaitingSlots) PopFirstSlot() *TaskSlot {
	n.lock.Lock()
	defer n.lock.Unlock()
	for _, slot := range n.slots {
		return slot
	}
	return nil
}

func (n *WaitingSlots) PushHead(subTask *TaskSlot) {
	subTask.State = WaitingTask
	subTask.SystemUUID = ""
	subTask.CurQueuePos = 0

	n.lock.Lock()
	n.slots = append([]*TaskSlot{subTask}, n.slots...)
	n.lock.Unlock()
	//通知
	SingletonNodes().TaskSlotCh <- struct{}{}
	//排序
	n.Sort()
}

func NewWaitingSlots() *WaitingSlots {
	idleSlots := &WaitingSlots{}
	return idleSlots
}

type ProcessingSlots struct {
	lock  sync.Mutex
	slots []*TaskSlot
}

func (n *ProcessingSlots) PushSlot(slot *TaskSlot, subID string) {
	funName := "PushSlot"
	n.lock.Lock()
	slot.SubID = subID
	slot.State = RunningTask
	slot.CurQueuePos = 0
	n.slots = append(n.slots, slot)
	log4plus.Info(fmt.Sprintf("%s subID++++++++++++ subID=[%s]", funName, subID))
	n.lock.Unlock()

	n.Sort()
}

func (n *ProcessingSlots) findSlot(subID string) *TaskSlot {
	n.lock.Lock()
	defer n.lock.Unlock()

	for _, slot := range n.slots {
		if strings.EqualFold(slot.SubID, subID) {
			return slot
		}
	}
	return nil
}

func (n *ProcessingSlots) CleanSlot(subID string) {
	funName := "CleanSlot"
	n.lock.Lock()
	defer n.lock.Unlock()
	log4plus.Info(fmt.Sprintf("%s subID---------- subID=[%s]", funName, subID))

	var newSlots []*TaskSlot
	for _, slot := range n.slots {
		if !strings.EqualFold(slot.SubID, subID) {
			newSlots = append(newSlots, slot)
		}
	}
	n.slots = newSlots
}

func (n *ProcessingSlots) SlotSize() int {
	n.lock.Lock()
	defer n.lock.Unlock()
	return len(n.slots)
}

func (n *ProcessingSlots) Sort() {
	n.lock.Lock()
	defer n.lock.Unlock()
	for _, v := range n.slots {
		if v.State == RunningTask {
			v.CurWaitCount = 0
			v.CurQueuePos = 0
			v.CurLocalPos = 0
		} else if v.State == PublishingTask || v.State == PublishedTask {
			if node := SingletonNodes().FindNode(v.SystemUUID); node != nil {
				if exist, waitCount := node.Match.GetTaskPos(v.SubID); exist {
					v.CurWaitCount = int64(waitCount)
					v.CurLocalPos = int64(waitCount)
				}
			}
		}
	}
}

func NewProcessingSlots() *ProcessingSlots {
	runningSlots := &ProcessingSlots{}
	return runningSlots
}

type (
	TaskRequestData struct {
		Title string `json:"title"`
		//InputType             string          `json:"inputType"`
		InputImageScaledRatio int64           `json:"inputImageScaledRatio,omitempty"`
		SystemUUID            string          `json:"systemUUID,omitempty"`
		ImageCount            int64           `json:"imageCount,omitempty"`
		ImagePath             string          `json:"imagePath,omitempty"`
		TaskID                string          `json:"taskID"`
		Data                  json.RawMessage `json:"data"`
	}
	ShowTaskSlot struct {
		SubID          string
		State          TaskState
		SystemUUID     string
		InitQueuePos   int64
		CurQueuePos    int64
		InLocalInitPos int64
		CurLocalPos    int64
		InitWaitCount  int64
		CurWaitCount   int64
		PublishTime    int64
		StartTime      int64
		EstimateMs     int64
		CompletedTime  int64
		OSSUrls        []string
	}
	ShowSessionTask struct {
		TaskID      string
		Session     string
		State       TaskState
		ImageNum    int64
		PublishTime int64
		ShowTasks   []ShowTaskSlot
	}
	SessionTask struct {
		Session               string
		WorkSpaceID           string
		Title                 string
		TaskID                string
		MaxSystemSlotSize     int64
		PublishTime           int64
		ImageNum              int64
		ImagePath             string
		Ratio                 string
		Width                 int64
		Height                int64
		InputImageScaledRatio int64
		SpareImageNum         int64
		State                 TaskState
		TaskData              TaskRequestData
		Tasks                 []*TaskSlot
	}
	UserSession struct {
		Session  string
		ClientIP string
		Match    *UserMatch
		Net      []*ws.Linker
	}
	ShowUserSession struct {
		Session string
		IP      []string
		Port    []string
		Handle  []uint64
	}
)

type UserSessions struct {
	lock                sync.Mutex
	sessions            map[string]*UserSession
	userWs              *ws.WSServer
	taskLock            sync.Mutex
	tasks               []*SessionTask
	waitingSlots        *WaitingSlots
	processingSlots     *ProcessingSlots
	SessionDisconnectCh chan string
}

var gUserSessions *UserSessions

// session
func (n *UserSessions) NewSession(session, clientIp string) (error, *UserSession) {
	userSession := &UserSession{
		Session:  session,
		ClientIP: clientIp,
	}
	userSession.Match = NewUserMatch(userSession)
	n.lock.Lock()
	n.sessions[userSession.Session] = userSession
	n.lock.Unlock()
	return nil, userSession
}

func (n *UserSessions) DeleteSessionLinker(session string, handle uint64) {
	n.lock.Lock()
	defer n.lock.Unlock()
	userSession, Ok := n.sessions[session]
	if Ok {
		for i := 0; i < len(userSession.Net); {
			if userSession.Net[i].Handle() == handle {
				_ = userSession.Net[i].Close()
				userSession.Net = append(userSession.Net[:i], userSession.Net[i+1:]...)
			} else {
				i++
			}
		}
		if len(userSession.Net) <= 0 {
			userSession.Net = userSession.Net[:0]
			userSession.Match.CloseCh <- true
			delete(n.sessions, session)
		}
	}
}

func (n *UserSessions) DeleteSession(session string) {
	n.lock.Lock()
	defer n.lock.Unlock()
	userSession, Ok := n.sessions[session]
	if Ok {
		for _, v := range userSession.Net {
			_ = v.Close()
		}
		userSession.Net = userSession.Net[:0]
		userSession.Match.CloseCh <- true
	}
	delete(n.sessions, session)
}

func (n *UserSessions) FindSession(session string) *UserSession {
	n.lock.Lock()
	defer n.lock.Unlock()
	userSession, Ok := n.sessions[session]
	if Ok {
		return userSession
	}
	return nil
}

func (n *UserSessions) Exist(session string) bool {
	return n.FindSession(session) != nil
}

func (n *UserSessions) GetSessions() []ShowUserSession {
	n.lock.Lock()
	defer n.lock.Unlock()
	var sessions []ShowUserSession
	for _, session := range n.sessions {
		tmpSession := ShowUserSession{
			Session: session.Session,
		}
		tmpSession.Handle = session.Match.SelfHandle()
		tmpSession.IP, tmpSession.Port = session.Match.IpPort()
		sessions = append(sessions, tmpSession)
	}
	return sessions
}

func (n *UserSessions) FindProcessing(subID string) *TaskSlot {
	if n.processingSlots != nil {
		return n.processingSlots.findSlot(subID)
	}
	return nil
}

func (n *UserSessions) CompletedSubTask(subID string, endTime int64) bool {
	if subTask := n.processingSlots.findSlot(subID); subTask != nil {
		n.processingSlots.CleanSlot(subID)
		subTask.State = CompletedTask
		subTask.CompletedTime = time.Now().UnixMilli()
		completed := n.checkCompletedTask(subTask.Session, subTask.TaskID)
		if completed {
			//判断是否成功
			if n.checkSuccessTask(subTask.Session, subTask.TaskID) {
				sendSyncs(subID, ImageGenerateSuccess)
			}
			n.checkSortTask(subTask.Session, subTask.TaskID)
			//回写DB
			_ = db.SingletonTaskDB().SetTaskResult(subTask.TaskID, time2String(endTime))
			return true
		}
	}
	return false
}

func (n *UserSessions) ExistOtherNodes(subTask *TaskSlot) bool {
	curNodes := SingletonNodes().GetPassNodes(subTask.WorkflowTitle)
	for _, node := range curNodes {
		for _, errNode := range subTask.ErrorNodes {
			if !strings.EqualFold(node.SystemUUID, errNode) {
				return true
			}
		}
	}
	return false
}

func (n *UserSessions) FailedSubTask(subID string, subTask *TaskSlot) {
	funName := "FailedSubTask"
	subTask.State = FailedTask
	subTask.CompletedTime = time.Now().UnixMilli()
	n.processingSlots.CleanSlot(subID)

	//判断是否任务完成
	completed := n.checkCompletedTask(subTask.Session, subTask.TaskID)
	log4plus.Info("%s checkCompletedTask taskID=[%s] Session=[%s] completed=[%t]", funName, subTask.TaskID, subTask.Session, completed)
	if completed {
		//判断是否成功
		success := n.checkSuccessTask(subTask.Session, subTask.TaskID)
		log4plus.Info("%s checkSuccessTask taskID=[%s] Session=[%s] success=[%t]", funName, subTask.TaskID, subTask.Session, success)

		n.checkSortTask(subTask.Session, subTask.TaskID)
		log4plus.Info("%s checkSortTask taskID=[%s] Session=[%s]", funName, subTask.TaskID, subTask.Session)

		//回写DB
		_ = db.SingletonTaskDB().SetTaskFailedResult(subTask.TaskID)
		log4plus.Info("%s SetTaskFailedResult taskID=[%s]", funName, subTask.TaskID)

		//断开WS链接
		n.DeleteSession(subTask.Session)
		log4plus.Info("%s DeleteSession Session=[%s]", funName, subTask.Session)
	}
}

func (n *UserSessions) DeleteProcessing(subID string) {
	if n.processingSlots != nil {
		n.processingSlots.CleanSlot(subID)
	}
}

func (n *UserSessions) PushWaitHead(subTask *TaskSlot) {
	n.waitingSlots.PushHead(subTask)
	if task := n.GetTask(subTask.Session, subTask.TaskID); task != nil {
		task.State = WaitingTask
	}
}

func (n *UserSessions) PopWait() *TaskSlot {
	if n.waitingSlots != nil {
		return n.waitingSlots.PopFirstSlot()
	}
	return nil
}

func (n *UserSessions) PopSlot(systemUUID string, workspaceIDs []string) *TaskSlot {
	if n.waitingSlots != nil {
		return n.waitingSlots.PopSlot(systemUUID, workspaceIDs)
	}
	return nil
}

func (n *UserSessions) PopSlotFromSystemUUID(systemUUID string) *TaskSlot {
	if n.waitingSlots != nil {
		return n.waitingSlots.PopSlotFromSystemUUID(systemUUID)
	}
	return nil
}

func (n *UserSessions) PushProcessing(subTask *TaskSlot) {
	n.waitingSlots.CleanSlot(subTask.SubID)
	n.processingSlots.PushSlot(subTask, subTask.SubID)
	task := n.GetTask(subTask.Session, subTask.TaskID)
	if task != nil {
		if task.State == WaitingTask {
			task.State = RunningTask
			publishTime := time.Now().Format("2006-01-02 15:04:05")
			_ = db.SingletonTaskDB().SetPublishTime(subTask.TaskID, publishTime)
		}
	}
	_ = db.SingletonTaskDB().SetSubTaskPublish(subTask.TaskID, subTask.SubID, subTask.SystemUUID, time.Now().Format("2006-01-02 15:04:05"))
}

func (n *UserSessions) WaitQueueSize() int64 {
	return int64(n.waitingSlots.SlotSize())
}

func (n *UserSessions) WaitQueuePos(subID string) (bool, int64) {
	if n.waitingSlots != nil {
		return n.waitingSlots.SlotPos(subID)
	}
	return false, 0
}

func (n *UserSessions) ProcessingNum() int {
	if n.processingSlots != nil {
		return n.processingSlots.SlotSize()
	}
	return 0
}

//main task

func (n *UserSessions) NewTask(session string, taskData TaskRequestData, message []byte) *SessionTask {
	funName := "NewTask"
	err, workflow, workspaceIDs := db.SingletonNodeBaseDB().Title2Workflow(taskData.Title)
	if err != nil {
		errString := fmt.Sprintf(" %s Title2Workflow Failed taskData.WorkflowTitle=[%s] err=[%s]", funName, taskData.Title, err.Error())
		log4plus.Error(errString)
		return nil
	}
	if len(workspaceIDs) != 1 {
		errString := fmt.Sprintf(" %s Title2Workflow Result taskData.WorkflowTitle=[%s] len(workspaceIDs)=[%d]", funName, taskData.Title, len(workspaceIDs))
		log4plus.Error(errString)
		return nil
	}
	mainData, subData, ratio, width, height, err := SingletonWorkflows().SetTemplateData(taskData.Title, json.RawMessage(workflow.Template), workflow.Configure, taskData.Data)
	if err != nil {
		errString := fmt.Sprintf(" %s SetTemplateData Failed taskData.JobType=[%s] err=[%s]", funName, taskData.Title, err.Error())
		log4plus.Error(errString)
		return nil
	}
	if width == 0 || height == 0 {
		errString := fmt.Sprintf(" %s SetTemplateData Failed title=[%s] ratio=[%s] width=[%d] height=[%d]", funName, taskData.Title, ratio, width, height)
		log4plus.Error(errString)
		return nil
	}
	//更新这个任务所支持的算力节点重新加载工作流信息
	for _, workspaceID := range workspaceIDs {
		systemUUIDs := SingletonNodes().GetSystemUUIDs(workspaceID)
		for _, systemUUID := range systemUUIDs {
			SingletonNodes().ReloadWorkflow(systemUUID)
		}
	}
	taskID := taskData.TaskID
	workspaceID := workspaceIDs[0]
	err = db.SingletonTaskDB().InsertMainTask(taskID, session, taskData.Title, taskData.ImageCount, db.NormalTask, string(taskData.Data), workspaceID, mainData, string(message))
	if err == nil {
		estimateWaitCount := SingletonNodes().EstimateWaitCount(taskData.Title)
		task := &SessionTask{
			Session:               session,
			WorkSpaceID:           workspaceID,
			Title:                 taskData.Title,
			TaskID:                taskID,
			MaxSystemSlotSize:     estimateWaitCount,
			PublishTime:           time.Now().UnixMilli(),
			State:                 WaitingTask,
			ImageNum:              taskData.ImageCount,
			ImagePath:             taskData.ImagePath,
			Ratio:                 ratio,
			Width:                 width,
			Height:                height,
			InputImageScaledRatio: taskData.InputImageScaledRatio,
		}
		task.TaskData.Title = taskData.Title
		_ = copier.Copy(&task.TaskData.Data, &mainData)
		n.AddTask(task, workflow, subData, taskID, taskData, true)
		return task
	}
	errString := fmt.Sprintf(" %s InsertMainTask Failed err=[%s]", funName, err.Error())
	log4plus.Error(errString)
	return nil
}

func (n *UserSessions) AddTask(sessionTask *SessionTask, workflow db.DockerWorkflow, subBody json.RawMessage, taskID string, taskData TaskRequestData, writeDB bool) {
	funName := "AddTask"
	n.taskLock.Lock()
	n.tasks = append(n.tasks, sessionTask)
	n.taskLock.Unlock()
	for i := 1; i <= int(sessionTask.ImageNum); i++ {
		//new seed
		seedSubBody, err := SingletonWorkflows().SetSeed(subBody, workflow)
		if err != nil {
			errString := fmt.Sprintf("%s setSeed failed err=[%s]", funName, err.Error())
			log4plus.Error(errString)
		}
		//new slot
		subID := fmt.Sprintf("%s_%04d", taskID, i)
		subTask := &TaskSlot{
			Session:               sessionTask.Session,
			WorkSpaceID:           sessionTask.WorkSpaceID,
			WorkflowTitle:         sessionTask.Title,
			TaskID:                sessionTask.TaskID,
			MaxSystemSlotSize:     sessionTask.MaxSystemSlotSize,
			SubID:                 subID,
			ImageCount:            1,
			ImagePath:             sessionTask.ImagePath,
			InputImageScaledRatio: sessionTask.InputImageScaledRatio,
			Width:                 sessionTask.Width,
			Height:                sessionTask.Height,
			SystemUUID:            taskData.SystemUUID,
			State:                 WaitingTask,
			PublishTime:           time.Now().UnixMilli(),
		}
		subTask.TaskData.Title = sessionTask.TaskData.Title
		_ = copier.Copy(&subTask.TaskData.Data, &seedSubBody)
		sessionTask.Tasks = append(sessionTask.Tasks, subTask)
	}
	//通知用户每个子任务的进度
	for _, subTask := range sessionTask.Tasks {
		if SingletonNodes().GetIdleSlotNum() > 0 {
			subTask.PushQueueTime = time.Now().UnixMilli()
			n.waitingSlots.PushSlot(subTask)
			if subTask.State == PublishingTask || subTask.State == PublishedTask || subTask.State == RunningTask {
				//任务已经分发处理，并不在全局队列中
				subTask.InitQueuePos = 0
				useNode := SingletonNodes().FindNode(subTask.SystemUUID)
				if useNode != nil {
					exist, waitCount := useNode.Match.GetTaskPos(subTask.SubID)
					if exist {
						subTask.InitWaitCount = int64(waitCount)
					} else {
						subTask.InitWaitCount = 0
					}
					subTask.CurWaitCount = subTask.InitWaitCount
					subTask.InLocalInitPos = subTask.CurWaitCount
					subTask.CurLocalPos = subTask.InLocalInitPos
				}
			} else if subTask.State == WaitingTask {
				//任务在全局队列中
				if exist, pos := n.WaitQueuePos(subTask.SubID); exist {
					subTask.InitQueuePos = pos
				} else {
					subTask.InitQueuePos = n.WaitQueueSize()
				}
				subTask.CurQueuePos = subTask.InitQueuePos
				subTask.InLocalInitPos = subTask.MaxSystemSlotSize
				subTask.CurLocalPos = subTask.InLocalInitPos
				subTask.InitWaitCount = subTask.InitQueuePos + subTask.MaxSystemSlotSize
				subTask.CurWaitCount = subTask.InitWaitCount
			}
		} else {
			//任务在全局队列中
			subTask.PushQueueTime = time.Now().UnixMilli()
			n.waitingSlots.PushSlot(subTask)
			if exist, pos := n.WaitQueuePos(subTask.SubID); exist {
				subTask.InitQueuePos = pos
			} else {
				subTask.InitQueuePos = n.WaitQueueSize()
			}
			subTask.InLocalInitPos = subTask.MaxSystemSlotSize
			subTask.CurLocalPos = subTask.MaxSystemSlotSize
			subTask.InitWaitCount = subTask.InitQueuePos + subTask.MaxSystemSlotSize
			subTask.CurWaitCount = subTask.InitWaitCount
		}
		//write db
		if writeDB {
			data, _ := json.Marshal(subTask.TaskData.Data)
			_ = db.SingletonTaskDB().InsertSubTask(subTask.TaskID, subTask.SubID, string(data), sessionTask.WorkSpaceID)
		}
	}
}

func (n *UserSessions) NewTaskFromDB(session string, historyTask *db.HistoryTask) {
	funName := "NewTaskFromDB"
	estimateWaitCount := SingletonNodes().EstimateWaitCount(historyTask.WorkflowTitle)
	publishTime, _ := time.ParseInLocation("2006-01-02 15:04:05", historyTask.PublishTime, time.Local)
	//解析用户请求数据
	request := struct {
		Title string `json:"title"`
		//InputType             string          `json:"inputType"`
		InputImageScaledRatio int64           `json:"inputImageScaledRatio,omitempty"`
		SystemUUID            string          `json:"systemUUID,omitempty"`
		ImageCount            int64           `json:"imageCount"`
		ImagePath             string          `json:"imagePath"`
		Data                  json.RawMessage `json:"data"`
	}{}
	if err := json.Unmarshal([]byte(historyTask.Request), &request); err != nil {
		errString := fmt.Sprintf(" %s Unmarshal Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return
	}
	//获取工作流
	err, workflow, workspaceIDs := db.SingletonNodeBaseDB().Title2Workflow(request.Title)
	if err != nil {
		errString := fmt.Sprintf(" %s Title2Workflow Failed taskData.WorkflowTitle=[%s] err=[%s]", funName, request.Title, err.Error())
		log4plus.Error(errString)
		return
	}
	if len(workspaceIDs) != 1 {
		errString := fmt.Sprintf(" %s Title2Workflow Result taskData.WorkflowTitle=[%s] len(workspaceIDs)=[%d]", funName, request.Title, len(workspaceIDs))
		log4plus.Error(errString)
		return
	}
	_, _, ratio, width, height, err := SingletonWorkflows().SetTemplateData(request.Title, json.RawMessage(workflow.Template), workflow.Configure, request.Data)
	if err != nil {
		errString := fmt.Sprintf(" %s SetTemplateData Failed taskData.JobType=[%s] err=[%s]", funName, request.Title, err.Error())
		log4plus.Error(errString)
		return
	}
	if width == 0 || height == 0 {
		errString := fmt.Sprintf(" %s SetTemplateData Failed title=[%s] ratio=[%s] width=[%d] height=[%d]", funName, request.Title, ratio, width, height)
		log4plus.Error(errString)
		return
	}
	//设置主任务
	sessionTask := &SessionTask{
		Session:               session,
		WorkSpaceID:           historyTask.WorkspaceID,
		Title:                 historyTask.WorkflowTitle,
		TaskID:                historyTask.TaskID,
		MaxSystemSlotSize:     estimateWaitCount,
		PublishTime:           publishTime.UnixMilli(),
		ImageNum:              int64(len(historyTask.SubTasks)),
		ImagePath:             request.ImagePath,
		Ratio:                 ratio,
		Width:                 width,
		Height:                height,
		InputImageScaledRatio: historyTask.InputImageScaledRatio,
		State:                 WaitingTask,
	}
	sessionTask.TaskData.Title = historyTask.WorkflowTitle
	sessionTask.TaskData.ImagePath = request.ImagePath
	sessionTask.TaskData.InputImageScaledRatio = historyTask.InputImageScaledRatio
	//sessionTask.TaskData.InputType = request.InputType
	data := json.RawMessage(historyTask.GenerateData)
	_ = copier.Copy(&sessionTask.TaskData.Data, &data)

	n.taskLock.Lock()
	n.tasks = append(n.tasks, sessionTask)
	n.taskLock.Unlock()

	//设置子任务
	for i := 1; i <= int(historyTask.SubNum); i++ {
		//new slot
		subTask := &TaskSlot{
			Session:               session,
			WorkSpaceID:           historyTask.SubTasks[i-1].WorkspaceID,
			WorkflowTitle:         historyTask.WorkflowTitle,
			TaskID:                historyTask.TaskID,
			MaxSystemSlotSize:     estimateWaitCount,
			SubID:                 historyTask.SubTasks[i-1].SubID,
			ImageCount:            1,
			ImagePath:             request.ImagePath,
			InputImageScaledRatio: historyTask.InputImageScaledRatio,
			Width:                 width,
			Height:                height,
			SystemUUID:            "",
			State:                 WaitingTask,
			PublishTime:           publishTime.UnixMilli(),
			InitQueuePos:          n.WaitQueueSize(),
			CurQueuePos:           n.WaitQueueSize(),
		}
		subTask.TaskData.Title = subTask.WorkflowTitle
		//new seed
		subBody := json.RawMessage(historyTask.SubTasks[i-1].Input)
		seedSubBody, errSeed := SingletonWorkflows().SetSeed(subBody, workflow)
		if errSeed != nil {
			errString := fmt.Sprintf("%s setSeed failed err=[%s]", funName, errSeed.Error())
			log4plus.Error(errString)
		}
		_ = copier.Copy(&subTask.TaskData.Data, &seedSubBody)
		sessionTask.Tasks = append(sessionTask.Tasks, subTask)
	}
	//通知用户每个子任务的进度
	for _, subTask := range sessionTask.Tasks {
		if SingletonNodes().GetIdleSlotNum() > 0 {
			subTask.PushQueueTime = time.Now().UnixMilli()
			n.waitingSlots.PushSlot(subTask)
			if subTask.State == PublishingTask || subTask.State == PublishedTask || subTask.State == RunningTask {
				//任务已经分发处理，并不在全局队列中
				subTask.InitQueuePos = 0
				useNode := SingletonNodes().FindNode(subTask.SystemUUID)
				if useNode != nil {
					exist, waitCount := useNode.Match.GetTaskPos(subTask.SubID)
					if exist {
						subTask.InitWaitCount = int64(waitCount)
					} else {
						subTask.InitWaitCount = 0
					}
					subTask.CurWaitCount = subTask.InitWaitCount
					subTask.InLocalInitPos = subTask.CurWaitCount
					subTask.CurLocalPos = subTask.InLocalInitPos
					n.ProcessSort()
				}
			} else if subTask.State == WaitingTask {
				//任务在全局队列中
				if exist, pos := n.WaitQueuePos(subTask.SubID); exist {
					subTask.InitQueuePos = pos
				} else {
					subTask.InitQueuePos = n.WaitQueueSize()
				}
				subTask.CurQueuePos = subTask.InitQueuePos
				subTask.InLocalInitPos = subTask.MaxSystemSlotSize
				subTask.CurLocalPos = subTask.InLocalInitPos
				subTask.InitWaitCount = subTask.InitQueuePos + subTask.MaxSystemSlotSize
				subTask.CurWaitCount = subTask.InitWaitCount
				n.WaitSort()
			}
		} else {
			//任务在全局队列中
			subTask.PushQueueTime = time.Now().UnixMilli()
			n.waitingSlots.PushSlot(subTask)
			if exist, pos := n.WaitQueuePos(subTask.SubID); exist {
				subTask.InitQueuePos = pos
			} else {
				subTask.InitQueuePos = n.WaitQueueSize()
			}
			subTask.InLocalInitPos = subTask.MaxSystemSlotSize
			subTask.CurLocalPos = subTask.MaxSystemSlotSize
			subTask.InitWaitCount = subTask.InitQueuePos + subTask.MaxSystemSlotSize
			subTask.CurWaitCount = subTask.InitWaitCount
			n.WaitSort()
		}
	}
}

func (n *UserSessions) DeleteTask(taskID string) {
	n.taskLock.Lock()
	defer n.taskLock.Unlock()
	var newQueue []*SessionTask
	for _, task := range n.tasks {
		if !strings.EqualFold(task.TaskID, taskID) {
			newQueue = append(newQueue, task)
		} else {
			for _, subTask := range task.Tasks {
				n.waitingSlots.CleanSlot(subTask.SubID)
			}
		}
	}
	n.tasks = newQueue
}

func (n *UserSessions) GetSessionTasks(session string) []*SessionTask {
	n.taskLock.Lock()
	defer n.taskLock.Unlock()
	var sessionTasks []*SessionTask
	for _, task := range n.tasks {
		if strings.EqualFold(task.Session, session) {
			sessionTasks = append(sessionTasks, task)
		}
	}
	return sessionTasks
}

func (n *UserSessions) GetTasks() []*SessionTask {
	n.taskLock.Lock()
	defer n.taskLock.Unlock()
	var sessionTasks []*SessionTask
	sessionTasks = append(sessionTasks, n.tasks...)
	return sessionTasks
}

func (n *UserSessions) GetShowTasks() []ShowSessionTask {
	n.taskLock.Lock()
	defer n.taskLock.Unlock()
	var sessionTasks []ShowSessionTask
	for _, v := range n.tasks {
		tmpSessionTask := ShowSessionTask{
			TaskID:      v.TaskID,
			Session:     v.Session,
			State:       v.State,
			ImageNum:    v.ImageNum,
			PublishTime: v.PublishTime,
		}
		for _, subTask := range v.Tasks {
			showSubTask := ShowTaskSlot{
				SubID:          subTask.SubID,
				State:          subTask.State,
				SystemUUID:     subTask.SystemUUID,
				InitQueuePos:   subTask.InitQueuePos,
				CurQueuePos:    subTask.CurQueuePos,
				InLocalInitPos: subTask.InLocalInitPos,
				CurLocalPos:    subTask.CurLocalPos,
				InitWaitCount:  subTask.InitWaitCount,
				CurWaitCount:   subTask.CurWaitCount,
				PublishTime:    subTask.PublishTime,
				StartTime:      subTask.StartTime,
				EstimateMs:     subTask.EstimateMs,
				CompletedTime:  subTask.CompletedTime,
			}
			for _, ossUrl := range subTask.OSSUrls {
				showSubTask.OSSUrls = append(showSubTask.OSSUrls, ossUrl)
			}
			tmpSessionTask.ShowTasks = append(tmpSessionTask.ShowTasks, showSubTask)
		}
		sessionTasks = append(sessionTasks, tmpSessionTask)
	}
	return sessionTasks
}

func (n *UserSessions) GetSyncTasks(session string) []ShowSessionTask {
	n.taskLock.Lock()
	defer n.taskLock.Unlock()
	var sessionTasks []ShowSessionTask
	for _, v := range n.tasks {
		if strings.EqualFold(session, v.Session) {
			tmpSessionTask := ShowSessionTask{
				TaskID:      v.TaskID,
				Session:     v.Session,
				State:       v.State,
				ImageNum:    v.ImageNum,
				PublishTime: v.PublishTime,
			}
			for _, subTask := range v.Tasks {
				showSubTask := ShowTaskSlot{
					SubID:          subTask.SubID,
					State:          subTask.State,
					InitQueuePos:   subTask.InitQueuePos,
					CurQueuePos:    subTask.CurQueuePos,
					InLocalInitPos: subTask.InLocalInitPos,
					CurLocalPos:    subTask.CurLocalPos,
					InitWaitCount:  subTask.InitWaitCount,
					CurWaitCount:   subTask.CurWaitCount,
					PublishTime:    subTask.PublishTime,
					StartTime:      subTask.StartTime,
					EstimateMs:     subTask.EstimateMs,
					CompletedTime:  subTask.CompletedTime,
				}
				for _, ossUrl := range subTask.OSSUrls {
					showSubTask.OSSUrls = append(showSubTask.OSSUrls, ossUrl)
				}
				tmpSessionTask.ShowTasks = append(tmpSessionTask.ShowTasks, showSubTask)
			}
			sessionTasks = append(sessionTasks, tmpSessionTask)
		}
	}
	return sessionTasks
}

func (n *UserSessions) GetTask(session, taskID string) *SessionTask {
	n.taskLock.Lock()
	defer n.taskLock.Unlock()
	for _, task := range n.tasks {
		if strings.EqualFold(task.Session, session) && strings.EqualFold(task.TaskID, taskID) {
			return task
		}
	}
	return nil
}

func (n *UserSessions) checkCompletedTask(session, taskID string) bool {
	task := n.GetTask(session, taskID)
	if task != nil {
		for _, v := range task.Tasks {
			if (v.State != CompletedTask) && (v.State != FailedTask) {
				return false
			}
		}
		task.State = CompletedTask
		return true
	}
	return false
}

func (n *UserSessions) checkSuccessTask(session, taskID string) bool {
	task := n.GetTask(session, taskID)
	if task != nil {
		for _, v := range task.Tasks {
			if v.State != CompletedTask && v.State != FailedTask {
				return false
			}
		}
		return true
	}
	return false
}

func (n *UserSessions) checkSortTask(session, taskID string) {
	task := n.GetTask(session, taskID)
	if task != nil {
		n.DeleteTask(taskID)
	}
}

func (n *UserSessions) WaitSort() {
	if n.waitingSlots != nil {
		n.waitingSlots.Sort()
	}
}

func (n *UserSessions) ProcessSort() {
	if n.processingSlots != nil {
		n.processingSlots.Sort()
	}
}

func (n *UserSessions) GetSessionNum() int {
	return len(n.sessions)
}

// ------------------------------------------------------
func (n *UserSessions) OnConnect(linker *ws.Linker, param, Ip, port string) {
	funName := "OnConnect"
	log4plus.Info("%s User OnConnect param=[%s] Ip=[%s] Port=[%s]", funName, param, Ip, port)
	session := n.FindSession(param)
	if session != nil {
		session.Net = append(session.Net, linker)
		if session.Match != nil {
			session.Match.SetHandle(linker.Handle(), Ip, port)
			sendSyncs2(param, SessionSync)
		}
	}
}

func (n *UserSessions) OnRead(linker *ws.Linker, param string, message []byte) error {
	session := n.FindSession(param)
	if session != nil {
		session.Match.RecvMessageCh <- message
	}
	return nil
}

func (n *UserSessions) OnDisconnect(linker *ws.Linker, param, Ip, port string) {
	funName := "OnDisconnect"
	log4plus.Info("%s User OnDisconnect param=[%s] Ip=[%s] Port=[%s]", funName, param, Ip, port)

	n.DeleteSessionLinker(param, linker.Handle())
}

// 加载历史中没有完成的任务，重新执行
func (n *UserSessions) loadHistory() {
	funName := "loadHistory"
	now := time.Now().AddDate(0, 0, -1*configure.SingletonConfigure().AITask.AITaskRecheckDays)
	historyTasks := db.SingletonTaskDB().LoadHistoryTasks(now)
	if len(historyTasks) > 0 {
		for _, task := range historyTasks {
			task.SubNum = int64(len(task.SubTasks))
			SingletonUsers().NewTaskFromDB(task.Session, task)
			session := SingletonUsers().FindSession(task.Session)
			if session == nil {
				if err, _ := SingletonUsers().NewSession(task.Session, ""); err != nil {
					errString := fmt.Sprintf("%s BindJSON Failed err=[%s]", funName, err.Error())
					log4plus.Error(errString)
				}
			}
		}
	}
}

func SingletonUsers() *UserSessions {
	if gUserSessions == nil {
		gUserSessions = &UserSessions{
			sessions:        make(map[string]*UserSession),
			waitingSlots:    NewWaitingSlots(),
			processingSlots: NewProcessingSlots(),
		}
		gUserSessions.userWs = ws.NewWSServer(configure.SingletonConfigure().Net.User.RoutePath,
			configure.SingletonConfigure().Net.User.Listen,
			ws.UserMode,
			10000,
			gUserSessions)
		time.Sleep(time.Duration(5) * time.Second)
		gUserSessions.loadHistory()
		go gUserSessions.userWs.StartServer()
	}
	return gUserSessions
}
