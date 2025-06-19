package business

import (
	"encoding/json"
	"fmt"
	"github.com/nextGPU/ng-cloud/common"
	"github.com/nextGPU/ng-cloud/header"
	"github.com/nextGPU/ng-cloud/process"
	log4plus "github.com/nextGPU/include/log4go"
	"github.com/gin-gonic/gin"
	"time"
)

type WorkSpace struct {
}

var gWorkSpace *WorkSpace

func (w *WorkSpace) start(c *gin.Context) {
	funName := "start"
	clientIp := common.ClientIP(c)
	now := time.Now().UnixMilli()
	defer func() {
		log4plus.Info("%s clientIp=[%s] consumption time=%d(ms)", funName, clientIp, time.Now().UnixMilli()-now)
	}()
	systemUUID := c.DefaultQuery("systemUUID", "")
	workSpaceID := c.DefaultQuery("workSpaceID", "")
	log4plus.Info("%s systemUUID=[%s] workSpaceID=[%s]", funName, systemUUID, workSpaceID)
	node := process.SingletonNodes().FindNode(systemUUID)
	if node == nil {
		errString := fmt.Sprintf("%s not found node systemUUID=[%s]", funName, systemUUID)
		log4plus.Error(errString)
		common.SendError(c, header.JsonParseError, errString)
		return
	}
	start := struct {
		MessageID   int64  `json:"messageID"`
		WorkSpaceID string `json:"workSpaceID"`
		StartupCMD  string `json:"startupCMD"`
	}{
		MessageID:   process.StartupWorkSpace,
		WorkSpaceID: workSpaceID,
		StartupCMD:  "",
	}
	data, err := json.Marshal(start)
	if err != nil {
		errString := fmt.Sprintf("%s Marshal Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		common.SendError(c, header.JsonParseError, errString)
		return
	}
	if node.Match != nil && node.Net != nil {
		node.Match.SendMessageCh <- data
	}
	common.SendSuccess(c)
}

func (w *WorkSpace) delete(c *gin.Context) {
	funName := "delete"
	clientIp := common.ClientIP(c)
	now := time.Now().UnixMilli()
	defer func() {
		log4plus.Info("%s clientIp=[%s] consumption time=%d(ms)", funName, clientIp, time.Now().UnixMilli()-now)
	}()
	systemUUID := c.DefaultQuery("systemUUID", "")
	workSpaceID := c.DefaultQuery("workSpaceID", "")
	log4plus.Info("%s systemUUID=[%s] workSpaceID=[%s]", funName, systemUUID, workSpaceID)
	node := process.SingletonNodes().FindNode(systemUUID)
	if node == nil {
		errString := fmt.Sprintf("%s not found node systemUUID=[%s]", funName, systemUUID)
		log4plus.Error(errString)
		common.SendError(c, header.JsonParseError, errString)
		return
	}
	deleteWorkSpace := struct {
		MessageID   int64  `json:"messageID"`
		WorkSpaceID string `json:"workSpaceID"`
	}{
		MessageID:   process.DelWorkSpace,
		WorkSpaceID: workSpaceID,
	}
	data, err := json.Marshal(deleteWorkSpace)
	if err != nil {
		errString := fmt.Sprintf("%s Marshal Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		common.SendError(c, header.JsonParseError, errString)
		return
	}
	if node.Match != nil && node.Net != nil {
		node.Match.SendMessageCh <- data
	}
	common.SendSuccess(c)
}

func (w *WorkSpace) Start(userGroup *gin.RouterGroup) {
	userGroup.GET("/start", w.start)
	userGroup.GET("/delete", w.delete)
}

func SingletonWorkSpace() *WorkSpace {
	if gWorkSpace == nil {
		gWorkSpace = &WorkSpace{}
	}
	return gWorkSpace
}
