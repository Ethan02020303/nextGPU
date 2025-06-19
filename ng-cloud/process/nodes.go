package process

import (
	"errors"
	"fmt"
	log4plus "github.com/nextGPU/include/log4go"
	"github.com/nextGPU/ng-cloud/configure"
	"github.com/nextGPU/ng-cloud/db"
	ws "github.com/nextGPU/ng-cloud/net/websocket"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	MeasureID   = string("00000000") //Measurement Tasks
	DispenseID  = string("00000001") //Workspace Task Distribution
	ConfigureID = string("00000002") //Configure
)

const (
	HeartTimeout = int64(30) //Measurement Tasks
)

type SlotStatus int // 自定义类型作为枚举的基础类型
const (
	StatusIdle SlotStatus = iota
	StatusRunning
)

type Slot struct {
	SystemUUID string
	SlotID     int
	TaskID     string
	SubID      string
	EstimateMs int64
	State      SlotStatus
}

type IdleSlots struct {
	lock  sync.Mutex
	slots []*Slot
}

func (n *IdleSlots) PushSlot(slot *Slot) {
	n.lock.Lock()
	slot.TaskID = ""
	slot.SubID = ""
	slot.State = StatusIdle
	n.slots = append(n.slots, slot)
	n.lock.Unlock()

	SingletonNodes().NodeSlotCh <- struct{}{}
}

func (n *IdleSlots) CleanSlot(systemUUID string) {
	n.lock.Lock()
	var newSlots []*Slot
	for _, slot := range n.slots {
		if !strings.EqualFold(slot.SystemUUID, systemUUID) {
			newSlots = append(newSlots, slot)
		}
	}
	n.slots = newSlots
	n.lock.Unlock()
}

func (n *IdleSlots) SlotSize() int {
	n.lock.Lock()
	defer n.lock.Unlock()
	return len(n.slots)
}

func (n *IdleSlots) PopSlot(systemUUID string, slotID int) *Slot {
	n.lock.Lock()
	defer n.lock.Unlock()

	for i, slot := range n.slots {
		if strings.EqualFold(systemUUID, slot.SystemUUID) && (slotID == slot.SlotID) {
			n.slots = append(n.slots[:i], n.slots[i+1:]...)
			return slot
		}
	}
	return nil
}

func (n *IdleSlots) PopAllWait() []Slot {
	n.lock.Lock()
	defer n.lock.Unlock()
	var waitSlots []Slot
	for _, v := range n.slots {
		waitSlots = append(waitSlots, Slot{
			SystemUUID: v.SystemUUID,
			SlotID:     v.SlotID,
			TaskID:     v.TaskID,
			SubID:      v.SubID,
			State:      v.State,
		})
	}
	return waitSlots
}

func NewIdleSlots() *IdleSlots {
	idleSlots := &IdleSlots{}
	return idleSlots
}

type RunningSlots struct {
	lock  sync.Mutex
	slots []*Slot
}

func (n *RunningSlots) PushSlot(slot *Slot, subId string) {
	n.lock.Lock()
	defer n.lock.Unlock()

	slot.SubID = subId
	slot.State = StatusRunning
	n.slots = append(n.slots, slot)
}

func (n *RunningSlots) PopSlot(systemUUID, subID string) *Slot {
	n.lock.Lock()
	defer n.lock.Unlock()

	for i, slot := range n.slots {
		if strings.EqualFold(slot.SubID, subID) && strings.EqualFold(slot.SystemUUID, systemUUID) {
			n.slots = append(n.slots[:i], n.slots[i+1:]...)
			return slot
		}
	}
	return nil
}

func (n *RunningSlots) CleanSlot(systemUUID string) {
	n.lock.Lock()
	defer n.lock.Unlock()

	var newSlots []*Slot
	for _, slot := range n.slots {
		if strings.EqualFold(systemUUID, slot.SystemUUID) {
			newSlots = append(newSlots, slot)
		}
	}
	n.slots = newSlots
}

func (n *RunningSlots) SlotSize() int {
	n.lock.Lock()
	defer n.lock.Unlock()
	return len(n.slots)
}

func (n *RunningSlots) SlotNum(systemUUID string) int {
	n.lock.Lock()
	defer n.lock.Unlock()

	size := 0
	for _, slot := range n.slots {
		if strings.EqualFold(slot.SystemUUID, systemUUID) {
			size++
		}
	}
	return size
}

func (n *RunningSlots) GetRunningSlots() []Slot {
	n.lock.Lock()
	defer n.lock.Unlock()
	var slots []Slot
	for _, slot := range n.slots {
		slots = append(slots, Slot{
			SystemUUID: slot.SystemUUID,
			SlotID:     slot.SlotID,
			TaskID:     slot.TaskID,
			SubID:      slot.SubID,
			EstimateMs: slot.EstimateMs,
			State:      slot.State,
		})
	}
	return slots
}

func NewRunningSlots() *RunningSlots {
	runningSlots := &RunningSlots{}
	return runningSlots
}

type Node struct {
	SystemUUID  string
	SlotSize    int
	State       db.NodeStatus
	MeasureTime time.Time
	Base        *NodeBase
	Match       *NodeMatch
	Net         *ws.Linker
	Slots       []*Slot
}

