package process

import (
	"encoding/json"
	"fmt"
	"github.com/jinzhu/copier"
	log4plus "github.com/nextGPU/include/log4go"
	"github.com/nextGPU/ng-cloud/db"
	"strings"
	"sync"
	"time"
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

type (
	// Basic Message Structure
	BaseMessage struct {
		MessageID int64 `json:"messageID"`
	}
	Layer struct {
		LayerId  string `json:"layerId"`
		Progress string `json:"progress"`
	}
	MeasureTaskSlot struct {
		WorkSpaceID   string
		Title         string
		TaskID        string
		SubID         string
		Index         int64
		EstimatesTime int64
		PublishTime   time.Time
		StartTime     time.Time
		CompletedTime time.Time
		Duration      int64
		State         TaskState
		SystemUUID    string
		TaskData      TaskRequestData
	}
	DockerArray struct {
		WorkSpaceID          string               `json:"workSpaceID"`
		RunningMode          db.RunMode           `json:"runMode"`
		State                db.DockerStatus      `json:"state"`
		MajorCMD             string               `json:"majorCMD"`
		MinorCMD             string               `json:"minorCMD"`
		ComfyUIHttp          string               `json:"comfyUIHttp"`
		ComfyUIWS            string               `json:"comfyUIWS"`
		SaveDir              string               `json:"saveDir"`
		GPUMem               string               `json:"gpuMem"`
		Deleted              db.DockerDeleted     `json:"deleted"`
		MeasureTitle         string               `json:"measureTitle"`
		Measure              string               `json:"measure"`
		MeasureConfiguration string               `json:"measureConfiguration"`
		Workflows            []*db.DockerWorkflow `json:"workflows"`
		MeasureTaskID        string
		MeasureSubTasks      []*MeasureTaskSlot
	}
)

type NodeMatch struct {
	node          *Node
	systemUUID    string
	selfHandle    uint64
	wsIP          string
	wsPort        string
	taskLock      sync.Mutex
	tasks         []*TaskSlot
	OfflineCh     chan bool
	CloseCh       chan bool
	RecvMessageCh chan []byte
	SendMessageCh chan []byte
}

func parseBase(data []byte) (BaseMessage, error) {
	var base BaseMessage
	err := json.Unmarshal(data, &base)
	return base, err
}

func (f *NodeMatch) SelfHandle() uint64 {
	return f.selfHandle
}

func (f *NodeMatch) SetHandle(selfHandle uint64, ip string, port string) {
	f.selfHandle = selfHandle
	f.wsIP = ip
	f.wsPort = port
}

func (f *NodeMatch) AddTask(subTask *TaskSlot) {
	funName := "AddTask"
	msg := struct {
		MessageID   int64           `json:"messageID"`
		SubID       string          `json:"subID"`
		WorkSpaceID string          `json:"workSpaceID"`
		Session     string          `json:"session"`
		ImagePath   string          `json:"imagePath"`
		Data        json.RawMessage `json:"data"`
	}{
		MessageID:   TaskPublish,
		SubID:       subTask.SubID,
		WorkSpaceID: subTask.WorkSpaceID,
		Session:     subTask.Session,
		ImagePath:   subTask.ImagePath,
	}
	data, _ := json.Marshal(subTask.TaskData)
	_ = copier.Copy(&msg.Data, &data)
	message, err := json.Marshal(msg)
	if err != nil {
		log4plus.Error("%s [%s]---->>>>[%s]", funName, msg.SubID, f.node.SystemUUID)
		return
	}
	f.SendMessageCh <- message

	f.taskLock.Lock()
	defer f.taskLock.Unlock()

	log4plus.Info("%s subID=[%s] Match >>>>", funName, subTask.SubID)
	f.tasks = append(f.tasks, subTask)
}

func (f *NodeMatch) DelTask(subID string) {
	f.taskLock.Lock()
	defer f.taskLock.Unlock()
	var tasks []*TaskSlot
	for _, task := range f.tasks {
		if !strings.EqualFold(task.SubID, subID) {
			tasks = append(tasks, task)
		}
	}
	f.tasks = tasks
}

func (f *NodeMatch) GetTasks() []*TaskSlot {
	f.taskLock.Lock()
	defer f.taskLock.Unlock()
	var tasks []*TaskSlot
	tasks = append(tasks, f.tasks...)
	return tasks
}

func (f *NodeMatch) ClearTasks() {
	f.taskLock.Lock()
	defer f.taskLock.Unlock()
	f.tasks = f.tasks[:0]
}

func (f *NodeMatch) GetTaskSize() int {
	return len(f.tasks)
}

func (f *NodeMatch) GetTaskPos(subID string) (bool, int) {
	f.taskLock.Lock()
	defer f.taskLock.Unlock()
	for index, task := range f.tasks {
		if strings.EqualFold(task.SubID, subID) {
			return true, index
		}
	}
	return false, 0
}

func (f *NodeMatch) offline() {
	f.selfHandle = 0
	f.wsIP = ""
	f.wsPort = ""

	f.taskLock.Lock()
	defer f.taskLock.Unlock()
	f.tasks = f.tasks[:0]
}

func (f *NodeMatch) IpPort() (string, string) {
	return f.wsIP, f.wsPort
}

func (f *NodeMatch) transfer() {
	for {
		select {
		case msg := <-f.RecvMessageCh:
			f.dispense(msg)
		case task := <-f.SendMessageCh:
			f.sendMessage(task)
		case <-f.OfflineCh:
			f.offline()
		case <-f.CloseCh:
			f.offline()
			return
		}
	}
}

// send message
func (f *NodeMatch) sendMessage(message []byte) {
	if f.node.Net != nil {
		_ = f.node.Net.WriteMessage(message)
	}
}

func (f *NodeMatch) dispense(message []byte) {
	funName := "dispense"
	if baseMessage, err := parseBase(message); err != nil {
		errString := fmt.Sprintf("%s parseBase Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
	} else {
		switch baseMessage.MessageID {
		//ai task
		case TaskPublish:
			taskPublish(message)
		case TaskStart:
			taskStart(message)
		case TaskInferenceCompleted:
			taskInferenceCompleted(message)
		case TaskCompleted:
			taskCompleted(f.systemUUID, message)
		case TaskFailed:
			taskFailed(f.systemUUID, message)
		//workspace
		case DispenseStart:
			dispenseStart(f.systemUUID, message)
		case DispenseProgress:
			dispenseProgress(f.systemUUID, message)
		case DispenseComplete:
			dispenseComplete(f.systemUUID, message)
		case StartupWorkSpace:
			startupWorkSpace(f.systemUUID, message)
		case StartupComplete:
			startupComplete(f.systemUUID, message)
		case DelWorkSpace:
			deleteDocker(f.systemUUID, message)
		case PullFail:
			pullFail(f.systemUUID, message)
		case StartupFail:
			startupFail(f.systemUUID, message)
		case DeleteContainer:
			deleteContainer(f.systemUUID, message)
		//measure
		case MeasurePublish:
			measurePublish(f.systemUUID, message)
		case MeasureStart:
			measureStart(f.systemUUID, message)
		case MeasureComplete:
			measureComplete(f.systemUUID, message)
		case MeasureFailed:
			measureFailed(f.systemUUID, message)
		//configure
		case OSSConfigure:
			sendOSSConfigure(f.systemUUID)
		case MonitorConfigure:
			sendMonitorConfigure(f.systemUUID)
		case NodePing:
			nodePing(f.systemUUID, f.wsIP, message)
		case ComfyUIStartupSuccess:
			comfyUIStartupSuccess(f.systemUUID, message)
		case NodeExporterData:
			nodeExporterData(f.systemUUID, message)
		}
	}
}

func NewNodeMatch(node *Node, systemUUID string) *NodeMatch {
	match := &NodeMatch{
		node:          node,
		systemUUID:    systemUUID,
		selfHandle:    0,
		wsIP:          "",
		wsPort:        "",
		CloseCh:       make(chan bool),
		OfflineCh:     make(chan bool),
		SendMessageCh: make(chan []byte, 1024),
		RecvMessageCh: make(chan []byte, 1024),
	}
	go match.transfer()
	return match
}
