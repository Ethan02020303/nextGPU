package process

import (
	"fmt"
	log4plus "github.com/nextGPU/include/log4go"
	"github.com/nextGPU/ng-cloud/db"
	"strings"
	"time"
)

const (
	// 测量超时时长
	ExpiredTimeout = int64(24 * 60 * 60)
	UnPassTimeout  = int64(10 * 60)
)

type NodeBase struct {
	systemUUID   string
	node         *Node
	RegisterTime time.Time
	EnrollTime   time.Time
	Versions     string
	NodeIp       string
	HeartTime    time.Time
	OS           string
	CPU          string
	Memory       uint64
	GPU          string
	Version      string
	GpuDriver    string
	Dockers      []*DockerArray
}

func (n *NodeBase) loadNodeBase(systemUUID string) error {
	funName := "loadNodeBase"
	err, nodeBaseDB := db.SingletonNodeBaseDB().LoadNode(systemUUID)
	if err == nil {
		//base
		registerTime, errRegister := time.Parse("2006-01-02 15:04:05", nodeBaseDB.RegisterTime)
		if errRegister != nil {
			n.RegisterTime, _ = time.Parse("2006-01-02 15:04:05", "2006-01-02 15:04:05")
		} else {
			n.RegisterTime = registerTime
		}
		n.EnrollTime = time.Now()
		n.Versions = nodeBaseDB.Versions
		errResult, slotSize := db.SingletonNodeBaseDB().SlotSize(systemUUID)
		if errResult != nil {
			n.node.SlotSize = 1
		} else {
			n.node.SlotSize = slotSize
		}
		//heart
		n.OS = nodeBaseDB.OS
		n.CPU = nodeBaseDB.CPU
		n.GPU = nodeBaseDB.GPU
		n.GpuDriver = nodeBaseDB.GpuDriver
		n.Memory = nodeBaseDB.Memory
		//measure
		if nodeBaseDB.MeasureTime != "" {
			measureTime, errMeasure := time.Parse("2006-01-02 15:04:05", nodeBaseDB.MeasureTime)
			if errMeasure != nil {
				n.node.MeasureTime, _ = time.Parse("2006-01-02 15:04:05", "2006-01-02 15:04:05")
			} else {
				n.node.MeasureTime = measureTime
			}
		}
		return nil
	}
	log4plus.Error(fmt.Sprintf("%s db.SingletonNodeBaseDB().LoadNode Failed systemUUID=[%s] err=[%s]", funName, systemUUID, err.Error()))
	return err
}

func (n *NodeBase) LoadDocker(systemUUID string) {
	funName := "LoadDocker"
	err, dockers := db.SingletonNodeBaseDB().NodeDockers(systemUUID)
	if err != nil {
		errString := fmt.Sprintf("%s NodeDockers Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		n.Dockers = nil
	} else {
		n.Dockers = n.Dockers[:0]
		for _, v := range dockers {
			docker := &DockerArray{
				WorkSpaceID:          v.WorkSpaceID,
				RunningMode:          v.RunningMode,
				State:                v.State,
				MajorCMD:             v.MajorCMD,
				MinorCMD:             v.MinorCMD,
				ComfyUIHttp:          v.ComfyUIHttp,
				ComfyUIWS:            v.ComfyUIWS,
				SaveDir:              v.SaveDir,
				Deleted:              v.Delete,
				MeasureTitle:         v.MeasureTitle,
				Measure:              v.Measure,
				MeasureConfiguration: v.MeasureConfiguration,
			}
			if errWorkflow, workflows := db.SingletonNodeBaseDB().WorkSpace2Title(docker.WorkSpaceID); errWorkflow == nil {
				docker.Workflows = append(docker.Workflows, workflows...)
			}
			n.Dockers = append(n.Dockers, docker)
		}
	}
}

func (n *NodeBase) LoadDockerWorkflow(systemUUID string) {
	for _, docker := range n.Dockers {
		docker.Workflows = docker.Workflows[:0]
		if errWorkflow, workflows := db.SingletonNodeBaseDB().WorkSpace2Title(docker.WorkSpaceID); errWorkflow == nil {
			docker.Workflows = append(docker.Workflows, workflows...)
		}
	}
}

func (n *NodeBase) DelDocker(workSpaceID string) {
	for i, v := range n.Dockers {
		if strings.EqualFold(v.WorkSpaceID, workSpaceID) {
			n.Dockers = append(n.Dockers[:i], n.Dockers[i+1:]...)
			return
		}
	}
}

func (n *NodeBase) SetHeart(systemUUID, nodeIp, OS, CPU, GPU, Version, GpuDriver string, memory uint64) error {
	funName := "SetHeart"
	n.HeartTime = time.Now()
	n.NodeIp = nodeIp
	n.OS = OS
	n.CPU = CPU
	n.Memory = memory
	n.GPU = GPU
	n.Version = Version
	n.GpuDriver = GpuDriver
	if err := db.SingletonNodeBaseDB().NodeHeart(systemUUID, nodeIp, OS, CPU, GPU, GpuDriver, Version, memory); err != nil {
		log4plus.Error(fmt.Sprintf("%s NodeHeart Failed err=[%s]", funName, err.Error()))
		return err
	}
	return nil
}

func NewBaseHeart(systemUUID string, nodeIp string, node *Node) *NodeBase {
	base := &NodeBase{
		systemUUID: systemUUID,
		node:       node,
	}
	base.NodeIp = nodeIp
	base.EnrollTime = time.Now()
	if err := base.loadNodeBase(systemUUID); err != nil {
		return nil
	}
	base.LoadDocker(systemUUID)
	return base
}