type NodeOnlineTime struct {
	SystemUUID   string
	NodeIp       string
	RegisterTime time.Time
	EnrollTime   time.Time
}

type Nodes struct {
	nodeWs         *ws.WSServer //websocket
	lock           sync.Mutex
	nodes          map[string]*Node //all nodes
	idleSlots      *IdleSlots       //idle queue
	runningSlots   *RunningSlots    //running queue
	InitiateTaskCh chan struct{}    //有任务插入头部通道
	NodeSlotCh     chan struct{}    //空闲节点槽通道
	TaskSlotCh     chan struct{}    //等待任务槽通道
	OfflineCh      chan string
	TimeoutCh      chan string
	ExporterDataCh chan MonitorData
}

var gNodes *Nodes

func (n *Nodes) AddNode(systemUUID string, clientIp string) (error, *Node) {
	n.lock.Lock()
	defer n.lock.Unlock()
	_, Ok := n.nodes[systemUUID]
	if Ok {
		return errors.New(""), nil
	}
	node := &Node{
		SystemUUID: systemUUID,
		State:      db.Enrolled,
	}
	node.State = db.Unavailable
	node.Base = NewBaseHeart(systemUUID, clientIp, node)
	node.Match = NewNodeMatch(node, systemUUID)
	n.nodes[node.SystemUUID] = node
	if node.SlotSize <= 0 {
		node.SlotSize = 1
	}
	for _, docker := range node.Base.Dockers {
		if docker.Deleted == db.UnDelete {
			for i := 1; i <= node.SlotSize; i++ {
				slot := &Slot{
					SystemUUID: systemUUID,
					SlotID:     i,
					State:      StatusIdle,
				}
				node.Slots = append(node.Slots, slot)
			}
		}
	}
	return nil, node
}

func (n *Nodes) offlineNode(systemUUID string, offline bool) {
	if node := n.FindNode(systemUUID); node != nil {
		if node.Net != nil {
			_ = node.Net.Close()
			node.Net = nil
		}
		node.State = db.Disconnect
		subTasks := node.Match.GetTasks()
		if len(subTasks) > 0 {
			for _, subTask := range subTasks {
				SingletonUsers().DeleteProcessing(subTask.SubID)
				SingletonUsers().PushWaitHead(subTask)
			}
		}
		if offline {
			node.Match.OfflineCh <- true
		} else {
			node.Match.CloseCh <- true
		}
		n.idleSlots.CleanSlot(systemUUID)
		n.runningSlots.CleanSlot(systemUUID)
	}
}

func (n *Nodes) DelNode(systemUUID string) {
	n.offlineNode(systemUUID, false)

	n.lock.Lock()
	delete(n.nodes, systemUUID)
	n.lock.Unlock()
}

func (n *Nodes) CleanIdleSlot(systemUUID string) {
	n.idleSlots.CleanSlot(systemUUID)
	n.runningSlots.CleanSlot(systemUUID)
}

func (n *Nodes) GetIdleSlotNum() int {
	return n.idleSlots.SlotSize()
}

func (n *Nodes) GetRunningSlotNum() int {
	return n.runningSlots.SlotSize()
}

func (n *Nodes) FindNode(systemUUID string) *Node {
	n.lock.Lock()
	defer n.lock.Unlock()
	node, Ok := n.nodes[systemUUID]
	if Ok {
		return node
	}
	return nil
}

func (n *Nodes) Exist(systemUUID string) bool {
	n.lock.Lock()
	defer n.lock.Unlock()
	_, Ok := n.nodes[systemUUID]
	return Ok
}

func (n *Nodes) GetNodes() []*Node {
	n.lock.Lock()
	defer n.lock.Unlock()
	var nodes []*Node
	for _, v := range n.nodes {
		nodes = append(nodes, v)
	}
	return nodes
}

func (n *Nodes) getAllNodes() []NodeOnlineTime {
	n.lock.Lock()
	defer n.lock.Unlock()
	var nodes []NodeOnlineTime
	for _, v := range n.nodes {
		nodes = append(nodes, NodeOnlineTime{
			SystemUUID:   v.SystemUUID,
			RegisterTime: v.Base.RegisterTime,
			EnrollTime:   v.Base.EnrollTime,
		})
	}
	return nodes
}

func (n *Nodes) GetWorkspaceIDs(systemUUID string) []string {
	n.lock.Lock()
	defer n.lock.Unlock()
	var workspaceIDs []string
	if node, Ok := n.nodes[systemUUID]; Ok {
		for _, docker := range node.Base.Dockers {
			workspaceIDs = append(workspaceIDs, docker.WorkSpaceID)
		}
	}
	return workspaceIDs
}

func (n *Nodes) GetGPUIndex(checkGPU string, gpus []string) int {
	for index, gpu := range gpus {
		rtxIndex := strings.Index(checkGPU, fmt.Sprintf(" %s ", gpu))
		if rtxIndex == -1 {
			continue
		}
		return index
	}
	return -1
}

