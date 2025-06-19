package business

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	log4plus "github.com/nextGPU/include/log4go"
	"github.com/nextGPU/ng-cloud/common"
	"github.com/nextGPU/ng-cloud/db"
	"github.com/nextGPU/ng-cloud/header"
	"github.com/nextGPU/ng-cloud/process"
	"net/http"
	"time"
)

type (
	UserEnrollResponse struct {
		CodeId  int64  `json:"codeId"`
		Msg     string `json:"msg"`
		Session string `json:"session"`
		Connect string `json:"connect"`
	}
)

type User struct {
}

var gUser *User

func (w *User) publish(c *gin.Context) {
	funName := "publish"
	clientIp := common.ClientIP(c)
	now := time.Now().UnixMilli()
	defer func() {
		log4plus.Info("%s clientIp=[%s] consumption time=%d(ms)", funName, clientIp, time.Now().UnixMilli()-now)
	}()
	request := struct {
		Title string `json:"title"`
		//InputType             string          `json:"inputType"`
		InputImageScaledRatio int64           `json:"inputImageScaledRatio,omitempty"`
		ImageCount            int64           `json:"imageCount,omitempty"`
		ImagePath             string          `json:"imagePath,omitempty"`
		TaskID                string          `json:"taskID"`
		Data                  json.RawMessage `json:"data"`
	}{}
	if err := c.BindJSON(&request); err != nil {
		errString := fmt.Sprintf("%s BindJSON Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		common.SendError(c, header.JsonParseError, errString)
		return
	}
	newSessionID := common.SessionID()
	log4plus.Info("%s session=[%s] request=[%#v]", funName, newSessionID, request)
	var err error
	err, userSession := process.SingletonUsers().NewSession(newSessionID, clientIp)
	if err != nil {
		errString := fmt.Sprintf("%s BindJSON Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		common.SendError(c, header.JsonParseError, errString)
		return
	}
	if request.InputImageScaledRatio == 0 {
		request.InputImageScaledRatio = 1
	}
	if request.ImageCount == 0 {
		request.ImageCount = 1
	}
	message, _ := json.Marshal(request)
	errPublish, publishRes := userSession.Match.UserPublish(message)
	if errPublish != nil {
		errString := fmt.Sprintf("%s BindJSON Failed err=[%s]", funName, errPublish.Error())
		log4plus.Error(errString)
		common.SendError(c, header.JsonParseError, errString)
		return
	}
	c.JSON(http.StatusOK, publishRes)
}

func (w *User) delete(c *gin.Context) {
	funName := "delete"
	clientIp := common.ClientIP(c)
	now := time.Now().UnixMilli()
	defer func() {
		log4plus.Info("%s clientIp=[%s] consumption time=%d(ms)", funName, clientIp, time.Now().UnixMilli()-now)
	}()
	request := struct {
		Session string `json:"session"`
		TaskID  string `json:"taskID"`
	}{}
	if err := c.BindJSON(&request); err != nil {
		errString := fmt.Sprintf("%s BindJSON Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		common.SendError(c, header.JsonParseError, errString)
		return
	}
	session := process.SingletonUsers().FindSession(request.Session)
	if session == nil {
		errString := fmt.Sprintf("%s FindSession not found session=[%s]", funName, request.Session)
		log4plus.Error(errString)
		common.SendError(c, header.JsonParseError, errString)
		return
	}
	userRequest := struct {
		TaskID string `json:"taskID"`
	}{
		TaskID: request.TaskID,
	}
	message, _ := json.Marshal(userRequest)
	errStop, stopRes := session.Match.UserDelete(message)
	if errStop != nil {
		errString := fmt.Sprintf("%s BindJSON Failed err=[%s]", funName, errStop.Error())
		log4plus.Error(errString)
		common.SendError(c, header.JsonParseError, errString)
		return
	}
	c.JSON(http.StatusOK, stopRes)
}

func (w *User) getTask(c *gin.Context) {
	funName := "getTask"
	clientIp := common.ClientIP(c)
	now := time.Now().UnixMilli()
	defer func() {
		log4plus.Info("%s clientIp=[%s] consumption time=%d(ms)", funName, clientIp, time.Now().UnixMilli()-now)
	}()
	request := struct {
		TaskID string `json:"taskId"`
	}{}
	if err := c.BindJSON(&request); err != nil {
		errString := fmt.Sprintf("%s BindJSON Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		common.SendError(c, header.JsonParseError, errString)
		return
	}
	log4plus.Info("%s DefaultQuery taskID=[%s]", funName, request.TaskID)

	err, major := db.SingletonTaskDB().GetTaskID(request.TaskID)
	if err != nil {
		errString := fmt.Sprintf("%s BindJSON Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		common.SendError(c, header.DBError, errString)
		return
	}
	taskResponse := struct {
		ResultID   int64  `json:"result"`
		ReturnCode int64  `json:"returnCode"`
		ReturnMsg  string `json:"returnMsg"`
		ProductID  string `json:"productId"`
		Consume    int64  `json:"consume"`
		TaskID     string `json:"taskId"`
		Status     int64  `json:"status"`
		Output     string `json:"output"`
	}{
		ReturnCode: 0,
		ReturnMsg:  "success",
		ProductID:  "nextGPU",
		Consume:    major.Duration,
		TaskID:     request.TaskID,
	}
	if major.State == 0 {
		taskResponse.ResultID = 0
		taskResponse.Status = 0
	} else if major.State == 1 || major.State == 2 {
		taskResponse.ResultID = 0
		taskResponse.Status = 1
	} else if major.State == 3 {
		taskResponse.ResultID = 1
		taskResponse.Status = 2
	}
	jsonBytes, err := json.Marshal(major)
	if err != nil {
		errString := fmt.Sprintf("%s json.Marshal Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		common.SendError(c, header.JsonParseError, errString)
		return
	}
	taskResponse.Output = string(jsonBytes)
	c.JSON(http.StatusOK, taskResponse)
}

func (w *User) Start(userGroup *gin.RouterGroup) {
	userGroup.POST("/publish", w.publish)
	userGroup.POST("/getTask", w.getTask)
}

func SingletonUser() *User {
	if gUser == nil {
		gUser = &User{}
	}
	return gUser
}
