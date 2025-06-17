package business

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/copier"
	log4plus "github.com/nextGPU/include/log4go"
	"github.com/nextGPU/ng-backend/common"
	"github.com/nextGPU/ng-backend/db"
	"github.com/nextGPU/ng-backend/header"
	"net/http"
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
		errString := fmt.Sprintf("%s SingletonNodeBaseDB().GPUs Failed err=[%s]", funName, err.Error())
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
			errString := fmt.Sprintf("%s SingletonNodeBaseDB().Nodes Failed err=[%s]", funName, err.Error())
			log4plus.Error(errString)
			common.SendError(c, header.JsonParseError, errString)
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
			errString := fmt.Sprintf("%s SingletonNodeBaseDB().Node Failed err=[%s]", funName, err.Error())
			log4plus.Error(errString)
			common.SendError(c, header.JsonParseError, errString)
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
			errString := fmt.Sprintf("%s SingletonNodeBaseDB().Workflows Failed err=[%s]", funName, err.Error())
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
			errString := fmt.Sprintf("%s SingletonNodeBaseDB().Workflows Failed err=[%s]", funName, err.Error())
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
			errString := fmt.Sprintf("%s SingletonNodeBaseDB().Workflows Failed err=[%s]", funName, err.Error())
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
			errString := fmt.Sprintf("%s SingletonNodeBaseDB().Workflows Failed err=[%s]", funName, err.Error())
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
		errString := fmt.Sprintf("%s SingletonNodeBaseDB().Task Failed err=[%s]", funName, err.Error())
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
		errString := fmt.Sprintf("%s SingletonNodeBaseDB().Models Failed err=[%s]", funName, err.Error())
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
		errString := fmt.Sprintf("%s SingletonNodeBaseDB().Models Failed err=[%s]", funName, err.Error())
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
		errString := fmt.Sprintf("%s SingletonNodeBaseDB().Models Failed err=[%s]", funName, err.Error())
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
		errString := fmt.Sprintf("%s SingletonNodeBaseDB().Models Failed err=[%s]", funName, err.Error())
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
		errString := fmt.Sprintf("%s SingletonNodeBaseDB().Models Failed title=[%s] err=[%s]", funName, request.Title, err.Error())
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
		errString := fmt.Sprintf("%s SingletonNodeBaseDB().Models Failed err=[%s]", funName, err.Error())
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
		errString := fmt.Sprintf("%s SingletonNodeBaseDB().Models Failed err=[%s]", funName, err.Error())
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
		errString := fmt.Sprintf("%s SingletonNodeBaseDB().TaskCount Failed err=[%s]", funName, err.Error())
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
		errString := fmt.Sprintf("%s SingletonNodeBaseDB().TaskCount Failed err=[%s]", funName, err.Error())
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
		errString := fmt.Sprintf("%s SingletonNodeBaseDB().CurNodes Failed err=[%s]", funName, err.Error())
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
		errString := fmt.Sprintf("%s SingletonNodeBaseDB().CurNodes Failed err=[%s]", funName, err.Error())
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
		errString := fmt.Sprintf("%s SingletonNodeBaseDB().CurNodes Failed err=[%s]", funName, err.Error())
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

func (w *Backend) Start(nodeGroup *gin.RouterGroup) {
	nodeGroup.POST("/gpus", w.gpus)
	nodeGroup.POST("/nodes", w.nodes)
	nodeGroup.POST("/nodeOnline", w.nodeOnline)
	nodeGroup.POST("/workflows", w.workflows)
	nodeGroup.POST("/nodeWorkflows", w.nodeWorkflows)
	nodeGroup.POST("/getTask", w.getTask)
	nodeGroup.GET("/getModels", w.getModels)
	nodeGroup.GET("/getCategories", w.getCategories)
	nodeGroup.POST("/getCategorie", w.getCategorie)
	nodeGroup.GET("/getTags", w.getTags)
	nodeGroup.POST("/getWorkflow", w.getWorkflow)
	nodeGroup.GET("/getUser", w.getUser)
	nodeGroup.GET("/getMyTask", w.getMyTask)
	nodeGroup.GET("/getTaskCount", w.getTaskCount)
	nodeGroup.GET("/getNodeCount", w.getNodeCount)
	nodeGroup.GET("/getCurNode", w.getCurNode)
	nodeGroup.POST("/register", w.register)
	nodeGroup.POST("/login", w.login)
}

func SingletonBackend() *Backend {
	if gBackend == nil {
		gBackend = &Backend{}
	}
	return gBackend
}