func familyName(gpu string) (error, string) {
	funName := "familyName"
	splitAfterColon := strings.SplitN(gpu, ": ", 2)
	if len(splitAfterColon) < 2 {
		errString := fmt.Sprintf("%s SplitN Failed gpu=[%s] ", funName, gpu)
		log4plus.Error(errString)
		return errors.New(errString), ""
	}
	modelPart := splitAfterColon[1]
	splitBeforeUUID := strings.SplitN(modelPart, " (", 2)
	gpuModel := splitBeforeUUID[0]
	return nil, gpuModel
}

func (n *Nodes) GPUFamilyIDVideoRam(gpu string) (err error, familyID int64, videoRam int64, gpuWeight float64) {
	var gpuName string
	if err, gpuName = familyName(gpu); err != nil {
		return err, 0, 0, 0
	} else {
		return db.SingletonNodeBaseDB().GetFamilyVRam(gpuName)
	}
}

func (n *Nodes) EstimateWaitCount(title string) (waitCount int64) {
	//查询当前能够处理此title类型工作流的算力
	funName := "EstimateWaitCount"
	waitCount = 0
	for _, node := range n.nodes {
		exist := false
		for _, docker := range node.Base.Dockers {
			for _, workflow := range docker.Workflows {
				if strings.EqualFold(workflow.Title, title) {
					exist = true
					break
				}
			}
			if exist {
				break
			}
		}
		if exist {
			err, slotSize := db.SingletonNodeBaseDB().SlotSize(node.SystemUUID)
			if err != nil {
				errString := fmt.Sprintf("%s SlotSize Failed err=[%s] gpu=[%s]", funName, err.Error(), node.Base.GPU)
				log4plus.Error(errString)
				return 0
			}
			if int64(slotSize) >= waitCount {
				waitCount = int64(slotSize)
			}
		}
	}
	return waitCount
}

func (n *Nodes) GetSystemUUIDs(workspaceID string) []string {
	n.lock.Lock()
	defer n.lock.Unlock()
	var systemUUIDs []string
	for _, node := range n.nodes {
		for _, docker := range node.Base.Dockers {
			if strings.EqualFold(docker.WorkSpaceID, workspaceID) {
				systemUUIDs = append(systemUUIDs, node.SystemUUID)
				break
			}
		}
	}
	return systemUUIDs
}

func (n *Nodes) GetTitlesFromWorkSpaceID(systemUUID, workspaceID string) []string {
	n.lock.Lock()
	defer n.lock.Unlock()
	var workflowTitles []string
	if node, Ok := n.nodes[systemUUID]; Ok {
		for _, docker := range node.Base.Dockers {
			if strings.EqualFold(docker.WorkSpaceID, workspaceID) {
				for _, workflow := range docker.Workflows {
					workflowTitles = append(workflowTitles, workflow.Title)
				}
			}
		}
	}
	return workflowTitles
}

func (n *Nodes) ReloadWorkflow(systemUUID string) {
	n.lock.Lock()
	defer n.lock.Unlock()
	if node, Ok := n.nodes[systemUUID]; Ok {
		node.Base.LoadDockerWorkflow(node.SystemUUID)
	}
}

func (n *Nodes) GetPassNodes(title string) []*Node {
	n.lock.Lock()
	defer n.lock.Unlock()
	var nodes []*Node
	for _, v := range n.nodes {
		if v.State == db.MeasurementComplete {
			for _, docker := range v.Base.Dockers {
				isAdd := false
				for _, workflow := range docker.Workflows {
					if strings.EqualFold(workflow.Title, title) && (docker.State == db.Passed) {
						nodes = append(nodes, v)
						isAdd = true
						break
					}
				}
				if isAdd {
					break
				}
			}
		}
	}
	return nodes
}

func (n *Nodes) SetNodeStatus(systemUUID string, status db.NodeStatus) {
	if node := n.FindNode(systemUUID); node != nil {
		node.State = status
	}
}

func (n *Nodes) GetDockerStatus(systemUUID, workspaceID string) db.DockerStatus {
	if node := n.FindNode(systemUUID); node != nil {
		for _, docker := range node.Base.Dockers {
			if strings.EqualFold(docker.WorkSpaceID, workspaceID) {
				return docker.State
			}
		}
	}
	return db.UnPassed
}

func (n *Nodes) SetDockerStatus(systemUUID string, workspaceID string, status db.DockerStatus) {
	if node := n.FindNode(systemUUID); node != nil {
		if status == db.UnPassed {

		} else if status == db.Passed {

		}
		for _, docker := range node.Base.Dockers {
			if strings.EqualFold(docker.WorkSpaceID, workspaceID) {
				docker.State = status
				return
			}
		}
	}
}

func (n *Nodes) SetNodeHeart(systemUUID, ip, os, cpu, gpu, version, gpuDriver string, memory uint64) {
	if node := n.FindNode(systemUUID); node != nil {
		node.Base.NodeIp = ip
		node.Base.OS = os
		node.Base.CPU = cpu
		node.Base.GPU = gpu
		node.Base.Version = version
		node.Base.GpuDriver = gpuDriver
		node.Base.Memory = memory
		node.Base.HeartTime = time.Now()
	}
}

