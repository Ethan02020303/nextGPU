package header

import (
	"encoding/json"
)

const (
	Base           = iota + 300
	JsonParseError = Base + 1
)

type RunMode int

const (
	DockerRun RunMode = iota
	ComposeMode
)

type DockerDeleted int

const (
	unDelete DockerDeleted = iota
	Deleted
	DockerDeleteMax
)

type DockerStatus int

const (
	UnInstalled DockerStatus = iota
	Pulling
	PullFailed
	PullCompleted
	Startuping
	StartupFailed
	StartupCompleted
	StartupSuccess
	UnPassed
	DockerStatusMax
)

type NodeStatus int

const (
	Unavailable NodeStatus = iota
	Disconnect
	Enrolling
	Enrolled
	WebSocketConnecting
	WebSocketConnected
	Measuring
	MeasurementFailed
	MeasurementComplete
)

type (
	// Basic Message Structure
	BaseMessage struct {
		MessageID int64 `json:"messageID"`
	}
	PublishMessage struct {
		MessageID   int64           `json:"messageID"`
		SubID       string          `json:"subID"`
		WorkSpaceID string          `json:"workSpaceID"`
		Session     string          `json:"session"`
		ImagePath   string          `json:"imagePath"`
		Data        json.RawMessage `json:"data"`
	}
	MeasureMessage struct {
		MessageID   int64           `json:"messageID"`
		TaskID      string          `json:"taskID"`
		SubID       string          `json:"subID"`
		WorkSpaceID string          `json:"workSpaceID"`
		Data        json.RawMessage `json:"data"`
	}
	DispenseMessage struct {
		MessageID   int64         `json:"messageID"`
		SubID       string        `json:"subID"`
		WorkSpaceID string        `json:"workSpaceID"`
		RunningMode RunMode       `json:"runMode"`
		State       DockerStatus  `json:"state"`
		MajorCMD    string        `json:"majorCMD"`
		MinorCMD    string        `json:"minorCMD"`
		ComfyUIHttp string        `json:"comfyUIHttp"`
		ComfyUIWS   string        `json:"comfyUIWS"`
		SaveDir     string        `json:"saveDir"`
		Deleted     DockerDeleted `json:"deleted"`
	}
	StartupWorkSpaceMessage struct {
		MessageID   int64  `json:"messageID"`
		WorkSpaceID string `json:"workSpaceID"`
	}
	RequestData struct {
		Title string          `json:"title"`
		Data  json.RawMessage `json:"data"`
	}
	NodeHeart struct {
		HeartTime   int64  `json:"heartTime"`
		NodeIp      string `json:"nodeIp"`
		OS          string `json:"os"`
		Board       string `json:"board"`
		CPU         string `json:"cpu"`
		Memory      uint64 `json:"memory"`
		GPU         string `json:"gpu"`
		Version     string `json:"version"`
		GpuDriver   string `json:"gpuDriver"`
		MeasureID   string `json:"measureID"`
		MeasureTime int64  `json:"measureTime"`
	}
	OSS struct {
		Endpoint        string `json:"endpoint"`
		AccessKeyID     string `json:"accessKeyID"`
		AccessKeySecret string `json:"accessKeySecret"`
		BucketName      string `json:"bucketName"`
	}
)

type (
	DockerArray struct {
		WorkSpaceID string        `json:"workSpaceID"`
		RunningMode RunMode       `json:"runMode"`
		State       DockerStatus  `json:"state"`
		MajorCMD    string        `json:"majorCMD"`
		MinorCMD    string        `json:"minorCMD"`
		ComfyUIHttp string        `json:"comfyUIHttp"`
		ComfyUIWS   string        `json:"comfyUIWS"`
		SaveDir     string        `json:"saveDir"`
		GPUMem      string        `json:"gpuMem"`
		Deleted     DockerDeleted `json:"deleted"`
	}
	NodeEnrollResponse struct {
		CodeId  int64         `json:"codeId"`
		Msg     string        `json:"msg"`
		Connect string        `json:"connect"`
		Dockers []DockerArray `json:"dockers"`
	}
	NodeValidate struct {
		CodeId       int64  `json:"codeId"`
		Msg          string `json:"msg"`
		AllowInstall bool   `json:"allowInstall"`
	}
	QueueTask struct {
		SubID       string `json:"subID"`
		InLocalPos  int64  `json:"inLocalPos"`
		InLocalTime int64  `json:"inLocalTime"`
		CurQueuePos int64  `json:"curQueuePos"`
		StartTime   int64  `json:"startTime"`
		EndTime     int64  `json:"endTime"`
	}
	OssImage struct {
		OSSUrl      string `json:"ossUrl"`
		OSSDuration int64  `json:"ossDuration"`
		OSSSize     int64  `json:"ossSize"`
	}
	OSSOutput struct {
		Images []OssImage `json:"images"`
	}
)

type SlotStatus int // 自定义类型作为枚举的基础类型
const (
	Idleing SlotStatus = iota
	Running
)

type (
	Slot struct {
		SubID       string
		State       SlotStatus
		Session     string
		PromptID    string
		ImagePath   string
		InQueuePos  int64
		CurQueuePos int64
		InQueueTime int64
		StartTime   int64
		EndTime     int64
		TaskData    RequestData
	}
)
