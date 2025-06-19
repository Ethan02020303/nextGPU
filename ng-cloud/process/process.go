package process

import (
	"fmt"
	log4plus "github.com/nextGPU/include/log4go"
	"github.com/nextGPU/ng-cloud/db"
	"sort"
	"strings"
	"time"
)

type Process struct {
}

var gProcess *Process

func (p *Process) showNodeState(state db.NodeStatus) string {
	if state == db.Unavailable {
		return "Unavailable"
	} else if state == db.Disconnect {
		return "Disconnect"
	} else if state == db.Enrolling {
		return "Enrolling"
	} else if state == db.Enrolled {
		return "Enrolled"
	} else if state == db.Connecting {
		return "Connecting"
	} else if state == db.Connected {
		return "Connected"
	} else if state == db.Measuring {
		return "Measuring"
	} else if state == db.MeasurementFailed {
		return "MeasurementFailed"
	} else if state == db.MeasurementComplete {
		return "MeasurementComplete"
	} else {
		return "unDefine"
	}
}

func (p *Process) showDockerState(state db.DockerStatus) string {
	if state == db.UnInstalled {
		return "unInstalled"
	} else if state == db.Pulling {
		return "Pulling"
	} else if state == db.PullFailed {
		return "PullFailed"
	} else if state == db.PullCompleted {
		return "PullCompleted"
	} else if state == db.Startuping {
		return "Startuping"
	} else if state == db.StartupFailed {
		return "StartupFailed"
	} else if state == db.StartupCompleted {
		return "StartupCompleted"
	} else if state == db.StartupSuccess {
		return "StartupSuccess"
	} else if state == db.DockerStatusMax {
		return "DockerStatusMax"
	} else if state == db.Passed {
		return "Passed"
	} else if state == db.UnPassed {
		return "unPassed"
	} else {
		return "unDefine"
	}
}

func (p *Process) showTaskState(state TaskState) string {
	if state == WaitingTask {
		return "Waiting"
	} else if state == RunningTask {
		return "Running"
	} else if state == PublishingTask {
		return "PublishingTask"
	} else if state == PublishedTask {
		return "PublishedTask"
	} else if state == CompletedTask {
		return "Completed"
	} else if state == FailedTask {
		return "Failed"
	} else {
		return "unDefine"
	}
}

func (p *Process) showNodes() {
	log4plus.Info("allNode ---->>>> idleSlots=[%d] runningSlots=[%d]", SingletonNodes().GetIdleSlotNum(), SingletonNodes().GetRunningSlotNum())
	nodes := SingletonNodes().GetNodes()
	sort.Slice(nodes, func(p, q int) bool { return nodes[p].SystemUUID < nodes[q].SystemUUID })
	for _, node := range nodes {
		ip, port := node.Match.IpPort()
		if node.Base != nil {
			log4plus.Info("	systemUUID=[%s] measureTime=[%s] slotSize=[%d] state=[%s] ip=[%s] port=[%s] handle=[%d] ",
				node.SystemUUID,
				node.MeasureTime.Format("2006-01-02 15:04:05"),
				node.SlotSize,
				p.showNodeState(node.State),
				ip,
				port,
				node.Match.SelfHandle())
			for _, docker := range node.Base.Dockers {
				titles := SingletonNodes().GetTitlesFromWorkSpaceID(node.SystemUUID, docker.WorkSpaceID)
				titleStrings := strings.Join(titles, ",")
				log4plus.Info("		workspaceID=[%s] state=[%s] titles=[%s]", docker.WorkSpaceID, p.showDockerState(docker.State), titleStrings)
			}
		}
	}
}

func (p *Process) showSessions() {
	log4plus.Info("allSessions ---->>>> sessions=[%d]", SingletonUsers().GetSessionNum())
	sessions := SingletonUsers().GetSessions()
	sort.Slice(sessions, func(p, q int) bool { return sessions[p].Session < sessions[q].Session })
	for _, user := range sessions {
		for index, handle := range user.Handle {
			log4plus.Info("		session=[%s] ip=[%s] port=[%s] handle=[%d] ", user.Session, user.IP[index], user.Port[index], handle)
		}
	}
}