func (n *Nodes) SetNodeDockerStatus(systemUUID, workspaceID string, status db.DockerStatus) *Node {
	if node := n.FindNode(systemUUID); node != nil {
		for _, docker := range node.Base.Dockers {
			if strings.EqualFold(workspaceID, docker.WorkSpaceID) {
				docker.State = status
			}
		}
	}
	return nil
}

func (n *Nodes) DeleteDocker(systemUUID, workspaceID string) {
	if node := n.FindNode(systemUUID); node != nil {
		node.Base.DelDocker(workspaceID)
	}
}

func (n *Nodes) PushMeasureSlot(slot *Slot) {
	n.runningSlots.PushSlot(slot, slot.SubID)
}

func (n *Nodes) PushDispenseSlot(slot *Slot) {
	n.runningSlots.PushSlot(slot, slot.SubID)
}

func (n *Nodes) PushIdle(systemUUID string, subID string) {
	if slot := n.runningSlots.PopSlot(systemUUID, subID); slot != nil {
		n.idleSlots.PushSlot(slot)
	}
}

func (n *Nodes) ClearIdle(systemUUID string, subID string) {
	if slot := n.runningSlots.PopSlot(systemUUID, subID); slot != nil {
		n.idleSlots.CleanSlot(systemUUID)
	}
}

func (n *Nodes) PushRunning(waitSlot Slot) error {
	funName := "PushRunning"
	slot := n.idleSlots.PopSlot(waitSlot.SystemUUID, waitSlot.SlotID)
	if slot != nil {
		slot.TaskID = waitSlot.TaskID
		slot.SubID = waitSlot.SubID
		slot.EstimateMs = waitSlot.EstimateMs
		n.runningSlots.PushSlot(slot, slot.SubID)
		return nil
	}
	errString := fmt.Sprintf("%s PopSlot failed  slot not found systemUUID=[%s] slotID=[%d]", funName, waitSlot.SystemUUID, waitSlot.SlotID)
	log4plus.Error(errString)
	return errors.New(errString)
}

func (n *Nodes) DeleteTaskFromMatch(systemUUID string, subID string) {
	if node := n.FindNode(systemUUID); node != nil {
		node.Match.DelTask(subID)
	}
}

func (n *Nodes) ClearTaskFromMatch(systemUUID string) {
	if node := n.FindNode(systemUUID); node != nil {
		node.Match.ClearTasks()
	}
}

func (n *Nodes) SetMeasureCompletion(systemUUID string) {
	funName := "SetMeasureCompletion"
	node := n.FindNode(systemUUID)
	if node != nil {
		node.State = db.MeasurementComplete
		_ = db.SingletonNodeBaseDB().SetMeasureCompletion(systemUUID, node.State)
		err, slotSize := db.SingletonNodeBaseDB().SlotSize(systemUUID)
		if err != nil {
			errString := fmt.Sprintf("%s SlotSize Failed err=[%s] gpu=[%s]", funName, err.Error(), node.Base.GPU)
			log4plus.Error(errString)
			return
		}
		node.SlotSize = slotSize
		n.CleanIdleSlot(systemUUID)
		node.Match.ClearTasks()
		node.Slots = node.Slots[:0]
		for i := 1; i <= node.SlotSize; i++ {
			slot := &Slot{
				SystemUUID: systemUUID,
				SlotID:     i,
				State:      StatusIdle,
			}
			node.Slots = append(node.Slots, slot)
		}
		for _, v := range node.Slots {
			n.idleSlots.PushSlot(v)
		}
		return
	}
	errString := fmt.Sprintf("%s The corresponding computing node was not found systemUUID=[%s]", funName, systemUUID)
	log4plus.Error(errString)
}

func (n *Nodes) SetMeasureUnPassed(systemUUID string) {
	funName := "SetMeasureCompletion"
	node := n.FindNode(systemUUID)
	if node != nil {
		log4plus.Info("%s FindNode systemUUID=[%s]", funName, systemUUID)
		node.State = db.MeasurementComplete
		node.MeasureTime = time.Now()
		_ = db.SingletonNodeBaseDB().SetMeasureCompletion(systemUUID, node.State)
		n.CleanIdleSlot(systemUUID)
		node.Match.ClearTasks()
		node.Slots = node.Slots[:0]
	} else {
		errString := fmt.Sprintf("%s The corresponding computing node was not found systemUUID=[%s]", funName, systemUUID)
		log4plus.Error(errString)
	}
}

func (n *Nodes) fullyIdleNodes() []string {
	n.lock.Lock()
	defer n.lock.Unlock()

	var systemUUIDs []string
	for _, node := range n.nodes {
		if node.Match.GetTaskSize() == 0 {
			if node.State == db.MeasurementComplete ||
				node.State == db.Measuring ||
				node.State == db.MeasurementFailed ||
				node.State == db.Connected {
				systemUUIDs = append(systemUUIDs, node.SystemUUID)
			}
		}
	}
	return systemUUIDs
}

