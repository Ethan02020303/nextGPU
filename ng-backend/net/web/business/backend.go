package business

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/copier"
	log4plus "github.com/nextGPU/include/log4go"
	"github.com/nextGPU/include/payment/plugs"
	"github.com/nextGPU/ng-backend/common"
	"github.com/nextGPU/ng-backend/configure"
	"github.com/nextGPU/ng-backend/db"
	"github.com/nextGPU/ng-backend/header"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	NodeBase             = header.Base + 350
	RegisterError        = NodeBase + 1
	EnrollDBError        = NodeBase + 2
	NewNodeError         = NodeBase + 3
	NotFoundNodeError    = NodeBase + 4
	ConfigurationDBError = NodeBase + 5
	ValidateFailed       = NodeBase + 6
)

type NodeValidate struct {
	OS        string `json:"os,omitempty"`
	CPU       string `json:"cpu,omitempty"`
	GPU       string `json:"gpu,omitempty"`
	Memory    uint64 `json:"memory,omitempty"`
	GpuDriver string `json:"gpuDriver,omitempty"`
}

type Backend struct {
	aliPublicKey []byte
}

var gBackend *Backend

func (w *Backend) gpus(c *gin.Context) {
	funName := "gpus"
	clientIp := common.ClientIP(c)
	now := time.Now().UnixMilli()
	defer func() {
		log4plus.Info("%s clientIp=[%s] consumption time=%d(ms)", funName, clientIp, time.Now().UnixMilli()-now)
	}()
	err, gpus := db.SingletonNodeBaseDB().GPUs()
	if err != nil {
		errString := fmt.Sprintf("%s GPUs Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		common.SendError(c, header.DBError, errString)
		return
	}
	response := struct {
		CodeId int64    `json:"codeId"`
		Msg    string   `json:"msg"`
		GPUs   []db.GPU `json:"gpus"`
	}{
		CodeId: 200,
		Msg:    "success",
	}
	response.GPUs = gpus
	c.JSON(http.StatusOK, response)
}

func (w *Backend) nodes(c *gin.Context) {
	funName := "nodes"
	clientIp := common.ClientIP(c)
	now := time.Now().UnixMilli()
	defer func() {
		log4plus.Info("%s clientIp=[%s] consumption time=%d(ms)", funName, clientIp, time.Now().UnixMilli()-now)
	}()
	request := struct {
		SystemUUID string `json:"systemUUID,omitempty"`
	}{}
	if err := c.BindJSON(&request); err != nil {
		errString := fmt.Sprintf("%s BindJSON Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		common.SendError(c, header.JsonParseError, errString)
		return
	}
	if request.SystemUUID == "" {
		err, nodes := db.SingletonNodeBaseDB().Nodes()
		if err != nil {
			errString := fmt.Sprintf("%s Nodes Failed err=[%s]", funName, err.Error())
			log4plus.Error(errString)
			common.SendError(c, header.DBError, errString)
			return
		}
		response := struct {
			CodeId int64     `json:"codeId"`
			Msg    string    `json:"msg"`
			Nodes  []db.Node `json:"nodes"`
		}{
			CodeId: 200,
			Msg:    "success",
			Nodes:  nodes,
		}
		c.JSON(http.StatusOK, response)
	} else {
		err, node := db.SingletonNodeBaseDB().Node(request.SystemUUID)
		if err != nil {
			errString := fmt.Sprintf("%s Node Failed err=[%s]", funName, err.Error())
			log4plus.Error(errString)
			common.SendError(c, header.DBError, errString)
			return
		}
		response := struct {
			CodeId int64   `json:"codeId"`
			Msg    string  `json:"msg"`
			Node   db.Node `json:"node"`
		}{
			CodeId: 200,
			Msg:    "success",
			Node:   node,
		}
		c.JSON(http.StatusOK, response)
	}
}

func (w *Backend) nodeOnline(c *gin.Context) {
	funName := "nodeOnline"
	clientIp := common.ClientIP(c)
	now := time.Now().UnixMilli()
	defer func() {
		log4plus.Info("%s clientIp=[%s] consumption time=%d(ms)", funName, clientIp, time.Now().UnixMilli()-now)
	}()
	request := struct {
		SystemUUID string `json:"systemUUID,omitempty"`
		StartTime  string `json:"startTime,omitempty"`
		EndTime    string `json:"endTime,omitempty"`
	}{}
	if err := c.BindJSON(&request); err != nil {
		errString := fmt.Sprintf("%s BindJSON Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		common.SendError(c, header.JsonParseError, errString)
		return
	}
}

func (w *Backend) workflows(c *gin.Context) {
	funName := "workflows"
	clientIp := common.ClientIP(c)
	now := time.Now().UnixMilli()
	defer func() {
		log4plus.Info("%s clientIp=[%s] consumption time=%d(ms)", funName, clientIp, time.Now().UnixMilli()-now)
	}()
	request := struct {
		Title string `json:"title,omitempty"`
	}{}
	if err := c.BindJSON(&request); err != nil {
		errString := fmt.Sprintf("%s BindJSON Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		common.SendError(c, header.JsonParseError, errString)
		return
	}
	if request.Title == "" {
		err, workflows := db.SingletonNodeBaseDB().Workflows()
		if err != nil {
			errString := fmt.Sprintf("%s Workflows Failed err=[%s]", funName, err.Error())
			log4plus.Error(errString)
			common.SendError(c, header.DBError, errString)
			return
		}
		response := struct {
			CodeId    int64         `json:"codeId"`
			Msg       string        `json:"msg"`
			Workflows []db.Workflow `json:"workflows"`
		}{
			CodeId:    200,
			Msg:       "success",
			Workflows: workflows,
		}
		c.JSON(http.StatusOK, response)
	} else {
		err, workflow := db.SingletonNodeBaseDB().Workflow(request.Title)
		if err != nil {
			errString := fmt.Sprintf("%s Workflow Failed err=[%s]", funName, err.Error())
			log4plus.Error(errString)
			common.SendError(c, header.DBError, errString)
			return
		}
		response := struct {
			CodeId   int64       `json:"codeId"`
			Msg      string      `json:"msg"`
			Workflow db.Workflow `json:"workflow"`
		}{
			CodeId:   200,
			Msg:      "success",
			Workflow: workflow,
		}
		c.JSON(http.StatusOK, response)
	}
}

func (w *Backend) nodeWorkflows(c *gin.Context) {
	funName := "nodeWorkflows"
	clientIp := common.ClientIP(c)
	now := time.Now().UnixMilli()
	defer func() {
		log4plus.Info("%s clientIp=[%s] consumption time=%d(ms)", funName, clientIp, time.Now().UnixMilli()-now)
	}()
	request := struct {
		SystemUUID string `json:"systemUUID,omitempty"`
	}{}
	if err := c.BindJSON(&request); err != nil {
		errString := fmt.Sprintf("%s BindJSON Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		common.SendError(c, header.JsonParseError, errString)
		return
	}
	if request.SystemUUID == "" {
		err, nodeWorkflows := db.SingletonNodeBaseDB().NodeWorkflows()
		if err != nil {
			errString := fmt.Sprintf("%s NodeWorkflows Failed err=[%s]", funName, err.Error())
			log4plus.Error(errString)
			common.SendError(c, header.DBError, errString)
			return
		}
		response := struct {
			CodeId    int64               `json:"codeId"`
			Msg       string              `json:"msg"`
			Workflows []db.NodeWorkflowDB `json:"nodeWorkflows"`
		}{
			CodeId: 200,
			Msg:    "success",
		}
		copier.Copy(&response.Workflows, &nodeWorkflows)
		c.JSON(http.StatusOK, response)
	} else {
		err, workflow := db.SingletonNodeBaseDB().NodeWorkflow(request.SystemUUID)
		if err != nil {
			errString := fmt.Sprintf("%s NodeWorkflow Failed err=[%s]", funName, err.Error())
			log4plus.Error(errString)
			common.SendError(c, header.DBError, errString)
			return
		}
		response := struct {
			CodeId   int64                  `json:"codeId"`
			Msg      string                 `json:"msg"`
			Workflow []db.WorkspaceWorkflow `json:"nodeWorkflow"`
		}{
			CodeId:   200,
			Msg:      "success",
			Workflow: workflow,
		}
		c.JSON(http.StatusOK, response)
	}
}

func (w *Backend) getTask(c *gin.Context) {
	funName := "getTask"
	clientIp := common.ClientIP(c)
	now := time.Now().UnixMilli()
	defer func() {
		log4plus.Info("%s clientIp=[%s] consumption time=%d(ms)", funName, clientIp, time.Now().UnixMilli()-now)
	}()
	request := struct {
		TaskID string `json:"taskID"`
	}{}
	if err := c.BindJSON(&request); err != nil {
		errString := fmt.Sprintf("%s BindJSON Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		common.SendError(c, header.JsonParseError, errString)
		return
	}
	err, mainTask := db.SingletonNodeBaseDB().Task(request.TaskID)
	if err != nil {
		errString := fmt.Sprintf("%s Task Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		common.SendError(c, header.DBError, errString)
		return
	}
	response := struct {
		CodeId int64       `json:"codeId"`
		Msg    string      `json:"msg"`
		Task   db.MainTask `json:"task"`
	}{
		CodeId: 200,
		Msg:    "success",
		Task:   mainTask,
	}
	c.JSON(http.StatusOK, response)
}

func (w *Backend) getModels(c *gin.Context) {
	funName := "getModels"
	clientIp := common.ClientIP(c)
	now := time.Now().UnixMilli()
	defer func() {
		log4plus.Info("%s clientIp=[%s] consumption time=%d(ms)", funName, clientIp, time.Now().UnixMilli()-now)
	}()
	err, models := db.SingletonNodeBaseDB().Models()
	if err != nil {
		errString := fmt.Sprintf("%s Models Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		common.SendError(c, header.DBError, errString)
		return
	}
	response := struct {
		CodeId int64      `json:"codeId"`
		Msg    string     `json:"msg"`
		Models []db.Model `json:"models"`
	}{
		CodeId: 200,
		Msg:    "success",
		Models: models,
	}
	c.JSON(http.StatusOK, response)
}

func (w *Backend) getCategories(c *gin.Context) {
	funName := "getCategories"
	clientIp := common.ClientIP(c)
	now := time.Now().UnixMilli()
	defer func() {
		log4plus.Info("%s clientIp=[%s] consumption time=%d(ms)", funName, clientIp, time.Now().UnixMilli()-now)
	}()
	err, categories := db.SingletonNodeBaseDB().Categories()
	if err != nil {
		errString := fmt.Sprintf("%s Categories Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		common.SendError(c, header.DBError, errString)
		return
	}
	response := struct {
		CodeId     int64          `json:"codeId"`
		Msg        string         `json:"msg"`
		Categories []db.Categorie `json:"categories"`
	}{
		CodeId:     200,
		Msg:        "success",
		Categories: categories,
	}
	c.JSON(http.StatusOK, response)
}

func (w *Backend) getCategorie(c *gin.Context) {
	funName := "getCategorie"
	clientIp := common.ClientIP(c)
	now := time.Now().UnixMilli()
	defer func() {
		log4plus.Info("%s clientIp=[%s] consumption time=%d(ms)", funName, clientIp, time.Now().UnixMilli()-now)
	}()
	request := struct {
		CategoryName string `json:"categoryName"`
	}{}
	if err := c.BindJSON(&request); err != nil {
		errString := fmt.Sprintf("%s BindJSON Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		common.SendError(c, header.JsonParseError, errString)
		return
	}
	err, models := db.SingletonNodeBaseDB().Categorie(request.CategoryName)
	if err != nil {
		errString := fmt.Sprintf("%s Categorie Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		common.SendError(c, header.DBError, errString)
		return
	}
	response := struct {
		CodeId int64      `json:"codeId"`
		Msg    string     `json:"msg"`
		Models []db.Model `json:"models"`
	}{
		CodeId: 200,
		Msg:    "success",
		Models: models,
	}
	c.JSON(http.StatusOK, response)
}

func (w *Backend) getTags(c *gin.Context) {
	funName := "getTags"
	clientIp := common.ClientIP(c)
	now := time.Now().UnixMilli()
	defer func() {
		log4plus.Info("%s clientIp=[%s] consumption time=%d(ms)", funName, clientIp, time.Now().UnixMilli()-now)
	}()
	err, tags := db.SingletonNodeBaseDB().Tags()
	if err != nil {
		errString := fmt.Sprintf("%s Tags Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		common.SendError(c, header.DBError, errString)
		return
	}
	response := struct {
		CodeId int64    `json:"codeId"`
		Msg    string   `json:"msg"`
		Tags   []db.Tag `json:"tags"`
	}{
		CodeId: 200,
		Msg:    "success",
		Tags:   tags,
	}
	c.JSON(http.StatusOK, response)
}

func (w *Backend) getWorkflow(c *gin.Context) {
	funName := "getWorkflow"
	clientIp := common.ClientIP(c)
	now := time.Now().UnixMilli()
	defer func() {
		log4plus.Info("%s clientIp=[%s] consumption time=%d(ms)", funName, clientIp, time.Now().UnixMilli()-now)
	}()
	request := struct {
		Title string `json:"title"`
	}{}
	if err := c.BindJSON(&request); err != nil {
		errString := fmt.Sprintf("%s BindJSON Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		common.SendError(c, header.JsonParseError, errString)
		return
	}
	log4plus.Info("%s title=[%s]", funName, request.Title)
	err, workflowBase := db.SingletonNodeBaseDB().WorkflowBase(request.Title)
	if err != nil {
		errString := fmt.Sprintf("%s WorkflowBase Failed title=[%s] err=[%s]", funName, request.Title, err.Error())
		log4plus.Error(errString)
		common.SendError(c, header.DBError, errString)
		return
	}
	response := struct {
		CodeId       int64           `json:"codeId"`
		Msg          string          `json:"msg"`
		WorkflowBase db.WorkflowBase `json:"workflowBase"`
	}{
		CodeId:       200,
		Msg:          "success",
		WorkflowBase: workflowBase,
	}
	c.JSON(http.StatusOK, response)
}

func (w *Backend) getUser(c *gin.Context) {
	funName := "getUser"
	clientIp := common.ClientIP(c)
	now := time.Now().UnixMilli()
	defer func() {
		log4plus.Info("%s clientIp=[%s] consumption time=%d(ms)", funName, clientIp, time.Now().UnixMilli()-now)
	}()
	userName := c.DefaultQuery("userName", "")
	log4plus.Info("%s DefaultQuery userName=[%s]", funName, userName)
	err, userBase := db.SingletonNodeBaseDB().UserBase(userName)
	if err != nil {
		errString := fmt.Sprintf("%s UserBase Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		common.SendError(c, header.DBError, errString)
		return
	}
	response := struct {
		CodeId int64       `json:"codeId"`
		Msg    string      `json:"msg"`
		Base   db.UserBase `json:"userBase"`
	}{
		CodeId: 200,
		Msg:    "success",
		Base:   userBase,
	}
	c.JSON(http.StatusOK, response)
}

func (w *Backend) getMyTask(c *gin.Context) {
	funName := "getMyTask"
	clientIp := common.ClientIP(c)
	now := time.Now().UnixMilli()
	defer func() {
		log4plus.Info("%s clientIp=[%s] consumption time=%d(ms)", funName, clientIp, time.Now().UnixMilli()-now)
	}()
	userName := c.DefaultQuery("userName", "")
	log4plus.Info("%s DefaultQuery userName=[%s]", funName, userName)
	err, tasks := db.SingletonNodeBaseDB().MyTask(userName)
	if err != nil {
		errString := fmt.Sprintf("%s MyTask Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		common.SendError(c, header.DBError, errString)
		return
	}
	response := struct {
		CodeId int64         `json:"codeId"`
		Msg    string        `json:"msg"`
		Tasks  []db.SelfTask `json:"tasks"`
	}{
		CodeId: 200,
		Msg:    "success",
		Tasks:  tasks,
	}
	c.JSON(http.StatusOK, response)
}

func (w *Backend) getTaskCount(c *gin.Context) {
	funName := "getTaskCount"
	clientIp := common.ClientIP(c)
	now := time.Now().UnixMilli()
	defer func() {
		log4plus.Info("%s clientIp=[%s] consumption time=%d(ms)", funName, clientIp, time.Now().UnixMilli()-now)
	}()
	err, taskCount := db.SingletonNodeBaseDB().TaskCount()
	if err != nil {
		errString := fmt.Sprintf("%s TaskCount Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		common.SendError(c, header.DBError, errString)
		return
	}
	response := struct {
		CodeId    int64  `json:"codeId"`
		Msg       string `json:"msg"`
		TaskCount int64  `json:"taskCount"`
	}{
		CodeId:    200,
		Msg:       "success",
		TaskCount: taskCount,
	}
	c.JSON(http.StatusOK, response)
}

func (w *Backend) getNodeCount(c *gin.Context) {
	funName := "getNodeCount"
	clientIp := common.ClientIP(c)
	now := time.Now().UnixMilli()
	defer func() {
		log4plus.Info("%s clientIp=[%s] consumption time=%d(ms)", funName, clientIp, time.Now().UnixMilli()-now)
	}()
	err, nodeCount := db.SingletonNodeBaseDB().NodeCount()
	if err != nil {
		errString := fmt.Sprintf("%s NodeCount Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		common.SendError(c, header.DBError, errString)
		return
	}
	response := struct {
		CodeId    int64  `json:"codeId"`
		Msg       string `json:"msg"`
		NodeCount int64  `json:"nodeCount"`
	}{
		CodeId:    200,
		Msg:       "success",
		NodeCount: nodeCount,
	}
	c.JSON(http.StatusOK, response)
}

func (w *Backend) getCurNode(c *gin.Context) {
	funName := "getCurNode"
	clientIp := common.ClientIP(c)
	now := time.Now().UnixMilli()
	defer func() {
		log4plus.Info("%s clientIp=[%s] consumption time=%d(ms)", funName, clientIp, time.Now().UnixMilli()-now)
	}()
	err, nodes := db.SingletonNodeBaseDB().CurNodes()
	if err != nil {
		errString := fmt.Sprintf("%s CurNodes Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		common.SendError(c, header.DBError, errString)
		return
	}
	response := struct {
		CodeId int64            `json:"codeId"`
		Msg    string           `json:"msg"`
		Nodes  []db.CurrentNode `json:"nodes"`
	}{
		CodeId: 200,
		Msg:    "success",
		Nodes:  nodes,
	}
	c.JSON(http.StatusOK, response)
}

func (w *Backend) register(c *gin.Context) {
	funName := "register"
	clientIp := common.ClientIP(c)
	now := time.Now().UnixMilli()
	defer func() {
		log4plus.Info("%s clientIp=[%s] consumption time=%d(ms)", funName, clientIp, time.Now().UnixMilli()-now)
	}()
	request := struct {
		UserName string `json:"userName"`
		EMail    string `json:"eMail"`
		Password string `json:"password"`
	}{}
	if err := c.BindJSON(&request); err != nil {
		errString := fmt.Sprintf("%s BindJSON Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		common.SendError(c, header.JsonParseError, errString)
		return
	}
	err, success, reason := db.SingletonNodeBaseDB().Register(request.UserName, request.EMail, request.Password)
	if err != nil {
		errString := fmt.Sprintf("%s Register Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		common.SendError(c, header.DBError, errString)
		return
	}
	response := struct {
		CodeId  int64  `json:"codeId"`
		Msg     string `json:"msg"`
		Success bool   `json:"success"`
		Reason  string `json:"reason"`
	}{
		CodeId:  200,
		Msg:     "success",
		Success: success,
		Reason:  reason,
	}
	c.JSON(http.StatusOK, response)
}

func (w *Backend) login(c *gin.Context) {
	funName := "login"
	clientIp := common.ClientIP(c)
	now := time.Now().UnixMilli()
	defer func() {
		log4plus.Info("%s clientIp=[%s] consumption time=%d(ms)", funName, clientIp, time.Now().UnixMilli()-now)
	}()
	request := struct {
		UserName string `json:"userName"`
		Password string `json:"password"`
	}{}
	if err := c.BindJSON(&request); err != nil {
		errString := fmt.Sprintf("%s BindJSON Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		common.SendError(c, header.JsonParseError, errString)
		return
	}
	err, user := db.SingletonNodeBaseDB().Login(request.UserName, request.Password)
	if err != nil {
		errString := fmt.Sprintf("%s Login Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		common.SendError(c, header.DBError, errString)
		return
	}
	response := struct {
		CodeId int64       `json:"codeId"`
		Msg    string      `json:"msg"`
		Base   db.UserBase `json:"userBase"`
	}{
		CodeId: 200,
		Msg:    "success",
		Base:   user,
	}
	c.JSON(http.StatusOK, response)
}

func (w *Backend) ssoLogin(c *gin.Context) {
	funName := "ssoLogin"
	clientIp := common.ClientIP(c)
	now := time.Now().UnixMilli()
	defer func() {
		log4plus.Info("%s clientIp=[%s] consumption time=%d(ms)", funName, clientIp, time.Now().UnixMilli()-now)
	}()
	request := struct {
		UserName string `json:"userName"`
	}{}
	if err := c.BindJSON(&request); err != nil {
		errString := fmt.Sprintf("%s BindJSON Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		common.SendError(c, header.JsonParseError, errString)
		return
	}
	err, user := db.SingletonNodeBaseDB().SSOLogin(request.UserName)
	if err != nil {
		errString := fmt.Sprintf("%s SSOLogin Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		common.SendError(c, header.DBError, errString)
		return
	}
	response := struct {
		CodeId int64       `json:"codeId"`
		Msg    string      `json:"msg"`
		Base   db.UserBase `json:"userBase"`
	}{
		CodeId: 200,
		Msg:    "success",
		Base:   user,
	}
	c.JSON(http.StatusOK, response)
}

func (w *Backend) getMyNode(c *gin.Context) {
	funName := "getMyNode"
	clientIp := common.ClientIP(c)
	now := time.Now().UnixMilli()
	defer func() {
		log4plus.Info("%s clientIp=[%s] consumption time=%d(ms)", funName, clientIp, time.Now().UnixMilli()-now)
	}()
	userName := c.DefaultQuery("userName", "")
	log4plus.Info("%s DefaultQuery userName=[%s]", funName, userName)
	err, nodes := db.SingletonNodeBaseDB().UserNodes(userName)
	if err != nil {
		errString := fmt.Sprintf("%s UserNodes Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		common.SendError(c, header.DBError, errString)
		return
	}
	response := struct {
		CodeId int64         `json:"codeId"`
		Msg    string        `json:"msg"`
		Nodes  []db.UserNode `json:"nodes"`
	}{
		CodeId: 200,
		Msg:    "success",
		Nodes:  nodes,
	}
	c.JSON(http.StatusOK, response)
}

func (w *Backend) getMyAllTasks(c *gin.Context) {
	funName := "getMyAllTasks"
	clientIp := common.ClientIP(c)
	now := time.Now().UnixMilli()
	defer func() {
		log4plus.Info("%s clientIp=[%s] consumption time=%d(ms)", funName, clientIp, time.Now().UnixMilli()-now)
	}()
	userName := c.DefaultQuery("userName", "")
	log4plus.Info("%s DefaultQuery userName=[%s]", funName, userName)
	err, allTasks := db.SingletonNodeBaseDB().MyAllTasks(userName)
	if err != nil {
		errString := fmt.Sprintf("%s MyAllTasks Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		common.SendError(c, header.DBError, errString)
		return
	}
	response := struct {
		CodeId int64          `json:"codeId"`
		Msg    string         `json:"msg"`
		Tasks  []db.MyAllTask `json:"tasks"`
	}{
		CodeId: 200,
		Msg:    "success",
		Tasks:  allTasks,
	}
	c.JSON(http.StatusOK, response)
}

func (w *Backend) wechatPayment(c *gin.Context) {
	funName := "wechatPayment"
	clientIp := common.ClientIP(c)
	now := time.Now().UnixMilli()
	defer func() {
		log4plus.Info("%s clientIp=[%s] consumption time=%d(ms)", funName, clientIp, time.Now().UnixMilli()-now)
	}()
	request := struct {
		SubscriptionID string `json:"subscriptionID"`
		UserName       string `json:"userName"`
	}{}
	if err := c.BindJSON(&request); err != nil {
		errString := fmt.Sprintf("%s BindJSON Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		common.SendError(c, header.JsonParseError, errString)
		return
	}
	var amount float64
	var description string
	if request.SubscriptionID == "primary" {
		amount = 29.00 * 100
		description = "nextGPU 订阅初级会员"
	} else if request.SubscriptionID == "intermediate" {
		amount = 39.00 * 100
		description = "nextGPU 订阅中级会员"
	} else if request.SubscriptionID == "premium" {
		amount = 69.00 * 100
		description = "nextGPU 订阅高级会员"
	}
	log4plus.Info(fmt.Sprintf("%s NativePay amount=[%.2f]", funName, amount))
	err, orderID, qrURL := plugs.SingletonWechat().NativePay(int(amount), "CNY", description)
	if err != nil {
		errString := fmt.Sprintf("%s NativePay Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		common.SendError(c, header.PaymentError, errString)
		return
	}
	if err = db.SingletonSubscriptionDB().InsertSubscription(orderID,
		request.UserName, request.SubscriptionID, amount, 0); err != nil {
		errString := fmt.Sprintf("%s InsertSubscription Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		common.SendError(c, header.DBError, errString)
		return
	}
	response := struct {
		CodeId  int64  `json:"codeId"`
		Msg     string `json:"msg"`
		QRCode  string `json:"qrcode"`
		OrderID string `json:"orderID"`
	}{
		CodeId:  200,
		Msg:     "success",
		QRCode:  qrURL,
		OrderID: orderID,
	}
	c.Header("Content-Type", "application/json")
	c.JSON(http.StatusOK, response)
}

func (w *Backend) wxPayNotify(c *gin.Context) {
	funName := "wxPayNotify"
	clientIp := common.ClientIP(c)
	now := time.Now().UnixMilli()
	defer func() {
		log4plus.Info("%s clientIp=[%s] consumption time=%d(ms)", funName, clientIp, time.Now().UnixMilli()-now)
	}()
	err, transaction := plugs.SingletonWechat().NotifyHandler(c)
	if err != nil {
		errString := fmt.Sprintf("%s NotifyHandler Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		common.SendError(c, header.PaymentError, errString)
		return
	}
	errString := fmt.Sprintf("%s NotifyHandler OutTradeNo=[%s] TradeState=[%s] Total=[%d]",
		funName, transaction.OutTradeNo, transaction.TradeState, transaction.Amount.Total)
	log4plus.Info(errString)
	if err = db.SingletonSubscriptionDB().UpdateSubscription(transaction.OutTradeNo, transaction.TradeState, transaction.Amount.Total); err != nil {
		errString = fmt.Sprintf("%s UpdateSubscription Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		common.SendError(c, header.DBError, errString)
		return
	}
	if strings.EqualFold(transaction.TradeState, "SUCCESS") {
		err = db.SingletonSubscriptionDB().SetUserVip(transaction.OutTradeNo, transaction.TradeState)
		if err != nil {
			errString = fmt.Sprintf("%s SetUserVip Failed err=[%s]", funName, err.Error())
			log4plus.Error(errString)
			common.SendError(c, header.DBError, errString)
			return
		}
	}
}

func (w *Backend) wechatPayStatus(c *gin.Context) {
	funName := "wechatPayStatus"
	clientIp := common.ClientIP(c)
	now := time.Now().UnixMilli()
	defer func() {
		log4plus.Info("%s clientIp=[%s] consumption time=%d(ms)", funName, clientIp, time.Now().UnixMilli()-now)
	}()
	request := struct {
		OrderID string `json:"orderID"`
	}{}
	if err := c.BindJSON(&request); err != nil {
		errString := fmt.Sprintf("%s BindJSON Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		common.SendError(c, header.JsonParseError, errString)
		return
	}
	log4plus.Info(fmt.Sprintf("%s orderID=[%s]", funName, request.OrderID))
	err, status := db.SingletonSubscriptionDB().CheckOrderStatus(request.OrderID)
	if err != nil {
		errString := fmt.Sprintf("%s CheckOrderStatus Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		common.SendError(c, header.DBError, errString)
		return
	}
	response := struct {
		CodeId  int64  `json:"codeId"`
		Msg     string `json:"msg"`
		OrderID string `json:"orderID"`
		Status  string `json:"status"`
	}{
		CodeId:  200,
		Msg:     "success",
		OrderID: request.OrderID,
		Status:  status,
	}
	c.Header("Content-Type", "application/json")
	c.JSON(http.StatusOK, response)
}

func (w *Backend) aliPayment(c *gin.Context) {
	funName := "aliPayment"
	clientIp := common.ClientIP(c)
	now := time.Now().UnixMilli()
	defer func() {
		log4plus.Info("%s clientIp=[%s] consumption time=%d(ms)", funName, clientIp, time.Now().UnixMilli()-now)
	}()
	request := struct {
		SubscriptionID string `json:"subscriptionID"`
		UserName       string `json:"userName"`
	}{}
	if err := c.BindJSON(&request); err != nil {
		errString := fmt.Sprintf("%s BindJSON Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		common.SendError(c, header.JsonParseError, errString)
		return
	}
	var amount float64
	var description string
	if request.SubscriptionID == "primary" {
		amount = 29.00
		description = "nextGPU 订阅初级会员"
	} else if request.SubscriptionID == "intermediate" {
		amount = 39.00
		description = "nextGPU 订阅中级会员"
	} else if request.SubscriptionID == "premium" {
		amount = 69.00
		description = "nextGPU 订阅高级会员"
	}
	log4plus.Info(fmt.Sprintf("%s NativePay amount=[%.2f]", funName, amount))
	err, payUrl, orderID := plugs.SingletonAliPay().NativePay(amount, description)
	if err != nil {
		errString := fmt.Sprintf("%s NativePay Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		common.SendError(c, header.PaymentError, errString)
		return
	}
	if err = db.SingletonSubscriptionDB().InsertSubscription(orderID,
		request.UserName, request.SubscriptionID, amount, 1); err != nil {
		errString := fmt.Sprintf("%s InsertSubscription Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		common.SendError(c, header.DBError, errString)
		return
	}
	response := struct {
		CodeId  int64  `json:"codeId"`
		Msg     string `json:"msg"`
		QRCode  string `json:"qrcode"`
		OrderID string `json:"orderID"`
	}{
		CodeId:  200,
		Msg:     "success",
		QRCode:  payUrl,
		OrderID: orderID,
	}
	c.Header("Content-Type", "application/json")
	c.JSON(http.StatusOK, response)
}

func (w *Backend) aliPayNotify(c *gin.Context) {
	funName := "aliPayNotify"
	clientIp := common.ClientIP(c)
	now := time.Now().UnixMilli()
	defer func() {
		log4plus.Info("%s clientIp=[%s] consumption time=%d(ms)", funName, clientIp, time.Now().UnixMilli()-now)
	}()
	err, transaction := plugs.SingletonAliPay().NotifyHandler(c, w.aliPublicKey)
	if err != nil {
		errString := fmt.Sprintf("%s NotifyHandler Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		common.SendError(c, header.PaymentError, errString)
		return
	}
	errString := fmt.Sprintf("%s NotifyHandler OutTradeNo=[%s] TradeState=[%s] Total=[%d]",
		funName, transaction.OutTradeNo, transaction.TradeState, transaction.Amount.Total)
	log4plus.Info(errString)
	if err = db.SingletonSubscriptionDB().UpdateSubscription(transaction.OutTradeNo,
		transaction.TradeState, transaction.Amount.Total); err != nil {
		errString = fmt.Sprintf("%s UpdateSubscription Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		common.SendError(c, header.DBError, errString)
		return
	}
	if strings.EqualFold(transaction.TradeState, "SUCCESS") {
		err = db.SingletonSubscriptionDB().SetUserVip(transaction.OutTradeNo, transaction.TradeState)
		if err != nil {
			errString = fmt.Sprintf("%s SetUserVip Failed err=[%s]", funName, err.Error())
			log4plus.Error(errString)
			common.SendError(c, header.DBError, errString)
			return
		}
	}
}

func (w *Backend) aliPayStatus(c *gin.Context) {
	funName := "aliPayStatus"
	clientIp := common.ClientIP(c)
	now := time.Now().UnixMilli()
	defer func() {
		log4plus.Info("%s clientIp=[%s] consumption time=%d(ms)", funName, clientIp, time.Now().UnixMilli()-now)
	}()
	request := struct {
		OrderID string `json:"orderID"`
	}{}
	if err := c.BindJSON(&request); err != nil {
		errString := fmt.Sprintf("%s BindJSON Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		common.SendError(c, header.JsonParseError, errString)
		return
	}
	log4plus.Info(fmt.Sprintf("%s orderID=[%s]", funName, request.OrderID))
	err, status := db.SingletonSubscriptionDB().CheckOrderStatus(request.OrderID)
	if err != nil {
		errString := fmt.Sprintf("%s CheckOrderStatus Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		common.SendError(c, header.DBError, errString)
		return
	}
	response := struct {
		CodeId  int64  `json:"codeId"`
		Msg     string `json:"msg"`
		OrderID string `json:"orderID"`
		Status  string `json:"status"`
	}{
		CodeId:  200,
		Msg:     "success",
		OrderID: request.OrderID,
		Status:  status,
	}
	c.Header("Content-Type", "application/json")
	c.JSON(http.StatusOK, response)
}

func (w *Backend) ssoUserinfo(c *gin.Context) {
	funName := "ssoUserinfo"
	clientIp := common.ClientIP(c)
	now := time.Now().UnixMilli()
	defer func() {
		log4plus.Info("%s clientIp=[%s] consumption time=%d(ms)", funName, clientIp, time.Now().UnixMilli()-now)
	}()
	authHeader := c.GetHeader("Authorization")
	// 处理认证逻辑
	if strings.EqualFold(authHeader, "") {
		errString := fmt.Sprintf("%s GetHeader Failed err=[not found Authorization]", funName)
		log4plus.Error(errString)
		common.SendError(c, http.StatusUnauthorized, errString)
		return
	}
	log4plus.Info("---------->>>>>>>>authHeader=[%s]", authHeader)
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if strings.EqualFold(token, "") {
		errString := fmt.Sprintf("%s GetHeader Failed err=[not found token]", funName)
		log4plus.Error(errString)
		common.SendError(c, http.StatusUnauthorized, errString)
		return
	}
	err, body := w.getRequestNoPem(token, "https://keycloak.local.moojnn.com/realms/aip/protocol/openid-connect/userinfo")
	if err != nil {
		errString := fmt.Sprintf("%s getRequest Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		common.SendError(c, http.StatusUnauthorized, errString)
		return
	}
	log4plus.Info("---------->>>>>>>>body=[%s]", string(body))

	type UserInfo struct {
		Sub               string `json:"sub"`
		EmailVerified     bool   `json:"email_verified"`
		Name              string `json:"name"`
		PreferredUsername string `json:"preferred_username"`
		GivenName         string `json:"given_name"`
		FamilyName        string `json:"family_name"`
		Email             string `json:"email"`
	}
	var user UserInfo
	err = json.Unmarshal(body, &user)
	if err != nil {
		errString := fmt.Sprintf("%s Unmarshal Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		common.SendError(c, http.StatusUnauthorized, errString)
		return
	}
	response := struct {
		CodeId   int64  `json:"codeId"`
		Msg      string `json:"msg"`
		UserName string `json:"userName"`
	}{
		CodeId:   200,
		Msg:      "success",
		UserName: user.PreferredUsername,
	}
	c.Header("Content-Type", "application/json")
	c.JSON(http.StatusOK, response)
}

func (w *Backend) getRequestNoPem(token string, url string) (error, []byte) {
	funName := "getRequestNoPem"
	now := time.Now().UnixMilli()
	defer func() {
		log4plus.Info("%s token=[%s] url=[%s] consumption time=%d(ms)", funName, token, url, time.Now().UnixMilli()-now)
	}()
	log4plus.Info("%s parse url=[%s]", funName, url)
	client := &http.Client{
		Transport: &http.Transport{
			Dial: func(netw, addr string) (net.Conn, error) {
				c, err := net.DialTimeout(netw, addr, time.Minute*10)
				if err != nil {
					log4plus.Error("%s dail timeout err=[%s]", funName, err.Error())
					return nil, err
				}
				return c, nil
			},
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // 跳过证书验证
			},
			MaxIdleConnsPerHost:   10,
			ResponseHeaderTimeout: time.Minute * 10,
		},
	}
	defer client.CloseIdleConnections()

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log4plus.Error("%s NewRequest Failed url=[%s] err=[%s]", funName, url, err.Error())
		return err, []byte{}
	}
	request.Header.Set("Content-Type", "application/json")               // 设置内容类型
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token)) // 设置认证头
	response, err := client.Do(request)
	if err != nil {
		log4plus.Error("%s Do Failed url=[%s] err=[%s]", funName, url, err.Error())
		return err, []byte{}
	}
	defer response.Body.Close()
	log4plus.Info("%s Check StatusCode=[%d]", funName, response.StatusCode)
	if response.StatusCode != 200 {
		log4plus.Error("%s Do url=[%s] StatusCode=[%d]", funName, url, response.StatusCode)
		return err, []byte{}
	}
	repBody, err := io.ReadAll(response.Body)
	if err != nil {
		log4plus.Error("%s ReadAll Failed url=[%s] err=[%s]", funName, url, err.Error())
		return err, []byte{}
	}
	return nil, repBody
}

func (w *Backend) getRequest(token string, url string) (error, []byte) {
	funName := "getRequest"
	now := time.Now().UnixMilli()
	defer func() {
		log4plus.Info("%s token=[%s] url=[%s] consumption time=%d(ms)", funName, token, url, time.Now().UnixMilli()-now)
	}()
	log4plus.Info("%s parse url=[%s]", funName, url)
	client := &http.Client{
		Transport: &http.Transport{
			Dial: func(netw, addr string) (net.Conn, error) {
				c, err := net.DialTimeout(netw, addr, time.Minute*10)
				if err != nil {
					log4plus.Error("%s dail timeout err=[%s]", funName, err.Error())
					return nil, err
				}
				return c, nil
			},
			MaxIdleConnsPerHost:   10,
			ResponseHeaderTimeout: time.Minute * 10,
		},
	}
	defer client.CloseIdleConnections()

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log4plus.Error("%s NewRequest Failed url=[%s] err=[%s]", funName, url, err.Error())
		return err, []byte{}
	}
	request.Header.Set("Content-Type", "application/json")               // 设置内容类型
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token)) // 设置认证头
	response, err := client.Do(request)
	if err != nil {
		log4plus.Error("%s Do Failed url=[%s] err=[%s]", funName, url, err.Error())
		return err, []byte{}
	}
	defer response.Body.Close()
	log4plus.Info("%s Check StatusCode=[%d]", funName, response.StatusCode)
	if response.StatusCode != 200 {
		log4plus.Error("%s Do url=[%s] StatusCode=[%d]", funName, url, response.StatusCode)
		return err, []byte{}
	}
	repBody, err := io.ReadAll(response.Body)
	if err != nil {
		log4plus.Error("%s ReadAll Failed url=[%s] err=[%s]", funName, url, err.Error())
		return err, []byte{}
	}
	return nil, repBody
}

func (w *Backend) postRequest(token string, url string, body []byte) (error, []byte) {
	funName := "postRequest"
	now := time.Now().UnixMilli()
	defer func() {
		log4plus.Info("%s token=[%s] url=[%s] consumption time=%d(ms)", funName, token, url, time.Now().UnixMilli()-now)
	}()
	log4plus.Info("%s parse url=[%s]", funName, url)
	client := &http.Client{
		Transport: &http.Transport{
			Dial: func(netw, addr string) (net.Conn, error) {
				c, err := net.DialTimeout(netw, addr, time.Minute*10)
				if err != nil {
					log4plus.Error("%s dail timeout err=[%s]", funName, err.Error())
					return nil, err
				}
				return c, nil
			},
			MaxIdleConnsPerHost:   10,
			ResponseHeaderTimeout: time.Minute * 10,
		},
	}
	defer client.CloseIdleConnections()

	bodyReader := bytes.NewReader(body)
	request, err := http.NewRequest("POST", url, bodyReader)
	if err != nil {
		log4plus.Error("%s NewRequest Failed url=[%s] err=[%s]", funName, url, err.Error())
		return err, []byte{}
	}
	request.Header.Set("Content-Type", "application/json")               // 设置内容类型
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token)) // 设置认证头
	response, err := client.Do(request)
	if err != nil {
		log4plus.Error("%s Do Failed url=[%s] err=[%s]", funName, url, err.Error())
		return err, []byte{}
	}
	defer response.Body.Close()

	log4plus.Info("%s Check StatusCode=[%d]", funName, response.StatusCode)
	if response.StatusCode != 200 {
		log4plus.Error("%s Do url=[%s] StatusCode=[%d]", funName, url, response.StatusCode)
		return err, []byte{}
	}
	repBody, err := io.ReadAll(response.Body)
	if err != nil {
		log4plus.Error("%s ReadAll Failed url=[%s] err=[%s]", funName, url, err.Error())
		return err, []byte{}
	}
	return nil, repBody
}

func (w *Backend) InitPayment(wechat, ali bool) bool {
	funName := "InitPayment"
	if wechat {
		/* init wechat */
		privateKey, err := os.ReadFile(configure.SingletonConfigure().Payment.Wechat.PrivateKeyFile)
		if err != nil {
			errString := fmt.Sprintf("%s ReadFile Failed err=[%s]", funName, err.Error())
			log4plus.Error(errString)
			return false
		}
		err = plugs.SingletonWechat().Init(configure.SingletonConfigure().Payment.Wechat.MCHid,
			configure.SingletonConfigure().Payment.Wechat.SerialNo,
			configure.SingletonConfigure().Payment.Wechat.ApiV3Key,
			string(privateKey),
			configure.SingletonConfigure().Payment.Wechat.AppID,
			configure.SingletonConfigure().Payment.Wechat.NotifyUrl,
			configure.SingletonConfigure().Payment.Wechat.NotifyUrl)
		if err != nil {
			errString := fmt.Sprintf("%s Init Failed err=[%s]", funName, err.Error())
			log4plus.Error(errString)
			return false
		}
	}
	if ali {
		/* init ali */
		privateKey, err := os.ReadFile(configure.SingletonConfigure().Payment.Ali.APPPrivateKey)
		if err != nil {
			errString := fmt.Sprintf("%s ReadFile Failed privateKeyFile=[%s] err=[%s]",
				funName, configure.SingletonConfigure().Payment.Ali.APPPrivateKey, err.Error())
			log4plus.Error(errString)
			return false
		}
		appPublicKeyContent, err := os.ReadFile(configure.SingletonConfigure().Payment.Ali.APPPublicKey)
		if err != nil {
			errString := fmt.Sprintf("%s ReadFile Failed appPublicKey=[%s] err=[%s]",
				funName, configure.SingletonConfigure().Payment.Ali.APPPublicKey, err.Error())
			log4plus.Error(errString)
			return false
		}
		//aliPublicKeyContent, err = os.ReadFile(configure.SingletonConfigure().Payment.Ali.AliPublicKey)
		w.aliPublicKey, err = os.ReadFile(configure.SingletonConfigure().Payment.Ali.AliPublicKey)
		if err != nil {
			errString := fmt.Sprintf("%s ReadFile Failed aliPublicKey=[%s] err=[%s]",
				funName, configure.SingletonConfigure().Payment.Ali.AliPublicKey, err.Error())
			log4plus.Error(errString)
			return false
		}
		aliRootKeyContent, err := os.ReadFile(configure.SingletonConfigure().Payment.Ali.AliRootKey)
		if err != nil {
			errString := fmt.Sprintf("%s ReadFile Failed aliRootKey=[%s] err=[%s]",
				funName, configure.SingletonConfigure().Payment.Ali.AliRootKey, err.Error())
			log4plus.Error(errString)
			return false
		}
		err = plugs.SingletonAliPay().Init(configure.SingletonConfigure().Payment.Ali.AppID,
			string(privateKey), appPublicKeyContent, w.aliPublicKey, aliRootKeyContent,
			configure.SingletonConfigure().Payment.Ali.IsProd, configure.SingletonConfigure().Payment.Ali.NotifyUrl)
		if err != nil {
			errString := fmt.Sprintf("%s Init Failed err=[%s]", funName, err.Error())
			log4plus.Error(errString)
			return false
		}
	}
	return true
}

const (
	KeyCloakEndpoint = "https://keycloak.local.moojnn.com"
	KeyCloakRealm    = "aip"
)

func handleUnauthorized(c *gin.Context) {
	if c.Request.URL.Path == "/user/login/" {
		c.Redirect(http.StatusFound, "https://aip.local.moojnn.com")
		return
	}
	c.AbortWithStatus(http.StatusUnauthorized)
}

func getUserInfoFromKeyCloak(token string) (map[string]interface{}, error) {
	funName := "getUserInfoFromKeyCloak"
	client := &http.Client{Timeout: 5 * time.Second}
	url := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/userinfo", KeyCloakEndpoint, KeyCloakRealm)
	req, err := http.NewRequest(
		"GET",
		url,
		nil,
	)
	if err != nil {
		errString := fmt.Sprintf("%s NewRequest url=[%s] err=[%s]", funName, url, err.Error())
		log4plus.Error(errString)
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	if err != nil {
		errString := fmt.Sprintf("%s Do err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		errString := fmt.Sprintf("%s keycloak returned status StatusCode=[%d]", funName, resp.StatusCode)
		log4plus.Error(errString)
		return nil, errors.New(errString)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		errString := fmt.Sprintf("%s ReadAll err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return nil, err
	}
	var result map[string]interface{}
	if err = json.Unmarshal(body, &result); err != nil {
		errString := fmt.Sprintf("%s Unmarshal err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return nil, err
	}
	return result, nil
}

func (w *Backend) SSOAuthMiddleware() gin.HandlerFunc {
	funName := "SSOAuthMiddleware"
	return func(c *gin.Context) {
		token, err := c.Cookie("kc-access")
		if err != nil {
			errString := fmt.Sprintf("%s Cookie No kc-access token found in cookies", funName)
			log4plus.Error(errString)
			handleUnauthorized(c)
			return
		}
		userInfo, err := getUserInfoFromKeyCloak(token)
		if err != nil {
			errString := fmt.Sprintf("%s getUserInfoFromKeyCloak Failed to get userinfo from KeyCloak err=[%s]", funName, err.Error())
			log4plus.Error(errString)
			handleUnauthorized(c)
			return
		}
		email, ok := userInfo["email"].(string)
		if !ok || email == "" {
			errString := fmt.Sprintf("%s Missing email in userinfo response", funName)
			log4plus.Error(errString)
			handleUnauthorized(c)
			return
		}
		userName, ok := userInfo["preferred_username"].(string)
		if !ok || userName == "" {
			userName, _ = userInfo["sub"].(string)
		}
		log4plus.Info("%s DefaultQuery userName=[%s]", funName, userName)
		err, userBase := db.SingletonNodeBaseDB().UserBase(userName)
		if err != nil {
			errString := fmt.Sprintf("%s UserBase Failed err=[%s]", funName, err.Error())
			log4plus.Error(errString)
			handleUnauthorized(c)
			return
		}
		c.Set("user", userBase)
		c.Set("is_keycloak_auth", true)
		c.Next()
	}
}

func (w *Backend) Start(nodeGroup *gin.RouterGroup) {
	nodeGroup.POST("/gpus", w.gpus)
	nodeGroup.POST("/nodes", w.nodes)
	nodeGroup.POST("/nodeOnline", w.nodeOnline)
	nodeGroup.POST("/workflows", w.workflows)
	nodeGroup.POST("/nodeWorkflows", w.nodeWorkflows)
	nodeGroup.POST("/getTask", w.getTask)

	/*backend*/
	nodeGroup.POST("/register", w.register)
	nodeGroup.POST("/login", w.login)
	nodeGroup.GET("/userInfo", w.getUser)
	nodeGroup.GET("/models", w.getModels)
	nodeGroup.GET("/categories", w.getCategories)
	nodeGroup.POST("/categorie", w.getCategorie)
	nodeGroup.GET("/getTags", w.getTags)
	nodeGroup.POST("/workflow", w.getWorkflow)
	nodeGroup.GET("/userTasks", w.getMyTask)
	nodeGroup.GET("/taskCount", w.getTaskCount)
	nodeGroup.GET("/nodeCount", w.getNodeCount)
	nodeGroup.GET("/availableNodes", w.getCurNode)
	nodeGroup.GET("/userNodes", w.getMyNode)
	nodeGroup.GET("/userAllTasks", w.getMyAllTasks)

	//SSO
	nodeGroup.POST("/ssoUserinfo", w.ssoUserinfo)
	nodeGroup.POST("/ssoLogin", w.ssoLogin)

	/*payment*/
	//wechat
	nodeGroup.POST("/wxPayment", w.wechatPayment)
	nodeGroup.POST("/wxNotify", w.wxPayNotify)
	nodeGroup.POST("/wxStatus", w.wechatPayStatus)
	//ali
	nodeGroup.POST("/aliPayment", w.aliPayment)
	nodeGroup.POST("/aliNotify", w.aliPayNotify)
	nodeGroup.POST("/aliStatus", w.aliPayStatus)
}

func SingletonBackend() *Backend {
	if gBackend == nil {
		gBackend = &Backend{}
		gBackend.InitPayment(true, true)
	}
	return gBackend
}