func (p *Process) showTasks() {
	log4plus.Info("allTasks ---->>>> waitTask=[%d] runningTask=[%d]", SingletonUsers().WaitQueueSize(), SingletonUsers().ProcessingNum())

	sessionTasks := SingletonUsers().GetShowTasks()
	sort.Slice(sessionTasks, func(p, q int) bool { return sessionTasks[p].TaskID < sessionTasks[q].TaskID })
	for _, task := range sessionTasks {
		log4plus.Info("	Main--->>>	session=[%s] taskID=[%s] state=[%s] imageNum=[%d] publishTime=[%d] ",
			task.Session, task.TaskID, p.showTaskState(task.State), task.ImageNum, task.PublishTime)
		for _, subTask := range task.ShowTasks {
			if subTask.State == WaitingTask {
				log4plus.Info("		⏳ >>>> 🆔[%s] 🚧=[%s] 🖥️[%s] ",
					subTask.SubID,
					fmt.Sprintf("%d/%d", subTask.CurWaitCount, subTask.InitWaitCount),
					"")
			} else if subTask.State == PublishingTask {
				log4plus.Info("		✈️ >>>> 🆔[%s] 🚧=[%s] 🖥️[%s]",
					subTask.SubID,
					fmt.Sprintf("%d/%d", subTask.CurWaitCount, subTask.InitWaitCount),
					subTask.SystemUUID)
			} else if subTask.State == PublishedTask {
				log4plus.Info("		📬 >>>> 🆔[%s] 🚧=[%s] 🖥️[%s]",
					subTask.SubID,
					fmt.Sprintf("%d/%d", subTask.CurWaitCount, subTask.InitWaitCount),
					subTask.SystemUUID)
			} else if subTask.State == RunningTask {
				log4plus.Info("		🔄 >>>> 🆔[%s] 🚧=[%s] 🖥️[%s] 🎬[%d] ⏳[%d] ",
					subTask.SubID, fmt.Sprintf("%d/%d", subTask.CurWaitCount, subTask.InitWaitCount),
					subTask.SystemUUID, subTask.StartTime, subTask.EstimateMs)
			} else if subTask.State == CompletedTask {
				log4plus.Info("		🏁 >>>> 🆔[%s] 🚧=[%s] 🖥️[%s] 🎬[%d] ⏳[%d] 💸[%d]",
					subTask.SubID,
					fmt.Sprintf("%d/%d", subTask.CurWaitCount, subTask.InitWaitCount),
					subTask.SystemUUID, subTask.StartTime, subTask.EstimateMs, subTask.CompletedTime-subTask.StartTime)
			} else if subTask.State == FailedTask {
				log4plus.Info("		🔴 >>>> 🆔[%s] 🖥️[%s] 🎬[%d]",
					subTask.SubID, subTask.SystemUUID, subTask.StartTime)
			}
		}
	}
}

func (p *Process) showWorkspaces() {
	workSpaces := SingletonWorkSpaces().AllWorkSpaces()
	sort.Slice(workSpaces, func(p, q int) bool {
		return workSpaces[p].Base.WorkspaceID < workSpaces[q].Base.WorkspaceID
	})
	for _, workSpace := range workSpaces {
		if workSpace.Base.Deleted == db.UnDelete {
			var dockerNames []string
			for _, v := range workSpace.Base.MajorCompose.Services {
				dockerNames = append(dockerNames, v.Image)
			}
			for _, v := range workSpace.Base.MinorCompose.Services {
				dockerNames = append(dockerNames, v.Image)
			}
			dockerNamesString := strings.Join(dockerNames, ",")
			log4plus.Info("	workspaceID=[%s] ImageName=[%s] ", workSpace.Base.WorkspaceID, dockerNamesString)
		}
	}
}

func (p *Process) pollShow() {
	showTicker := time.NewTicker(10 * time.Second)
	defer showTicker.Stop()

	for {
		select {
		case <-showTicker.C: //展示
			p.showNodes()
			p.showSessions()
			p.showTasks()
			p.showWorkspaces()
			log4plus.Info("--------------------------------------")
		}
	}
}
func SingletonProcess() *Process {
	funName := "SingletonProcess"
	if gProcess == nil {
		gProcess = &Process{}
		log4plus.Info("%s start pollShow", funName)
		go gProcess.pollShow()
	}
	return gProcess
}