func (n *Nodes) unPassed(systemUUIDs []string) []string {
	now := time.Now().Unix()
	var waitNodes []string
	for _, systemUUID := range systemUUIDs {
		if node := n.FindNode(systemUUID); node != nil {
			for _, docker := range node.Base.Dockers {
				if (docker.State == db.UnPassed) && (now-node.MeasureTime.Unix() >= UnPassTimeout) {
					exist := false
					for _, waitNode := range waitNodes {
						if strings.EqualFold(systemUUID, waitNode) {
							exist = true
						}
					}
					if !exist {
						waitNodes = append(waitNodes, systemUUID)
					}
				}
			}
		}
	}
	return waitNodes
}

func (n *Nodes) unExpired(systemUUIDs []string) []string {
	now := time.Now().Unix()
	var waitNodes []string
	for _, systemUUID := range systemUUIDs {
		if node := n.FindNode(systemUUID); node != nil {
			for _, docker := range node.Base.Dockers {
				if (docker.State == db.StartupSuccess) && (now-node.MeasureTime.Unix() >= ExpiredTimeout) {
					exist := false
					for _, waitNode := range waitNodes {
						if strings.EqualFold(systemUUID, waitNode) {
							exist = true
						}
					}
					if !exist {
						waitNodes = append(waitNodes, systemUUID)
					}
				}
			}
		}
	}
	return waitNodes
}

func (n *Nodes) measureNodes() {
	frees := n.fullyIdleNodes()
	//没有任务节点
	if len(frees) == 0 {
		return
	}
	//测量未通过
	fails := n.unPassed(frees)
	if len(fails) > 0 {
		for _, systemUUID := range fails {
			if node := n.FindNode(systemUUID); node != nil {
				for _, docker := range node.Base.Dockers {
					if docker.Deleted == db.UnDelete {
						sendMeasure(systemUUID, docker.WorkSpaceID, docker)
					}
				}
			}
		}
		return
	}
	//测量超时
	expires := n.unExpired(frees)
	if len(expires) > 0 {
		for _, systemUUID := range expires {
			if node := n.FindNode(systemUUID); node != nil {
				for _, docker := range node.Base.Dockers {
					if docker.Deleted == db.UnDelete {
						sendMeasure(systemUUID, docker.WorkSpaceID, docker)
					}
				}
			}
		}
		return
	}
}

type NodeIdleSlot struct {
	SystemUUID string
	CurTaskNum int
	GPU        string
	Level      int64
	SlotID     int
	TaskID     string
	SubID      string
	EstimateMs int64
	State      SlotStatus
}

func (n *Nodes) nodeGPUSort(nodeIdleSlots []*NodeIdleSlot) {
	funName := "nodeGPUSort"
	//需要对等待槽按照gpu有限级别进行排序
	err, gpus, levels := db.SingletonNodeBaseDB().GetGPUPriority()
	if err != nil {
		errString := fmt.Sprintf("%s GetGPUPriority Failed err=[%s] ", funName, err.Error())
		log4plus.Error(errString)
		return
	}
	for _, nodeSlot := range nodeIdleSlots {
		node := SingletonNodes().FindNode(nodeSlot.SystemUUID)
		if node != nil {
			position := SingletonNodes().GetGPUIndex(node.Base.GPU, gpus)
			if position != -1 {
				nodeSlot.GPU = gpus[position]
				nodeSlot.Level = levels[position]
			}
		}
	}
}

func (n *Nodes) nodePrioritySort(nodeIdleSlots []*NodeIdleSlot) {
	funName := "nodePrioritySort"
	//需要对等待槽按照gpu有限级别进行排序
	err, gpus, levels := db.SingletonNodeBaseDB().GetGPUPriority()
	if err != nil {
		errString := fmt.Sprintf("%s GetGPUPriority Failed err=[%s] ", funName, err.Error())
		log4plus.Error(errString)
		return
	}
	for _, nodeSlot := range nodeIdleSlots {
		node := n.FindNode(nodeSlot.SystemUUID)
		if node != nil {
			position := n.GetGPUIndex(node.Base.GPU, gpus)
			if position != -1 {
				nodeSlot.GPU = gpus[position]
				nodeSlot.Level = levels[position]
			}
		}
	}
	sort.Slice(nodeIdleSlots, func(p, q int) bool { return nodeIdleSlots[p].Level < nodeIdleSlots[q].Level })
}

func (n *Nodes) isNeedSchedule(nodeIdleSlots []*NodeIdleSlot) bool {
	curSystemUUID := ""
	for _, v := range nodeIdleSlots {
		if curSystemUUID == "" {
			curSystemUUID = v.SystemUUID
		} else {
			if !strings.EqualFold(curSystemUUID, v.SystemUUID) {
				return true
			}
		}
	}
	return false
}

func (n *Nodes) isSameGPU(nodeIdleSlots []*NodeIdleSlot) bool {
	curGPU := ""
	for _, v := range nodeIdleSlots {
		if curGPU == "" {
			curGPU = v.GPU
		} else {
			if !strings.EqualFold(curGPU, v.GPU) {
				return false
			}
		}
	}
	return true
}

func (n *Nodes) nodeTaskNumSort(nodeIdleSlots []*NodeIdleSlot) {
	for _, v := range nodeIdleSlots {
		v.CurTaskNum = n.CurNodeTaskNum(v.SystemUUID)
	}
	sort.Slice(nodeIdleSlots, func(p, q int) bool { return nodeIdleSlots[p].CurTaskNum < nodeIdleSlots[q].CurTaskNum })
}

type NodeRunningEstimateMS struct {
	SystemUUID string
	EstimateMs int64
}

func (n *Nodes) slotEstimateMSSort(nodeIdleSlots []*NodeIdleSlot) {
	var nodeRunningEstimateMS []NodeRunningEstimateMS
	slots := n.runningSlots.GetRunningSlots()
	for _, slot := range slots {
		exist := false
		for _, v := range nodeRunningEstimateMS {
			if strings.EqualFold(v.SystemUUID, slot.SystemUUID) {
				v.EstimateMs = v.EstimateMs + slot.EstimateMs
				exist = true
				break
			}
		}
		if !exist {
			nodeRunningEstimateMS = append(nodeRunningEstimateMS, NodeRunningEstimateMS{
				SystemUUID: slot.SystemUUID,
				EstimateMs: slot.EstimateMs,
			})
		}
	}
	sort.Slice(nodeRunningEstimateMS, func(p, q int) bool { return nodeRunningEstimateMS[p].EstimateMs < nodeRunningEstimateMS[q].EstimateMs })
	orderMap := make(map[string]int)
	for i, s := range nodeRunningEstimateMS {
		orderMap[s.SystemUUID] = i
	}
	sort.Slice(nodeIdleSlots, func(i, j int) bool {
		// 获取元素在 orderMap 中的索引
		idxI, existsI := orderMap[nodeIdleSlots[i].SystemUUID]
		idxJ, existsJ := orderMap[nodeIdleSlots[j].SystemUUID]

		// 两个元素都不存在时，按原始顺序排
		if !existsI && !existsJ {
			return i < j
		}
		// 不存在的元素排在最后
		if !existsI {
			return false
		}
		if !existsJ {
			return true
		}
		// 比较索引大小
		return idxI < idxJ
	})
}

func (n *Nodes) slotSort(nodeIdleSlots []*NodeIdleSlot) []*NodeIdleSlot {
	//判断是否需要调度（如果只是一个算力节点则不用进行调度）
	need := n.isNeedSchedule(nodeIdleSlots)
	if !need {
		//只有一个算力节点无需调度
		return nodeIdleSlots
	}
	//是否是同类型的GPU
	n.nodeGPUSort(nodeIdleSlots)
	//是否是同类型算力节点
	sameGPU := n.isSameGPU(nodeIdleSlots)
	if sameGPU {
		//相同类型GPU
		n.nodeTaskNumSort(nodeIdleSlots)
		return nodeIdleSlots
	} else {
		//不同类型的GPU，根据正在执行任务的预估时间进行排序
		runningSlots := n.runningSlots.GetRunningSlots()
		if len(runningSlots) <= 0 {
			//根据优先级排序
			n.nodePrioritySort(nodeIdleSlots)
			return nodeIdleSlots
		} else {
			//按照预估消耗时间排序
			n.slotEstimateMSSort(nodeIdleSlots)
			return nodeIdleSlots
		}
	}
}

// 使用算力节点任务槽查找
func (n *Nodes) TaskDispense() {
	funName := "TaskDispense"
	//得到节点任务槽
	waitSlots := n.idleSlots.PopAllWait()
	//没有任务槽
	if len(waitSlots) <= 0 {
		return
	}

	log4plus.Info(fmt.Sprintf("%s len(waitSlots)=[%d] ", funName, len(waitSlots)))
	if len(waitSlots) <= 1 {
		//只有一个任务槽
		waitSlot := waitSlots[0]
		//得到可以处理的任务（指定服务器的任务优先，然后才是自由分配的任务）
		subTask := SingletonUsers().PopSlotFromSystemUUID(waitSlot.SystemUUID)
		if subTask == nil {
			//随机获取一个任务(剔除出错的算力节点)
			workspaceIDs := SingletonNodes().GetWorkspaceIDs(waitSlot.SystemUUID)
			subTask = SingletonUsers().PopSlot(waitSlot.SystemUUID, workspaceIDs)
			if subTask == nil {
				return
			}
		}
		waitSlot.TaskID = subTask.TaskID
		waitSlot.SubID = subTask.SubID
		waitSlot.EstimateMs = subTask.EstimateMs
		node := n.FindNode(waitSlot.SystemUUID)
		if node != nil {
			//insert queue
			if err := n.PushRunning(waitSlot); err != nil {
				return
			}
			SingletonUsers().PushProcessing(subTask)
			node.Match.AddTask(subTask)
			subTask.InLocalTime = time.Now().UnixMilli()
			subTask.SystemUUID = waitSlot.SystemUUID
			subTask.State = PublishingTask
		}
		return
	}
	//多任务槽调度
	var nodeIdleSlots []*NodeIdleSlot
	for _, waitSlot := range waitSlots {
		nodeIdleSlot := &NodeIdleSlot{
			SystemUUID: waitSlot.SystemUUID,
			SlotID:     waitSlot.SlotID,
			TaskID:     waitSlot.TaskID,
			SubID:      waitSlot.SubID,
			EstimateMs: waitSlot.EstimateMs,
			State:      waitSlot.State,
		}
		nodeIdleSlots = append(nodeIdleSlots, nodeIdleSlot)
	}
	nodeIdleSlots = n.slotSort(nodeIdleSlots)
	log4plus.Info(fmt.Sprintf("%s len(nodeIdleSlots)=[%d] ", funName, len(nodeIdleSlots)))

	for _, v := range nodeIdleSlots {
		//得到可以处理的任务（指定服务器的任务优先，然后才是自由分配的任务）
		subTask := SingletonUsers().PopSlotFromSystemUUID(v.SystemUUID)
		if subTask == nil {
			//随机获取一个任务(剔除出错的算力节点)
			workspaceIDs := SingletonNodes().GetWorkspaceIDs(v.SystemUUID)
			log4plus.Info(fmt.Sprintf("%s SingletonNodes().GetWorkspaceIDs SystemUUID=[%s] workspaceIDs=[%v]", funName, v.SystemUUID, workspaceIDs))

			subTask = SingletonUsers().PopSlot(v.SystemUUID, workspaceIDs)
			if subTask == nil {
				return
			}
		}
		log4plus.Info(fmt.Sprintf("%s SystemUUID=[%s] ---->>>> SubID=[%v]", funName, v.SystemUUID, subTask.SubID))

		waitSlot := Slot{
			SystemUUID: v.SystemUUID,
			SlotID:     v.SlotID,
			TaskID:     v.TaskID,
			SubID:      v.SubID,
			EstimateMs: v.EstimateMs,
			State:      v.State,
		}
		waitSlot.TaskID = subTask.TaskID
		waitSlot.SubID = subTask.SubID
		node := n.FindNode(waitSlot.SystemUUID)
		if node != nil {
			//insert queue
			if err := n.PushRunning(waitSlot); err != nil {
				return
			}
			SingletonUsers().PushProcessing(subTask)
			node.Match.AddTask(subTask)
			subTask.InLocalTime = time.Now().UnixMilli()
			subTask.SystemUUID = waitSlot.SystemUUID
			subTask.State = PublishingTask
		}
	}
}

func (n *Nodes) dispenseStart() {
	nodes := n.GetNodes()
	for _, node := range nodes {
		n.Workspace2Node(node.SystemUUID)
	}
}

func (n *Nodes) Workspace2Node(systemUUID string) {
	funName := "Workspace2Node"
	curNode := n.FindNode(systemUUID)
	if curNode == nil {
		return
	}
	workSpaces := SingletonWorkSpaces().AllWorkSpaces()
	for _, workspace := range workSpaces {
		if workspace.Base.Deleted == db.UnDelete {
			//判读这个工作空间是否已经下发到达数量上限
			nodes := n.GetNodes()
			curOnlineCount := int64(0)
			for _, node := range nodes {
				for _, docker := range node.Base.Dockers {
					if strings.EqualFold(docker.WorkSpaceID, workspace.Base.WorkspaceID) {
						curOnlineCount++
					}
				}
			}
			requestCount := workspace.Base.DeployCount + workspace.Base.RedundantCount
			if curOnlineCount >= workspace.Base.DeployCount+workspace.Base.RedundantCount {
				log4plus.Info("%s workspaceID=[%s] requestCount=[%d] curOnlineCount=[%d]",
					funName, workspace.Base.WorkspaceID, requestCount, curOnlineCount)
				//已经下发满员了
				continue
			}
			//判断算力节点是否已经部署过这个工作空间
			exist := false
			for _, docker := range curNode.Base.Dockers {
				if strings.EqualFold(docker.WorkSpaceID, workspace.Base.WorkspaceID) {
					exist = true
					break
				}
			}
			if !exist {
				//检测最小GPU显存限制
				log4plus.Info("%s workspaceID=[%s] --->>> systemUUID=[%s]", funName, workspace.Base.WorkspaceID, curNode.SystemUUID)
				if err, _, videoRam, _ := SingletonNodes().GPUFamilyIDVideoRam(curNode.Base.GPU); err != nil {
					errString := fmt.Sprintf("%s GPUFamilyID Failed GPU=[%s] ", funName, curNode.Base.GPU)
					log4plus.Error(errString)
					continue
				} else {
					if videoRam >= workspace.Base.MinGB {
						sendDispenseStart(workspace, curNode)
						return
					}
				}
			}
		}
	}
}

func (n *Nodes) checkTimeout() {
	nodes := n.GetNodes()
	for _, node := range nodes {
		if node.State == db.Disconnect {
			if time.Now().Unix()-node.Base.HeartTime.Unix() >= HeartTimeout {
				n.TimeoutCh <- node.SystemUUID
			}
		}
	}
}

func (n *Nodes) CurNodeTaskNum(systemUUID string) int {
	return n.runningSlots.SlotNum(systemUUID)
}

func calcTimeDiff(t1, t2 time.Time) int {
	if t1.Hour() == t2.Hour() && t1.Day() == t2.Day() {
		return int(t1.Sub(t2).Seconds())
	}
	truncated := t1.Truncate(time.Hour)
	return int(t1.Sub(truncated).Seconds())
}

func calculateNextTrigger(now time.Time) time.Time {
	return now.Truncate(time.Hour).Add(time.Hour)
}

func (n *Nodes) nodeOnlineTime() {
	funName := "nodeOnlineTime"
	nodes := n.getAllNodes()
	now := time.Now()
	day := now.Format("2006-01-02")
	hour := now.Add(-1 * time.Hour).Hour()
	for _, node := range nodes {
		onlineTime := calcTimeDiff(now.Add(-1*time.Second), node.EnrollTime) + 1
		log4plus.Info("---->>>>>>>>>>>>>>>>%s now=[%s] enrollTime=[%s] onlineTime=[%d]",
			funName, now.Format("2006-01-02 15:04:05"), node.EnrollTime.Format("2006-01-02 15:04:05"), onlineTime)
		_ = db.SingletonNodeBaseDB().SetOnlineTime(node.SystemUUID, day, hour, onlineTime)
	}
}

func (n *Nodes) nodeOfflineTime(systemUUID string) {
	funName := "nodeOfflineTime"
	node := n.FindNode(systemUUID)
	if node != nil {
		now := time.Now()
		day := now.Format("2006-01-02")
		hour := now.Hour()
		onlineTime := calcTimeDiff(now, node.Base.EnrollTime)
		log4plus.Info("---->>>>>>>>>>>>>>>>%s now=[%s] enrollTime=[%s] onlineTime=[%d]",
			funName, now.Format("2006-01-02 15:04:05"), node.Base.EnrollTime.Format("2006-01-02 15:04:05"), onlineTime)
		_ = db.SingletonNodeBaseDB().SetOnlineTime(node.SystemUUID, day, hour, onlineTime)
	}
}

func (n *Nodes) check() {
	measureTicker := time.NewTicker(30 * time.Second)
	defer measureTicker.Stop()

	dispenseTicker := time.NewTicker(30 * time.Second)
	defer dispenseTicker.Stop()

	disconnectTicker := time.NewTicker(30 * time.Second)
	defer disconnectTicker.Stop()

	nextTrigger := calculateNextTrigger(time.Now())
	onlineTimeTicker := time.NewTimer(time.Until(nextTrigger))
	defer onlineTimeTicker.Stop()

	for {
		select {
		case <-measureTicker.C: //测量
			n.measureNodes()
		case <-disconnectTicker.C: //断开链接
			n.checkTimeout()
		case <-dispenseTicker.C: //分发工作空间
			n.dispenseStart()
		case <-onlineTimeTicker.C: //记录在线算力端在线时长
			n.nodeOnlineTime()
			nextTrigger = calculateNextTrigger(time.Now())
			onlineTimeTicker.Reset(time.Until(nextTrigger))
		case <-n.InitiateTaskCh: //有新任务
			n.TaskDispense()
		case <-n.NodeSlotCh: //有节点槽空闲
			n.TaskDispense()
		case <-n.TaskSlotCh: //有新的算力节点空闲
			n.TaskDispense()
		case systemUUID := <-n.OfflineCh: //链接断开
			n.nodeOfflineTime(systemUUID)
			n.DelNode(systemUUID)
		case systemUUID := <-n.TimeoutCh:
			n.DelNode(systemUUID)
		}
	}
}

// OnEvent
func (n *Nodes) OnConnect(linker *ws.Linker, param string, Ip string, port string) {
	node := n.FindNode(param)
	if node != nil {
		node.State = db.Connected
		node.Net = linker
		node.Match.SetHandle(node.Net.Handle(), Ip, port)
		if node.State == db.Unavailable { //first connect
			if node.SlotSize > 0 {
				for _, v := range node.Slots {
					n.idleSlots.PushSlot(v)
				}
			}
		}
	}
}

func (n *Nodes) OnRead(linker *ws.Linker, param string, message []byte) error {
	node := n.FindNode(param)
	if node != nil {
		node.Match.RecvMessageCh <- message
	}
	return nil
}

func (n *Nodes) OnDisconnect(linker *ws.Linker, param string, Ip string, port string) {
	n.OfflineCh <- param
}

func SingletonNodes() *Nodes {
	if gNodes == nil {
		gNodes = &Nodes{
			nodes:          make(map[string]*Node),
			idleSlots:      NewIdleSlots(),
			runningSlots:   NewRunningSlots(),
			InitiateTaskCh: make(chan struct{}, 1024),
			NodeSlotCh:     make(chan struct{}, 1024),
			TaskSlotCh:     make(chan struct{}, 1024),
			OfflineCh:      make(chan string, 1024),
			TimeoutCh:      make(chan string, 1024),
			ExporterDataCh: make(chan MonitorData, 1024),
		}
		_ = SingletonWorkSpaces()
		_ = SingletonConsumptions()

		go gNodes.check()
		if gNodes.nodeWs = ws.NewWSServer(configure.SingletonConfigure().Net.Node.RoutePath,
			configure.SingletonConfigure().Net.Node.Listen,
			ws.NodeMode, 1000, gNodes); gNodes.nodeWs != nil {
			go gNodes.nodeWs.StartServer()
		}
	}
	return gNodes
}
