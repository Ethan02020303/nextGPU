package db

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	log4plus "github.com/nextGPU/include/log4go"
	"github.com/nextGPU/ng-backend/configure"
	"strings"
)

type (
	GPU struct {
		GPUName            string  `json:"gpuName"`
		GPUFamilyID        string  `json:"gpuFamilyID"`
		Weight             float64 `json:"weight"`
		PoolSize           int64   `json:"poolSize"`
		VideoRAM           int64   `json:"videoRAM"`
		SchedulingPriority int64   `json:"schedulingPriority"`
	}
	Node struct {
		SystemUUID   string `json:"systemUUID"`
		Account      string `json:"account"`
		OrgName      string `json:"orgName"`
		NodeIP       string `json:"nodeIP"`
		OS           string `json:"os"`
		CPU          string `json:"cpu"`
		GPU          string `json:"gpu"`
		GPUDriver    string `json:"gpuDriver"`
		Memory       string `json:"memory"`
		State        int64  `json:"state"`
		NodeName     string `json:"nodeName"`
		RegisterTime string `json:"registerTime"`
		EnrollTime   string `json:"enrollTime"`
		HeartTime    string `json:"heartTime"`
		Versions     string `json:"versions"`
		MeasureTime  string `json:"measureTime"`
	}
	//
	Workflow struct {
		Deleted       bool   `json:"deleted"`
		Template      string `json:"template"`
		Configuration string `json:"configuration"`
		Description   string `json:"description"`
		Title         string `json:"title"`
	}
	//
	WorkSpace struct {
		WorkspaceID          string     `json:"workspaceID"`
		MajorCMD             string     `json:"majorCMD"`
		MinorCMD             string     `json:"minorCMD"`
		DeployCount          int64      `json:"deployCount"`
		RedundantCount       int64      `json:"redundantCount"`
		ComfyUIHTTP          string     `json:"comfyUIHTTP"`
		ComfyUIWS            string     `json:"comfyUIWS"`
		Deleted              bool       `json:"deleted"`
		MeasureTitle         string     `json:"measureTitle"`
		Measure              string     `json:"measure"`
		MeasureConfiguration string     `json:"measureConfiguration"`
		RunMode              int64      `json:"runMode"`
		MinGBVRAM            int64      `json:"minGBVRAM"`
		SavePath             string     `json:"savePath"`
		Workflows            []Workflow `json:"workflows"`
	}
	WorkspaceWorkflow struct {
		Workspace WorkSpace `json:"workspace"`
	}
	NodeWorkflowDB struct {
		SystemUUID   string              `json:"systemUUID"`
		NodeWorkflow []WorkspaceWorkflow `json:"nodeWorkflows"`
	}

	SubTask struct {
		SubID             string `json:"subID"`
		StartTime         string `json:"startTime"`
		EndTime           string `json:"endTime"`
		EstimateTime      int64  `json:"estimateTime"`
		Input             string `json:"input"`
		SystemUUID        string `json:"systemUUID"`
		State             int64  `json:"state"`
		Output            string `json:"output"`
		WorkspaceID       string `json:"workspaceID"`
		PublishTime       string `json:"publishTime"`
		Urls              string `json:"urls"`
		OSSDuration       string `json:"ossDuration"`
		OSSSize           int64  `json:"ossSize"`
		InferenceDuration int64  `json:"inferenceDuration"`
		CreateTime        string `json:"createTime"`
		RetryCount        int64  `json:"retryCount"`
		ErrorNodes        string `json:"errorNodes"`
		Delete            bool   `json:"delete"`
		FailureReason     string `json:"failureReason"`
	}
	MainTask struct {
		TaskID         string    `json:"taskID"`
		Session        string    `json:"session"`
		PublishTime    string    `json:"publishTime"`
		CompletionTime string    `json:"completionTime"`
		Title          string    `json:"title"`
		Input          string    `json:"input"`
		State          int64     `json:"state"`
		BillMode       bool      `json:"billMode"`
		Duration       int64     `json:"duration"`
		SubNum         int64     `json:"subNum"`
		WorkspaceID    string    `json:"workspaceID"`
		GeneratedData  string    `json:"generatedData"`
		StartTime      string    `json:"startTime"`
		Delete         bool      `json:"delete"`
		Request        string    `json:"request"`
		SubTasks       []SubTask `json:"subTasks"`
	}
	Model struct {
		WorkflowTitle string `json:"workflowTitle"`
		WorkflowCover string `json:"workflowCover"`
		Badge         int    `json:"badge"`
		AuthorName    string `json:"authorName"`
		DownloadCount int    `json:"downloadCount"`
		LikeCount     int    `json:"likeCount"`
		CommentCount  int    `json:"commentCount"`
		Model         string `json:"model"`
		Group         string `json:"group"`
	}
	Categorie struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	Tag struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	WorkflowBase struct {
		Deleted       bool   `json:"deleted"`
		Template      string `json:"template"`
		Configuration string `json:"configuration"`
		Description   string `json:"description"`
		Cover         string `json:"cover"`
		Badge         int    `json:"badge"`
		AuthorName    string `json:"authorName"`
		DownloadCount int    `json:"downloadCount"`
		LikeCount     int    `json:"likeCount"`
		CommentCount  int    `json:"commentCount"`
		Model         string `json:"model"`
		Title         string `json:"title"`
	}
	UseCount struct {
		Used  int `json:"used"`
		Total int `json:"total"`
	}
	UserBase struct {
		UserName  string   `json:"userName"`
		Level     int      `json:"level"`
		Credits   float64  `json:"credits"`
		Workflows UseCount `json:"workflows"`
		Storage   UseCount `json:"storage"`
		Images    int      `json:"images"`
		Face      string   `json:"face"`
		EMail     string   `json:"eMail"`
	}

	SelfSubTask struct {
		ID                int    `json:"id"`
		SubID             string `json:"subID"`
		StartTime         string `json:"startTime"`
		EndTime           string `json:"endTime"`
		SystemUUID        string `json:"systemUUID"`
		Url               string `json:"url"`
		OSSSize           int    `json:"ossSize"`
		InferenceDuration int    `json:"inferenceDuration"`
	}
	SelfTask struct {
		TaskID         string        `json:"taskID"`
		StartTime      string        `json:"startTime"`
		CompletionTime string        `json:"completionTime"`
		Title          string        `json:"title"`
		Input          string        `json:"input"`
		SubCount       int64         `json:"subCount"`
		Subs           []SelfSubTask `json:"subs"`
	}
	CurrentNode struct {
		SystemUUID string `json:"systemUUID"`
		NodeIP     string `json:"nodeIP"`
		GPU        string `json:"gpu"`
		GPUDriver  string `json:"gpuDriver"`
		Memory     int64  `json:"memory"`
		OnlineTime int64  `json:"onlineTime"`
		TasksCount int64  `json:"tasksCount"`
	}
)

type NodeDB struct {
	mysqlDb *MysqlManager
}

var gNodeDB *NodeDB

func (p *NodeDB) GPUs() (error, []GPU) {
	funName := "GPUs"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect ---->>>>", funName)
		log4plus.Error(errString)
		return errors.New(errString), []GPU{}
	}
	sql := fmt.Sprintf(`select IFNULL(a.f_id,-1), 
       	IFNULL(a.f_scheduling_priority, -1),
       	IFNULL(b.f_name, ''),
       	IFNULL(b.f_weight, 0),
    	IFNULL(b.f_pool_size, 0),
        IFNULL(b.f_gb_vram, 0)
       	from t_available_gpu as a, t_gpu_family as b where a.f_gpu_model = b.f_name;`)
	rows, err := p.mysqlDb.Mysqldb.Query(sql)
	if err != nil {
		errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, err.Error(), sql)
		log4plus.Error(errString)
		return errors.New(errString), []GPU{}
	}
	defer rows.Close()

	var gpus []GPU
	for rows.Next() {
		var tmpId int64
		var gpu GPU
		scanErr := rows.Scan(&tmpId, &gpu.SchedulingPriority, &gpu.GPUName, &gpu.Weight, &gpu.PoolSize, &gpu.VideoRAM)
		if scanErr != nil {
			log4plus.Error("%s Scan Error=[%s]", funName, scanErr.Error())
			continue
		}
		if tmpId != -1 {
			gpus = append(gpus, gpu)
		}
	}
	return nil, gpus
}

func (p *NodeDB) Node(systemUUID string) (error, Node) {
	funName := "Nodes"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect ---->>>>", funName)
		log4plus.Error(errString)
		return errors.New(errString), Node{}
	}
	sql := fmt.Sprintf(`select IFNULL(f_id,-1), 
       	IFNULL(f_system_uuid, ''),
       	IFNULL(f_account, ''),
       	IFNULL(f_org_name, ''),
    	IFNULL(f_node_ip, ''),
        IFNULL(f_os, ''),
        IFNULL(f_cpu, ''),
        IFNULL(f_gpu, ''),
        IFNULL(f_gpu_driver, ''),
        IFNULL(f_memory, -1),
        IFNULL(f_state, 0),
        IFNULL(f_node_name, ''),
        IFNULL(f_register_time, ''),
        IFNULL(f_enroll_time, ''),
        IFNULL(f_heart_time, ''),
        IFNULL(f_versions, ''),
         IFNULL(f_measure_time, '')
       	from t_nodes where f_system_uuid='%s';`, systemUUID)
	rows, err := p.mysqlDb.Mysqldb.Query(sql)
	if err != nil {
		errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, err.Error(), sql)
		log4plus.Error(errString)
		return errors.New(errString), Node{}
	}
	defer rows.Close()

	for rows.Next() {
		var tmpId int64
		var node Node
		scanErr := rows.Scan(&tmpId,
			&node.SystemUUID,
			&node.Account,
			&node.OrgName,
			&node.NodeIP,
			&node.OS,
			&node.CPU,
			&node.GPU,
			&node.GPUDriver,
			&node.Memory,
			&node.State,
			&node.NodeName,
			&node.RegisterTime,
			&node.EnrollTime,
			&node.HeartTime,
			&node.Versions,
			&node.MeasureTime)
		if scanErr != nil {
			log4plus.Error("%s Scan Error=[%s]", funName, scanErr.Error())
			continue
		}
		if tmpId != -1 {
			return nil, node
		}
	}
	return errors.New("not found systemUUID"), Node{}
}

func (p *NodeDB) Nodes() (error, []Node) {
	funName := "Nodes"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect ---->>>>", funName)
		log4plus.Error(errString)
		return errors.New(errString), []Node{}
	}
	sql := fmt.Sprintf(`select IFNULL(f_id,-1), 
       	IFNULL(f_system_uuid, ''),
       	IFNULL(f_account, ''),
       	IFNULL(f_org_name, ''),
    	IFNULL(f_node_ip, ''),
        IFNULL(f_os, ''),
        IFNULL(f_cpu, ''),
        IFNULL(f_gpu, ''),
        IFNULL(f_gpu_driver, ''),
        IFNULL(f_memory, -1),
        IFNULL(f_state, 0),
        IFNULL(f_node_name, ''),
        IFNULL(f_register_time, ''),
        IFNULL(f_enroll_time, ''),
        IFNULL(f_heart_time, ''),
        IFNULL(f_versions, ''),
         IFNULL(f_measure_time, '')
       	from t_nodes;`)
	rows, err := p.mysqlDb.Mysqldb.Query(sql)
	if err != nil {
		errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, err.Error(), sql)
		log4plus.Error(errString)
		return errors.New(errString), []Node{}
	}
	defer rows.Close()

	var nodes []Node
	for rows.Next() {
		var tmpId int64
		var node Node
		scanErr := rows.Scan(&tmpId,
			&node.SystemUUID,
			&node.Account,
			&node.OrgName,
			&node.NodeIP,
			&node.OS,
			&node.CPU,
			&node.GPU,
			&node.GPUDriver,
			&node.Memory,
			&node.State,
			&node.NodeName,
			&node.RegisterTime,
			&node.EnrollTime,
			&node.HeartTime,
			&node.Versions,
			&node.MeasureTime)
		if scanErr != nil {
			log4plus.Error("%s Scan Error=[%s]", funName, scanErr.Error())
			continue
		}
		if tmpId != -1 {
			nodes = append(nodes, node)
		}
	}
	return nil, nodes
}

func (p *NodeDB) Workflows() (error, []Workflow) {
	funName := "Workflows"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect ---->>>>", funName)
		log4plus.Error(errString)
		return errors.New(errString), []Workflow{}
	}
	sql := fmt.Sprintf(`select IFNULL(f_id,-1), 
       	IFNULL(f_deleted, false),
       	IFNULL(f_template, ''),
       	IFNULL(f_configuration, ''),
    	IFNULL(f_description, ''),
        IFNULL(f_title, '')
       	from t_workflows;`)
	rows, err := p.mysqlDb.Mysqldb.Query(sql)
	if err != nil {
		errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, err.Error(), sql)
		log4plus.Error(errString)
		return errors.New(errString), []Workflow{}
	}
	defer rows.Close()

	var workflowDB []Workflow
	for rows.Next() {
		var tmpId int64
		var workflow Workflow
		scanErr := rows.Scan(&tmpId,
			&workflow.Deleted,
			&workflow.Template,
			&workflow.Configuration,
			&workflow.Description,
			&workflow.Title)
		if scanErr != nil {
			log4plus.Error("%s Scan Error=[%s]", funName, scanErr.Error())
			continue
		}
		if tmpId != -1 {
			workflowDB = append(workflowDB, workflow)
		}
	}
	return nil, workflowDB
}

func (p *NodeDB) Workflow(title string) (error, Workflow) {
	funName := "Workflows"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect ---->>>>", funName)
		log4plus.Error(errString)
		return errors.New(errString), Workflow{}
	}
	sql := fmt.Sprintf(`select IFNULL(f_id,-1), 
       	IFNULL(f_deleted, false),
       	IFNULL(f_template, ''),
       	IFNULL(f_configuration, ''),
    	IFNULL(f_description, ''),
        IFNULL(f_title, '')
       	from t_workflows where f_title='%s';`, title)
	rows, err := p.mysqlDb.Mysqldb.Query(sql)
	if err != nil {
		errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, err.Error(), sql)
		log4plus.Error(errString)
		return errors.New(errString), Workflow{}
	}
	defer rows.Close()

	for rows.Next() {
		var tmpId int64
		var workflow Workflow
		scanErr := rows.Scan(&tmpId,
			&workflow.Deleted,
			&workflow.Template,
			&workflow.Configuration,
			&workflow.Description,
			&workflow.Title)
		if scanErr != nil {
			log4plus.Error("%s Scan Error=[%s]", funName, scanErr.Error())
			continue
		}
		if tmpId != -1 {
			return nil, workflow
		}
	}
	return errors.New("not found Workflow"), Workflow{}
}

func (p *NodeDB) NodeWorkflow(systemUUID string) (error, []WorkspaceWorkflow) {
	funName := "NodeWorkflow"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect ---->>>>", funName)
		log4plus.Error(errString)
		return errors.New(errString), []WorkspaceWorkflow{}
	}
	sql := fmt.Sprintf(`select IFNULL(f_id,-1), IFNULL(f_workspace_id, '') from t_node_workspaces where f_system_uuid='%s';`, systemUUID)
	rows, err := p.mysqlDb.Mysqldb.Query(sql)
	if err != nil {
		errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, err.Error(), sql)
		log4plus.Error(errString)
		return errors.New(errString), []WorkspaceWorkflow{}
	}
	defer rows.Close()

	var workSpaceIDs []string
	for rows.Next() {
		var tmpId int64
		var workSpaceID string
		scanErr := rows.Scan(&tmpId, &workSpaceID)
		if scanErr != nil {
			log4plus.Error("%s Scan Error=[%s]", funName, scanErr.Error())
			continue
		}
		workSpaceIDs = append(workSpaceIDs, workSpaceID)
	}
	var nodeWorkflowDB []WorkspaceWorkflow
	for _, workSpaceID := range workSpaceIDs {
		//workspace
		sql = fmt.Sprintf(`select IFNULL(f_id,-1), 
       IFNULL(f_major_cmd, ''),
       IFNULL(f_minor_cmd, ''),
       IFNULL(f_deploy_count, 0),
       IFNULL(f_redundant_count, 0),
       IFNULL(f_comfyui_http, ''),
       IFNULL(f_comfyui_ws, ''),
       IFNULL(f_deleted, false),
       IFNULL(f_measure_title, ''),
       IFNULL(f_measure, ''),
       IFNULL(f_measure_configuration, ''),
       IFNULL(f_run_mode, 0),
       IFNULL(f_min_gb_vram, 0),
       IFNULL(f_save_path, '')
       from t_workspaces where f_workspace_id='%s';`, workSpaceID)
		rowSelect, errSelect := p.mysqlDb.Mysqldb.Query(sql)
		if errSelect != nil {
			errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, errSelect.Error(), sql)
			log4plus.Error(errString)
			return errors.New(errString), []WorkspaceWorkflow{}
		}
		defer rowSelect.Close()

		var workspaceWorkflow WorkspaceWorkflow
		for rowSelect.Next() {
			var tmpId int64
			var workSpace WorkSpace
			scanErr := rowSelect.Scan(&tmpId, &workSpace.MajorCMD, &workSpace.MinorCMD, &workSpace.DeployCount,
				&workSpace.RedundantCount, &workSpace.ComfyUIHTTP, &workSpace.ComfyUIWS, &workSpace.Deleted,
				&workSpace.MeasureTitle, &workSpace.Measure, &workSpace.MeasureConfiguration, &workSpace.RunMode,
				&workSpace.MinGBVRAM, &workSpace.SavePath)
			if scanErr != nil {
				log4plus.Error("%s Scan Error=[%s]", funName, scanErr.Error())
				continue
			}
			workSpace.WorkspaceID = workSpaceID

			//workflow
			sql = fmt.Sprintf(`select IFNULL(b.f_id,-1), 
			   IFNULL(b.f_deleted, false),
			   IFNULL(b.f_template, ''),
			   IFNULL(b.f_configuration, ''),
			   IFNULL(b.f_description, ''),
			   IFNULL(b.f_title, '')
			   from t_workspace_workflow as a, t_workflows as b where a.f_title=b.f_title and a.f_workspace_id='%s';`, workSpaceID)
			rowSelect2, errSelect2 := p.mysqlDb.Mysqldb.Query(sql)
			if errSelect2 != nil {
				errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, errSelect2.Error(), sql)
				log4plus.Error(errString)
				return errors.New(errString), []WorkspaceWorkflow{}
			}
			defer rowSelect2.Close()

			for rowSelect2.Next() {
				var tmpId int64
				var workflow Workflow
				scanErrSelect2 := rowSelect2.Scan(&tmpId, &workflow.Deleted, &workflow.Template, &workflow.Configuration,
					&workflow.Description, &workflow.Title)
				if scanErrSelect2 != nil {
					log4plus.Error("%s Scan Error=[%s]", funName, scanErrSelect2.Error())
					continue
				}
				workSpace.Workflows = append(workSpace.Workflows, workflow)
			}
			workspaceWorkflow.Workspace = workSpace
		}
		nodeWorkflowDB = append(nodeWorkflowDB, workspaceWorkflow)
	}
	return nil, nodeWorkflowDB
}

func (p *NodeDB) NodeWorkflows() (error, []NodeWorkflowDB) {
	funName := "NodeWorkflows"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect ---->>>>", funName)
		log4plus.Error(errString)
		return errors.New(errString), []NodeWorkflowDB{}
	}
	sql := fmt.Sprintf(`select IFNULL(f_id,-1), IFNULL(f_workspace_id, ''), IFNULL(f_system_uuid, '') from t_node_workspaces;`)
	rows, err := p.mysqlDb.Mysqldb.Query(sql)
	if err != nil {
		errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, err.Error(), sql)
		log4plus.Error(errString)
		return errors.New(errString), []NodeWorkflowDB{}
	}
	defer rows.Close()

	var nodeWorkflowDBs []NodeWorkflowDB
	for rows.Next() {
		var tmpId int64
		var workSpaceID, systemUUID string
		scanErr := rows.Scan(&tmpId, &workSpaceID, &systemUUID)
		if scanErr != nil {
			log4plus.Error("%s Scan Error=[%s]", funName, scanErr.Error())
			continue
		}
		var nodeWorkflowDB NodeWorkflowDB
		nodeWorkflowDB.SystemUUID = systemUUID
		var workspaceWorkflow WorkspaceWorkflow
		sql = fmt.Sprintf(`select IFNULL(f_id,-1), 
			   IFNULL(f_major_cmd, ''),
			   IFNULL(f_minor_cmd, ''),
			   IFNULL(f_deploy_count, 0),
			   IFNULL(f_redundant_count, 0),
			   IFNULL(f_comfyui_http, ''),
			   IFNULL(f_comfyui_ws, ''),
			   IFNULL(f_deleted, false),
			   IFNULL(f_measure_title, ''),
			   IFNULL(f_measure, ''),
			   IFNULL(f_measure_configuration, ''),
			   IFNULL(f_run_mode, 0),
			   IFNULL(f_min_gb_vram, 0),
			   IFNULL(f_save_path, '')
       			from t_workspaces where f_workspace_id='%s';`, workSpaceID)
		rowSelect, errSelect := p.mysqlDb.Mysqldb.Query(sql)
		if errSelect != nil {
			errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, errSelect.Error(), sql)
			log4plus.Error(errString)
			return errors.New(errString), []NodeWorkflowDB{}
		}
		defer rowSelect.Close()

		for rowSelect.Next() {
			var tmpId2 int64
			var workSpace WorkSpace
			workSpace.WorkspaceID = workSpaceID
			scanSelectErr := rowSelect.Scan(&tmpId2, &workSpace.MajorCMD, &workSpace.MinorCMD, &workSpace.DeployCount,
				&workSpace.RedundantCount, &workSpace.ComfyUIHTTP, &workSpace.ComfyUIWS, &workSpace.Deleted,
				&workSpace.MeasureTitle, &workSpace.Measure, &workSpace.MeasureConfiguration, &workSpace.RunMode,
				&workSpace.MinGBVRAM, &workSpace.SavePath)
			if scanSelectErr != nil {
				log4plus.Error("%s Scan Error=[%s]", funName, scanSelectErr.Error())
				continue
			}

			//workflow
			sql = fmt.Sprintf(`select IFNULL(b.f_id,-1), 
			   IFNULL(b.f_deleted, false),
			   IFNULL(b.f_template, ''),
			   IFNULL(b.f_configuration, ''),
			   IFNULL(b.f_description, ''),
			   IFNULL(b.f_title, '')
			   from t_workspace_workflow as a, t_workflows as b where a.f_title=b.f_title and a.f_workspace_id='%s';`, workSpaceID)
			rowSelect3, errSelect3 := p.mysqlDb.Mysqldb.Query(sql)
			if errSelect3 != nil {
				errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, errSelect3.Error(), sql)
				log4plus.Error(errString)
				return errors.New(errString), []NodeWorkflowDB{}
			}
			defer rowSelect3.Close()

			for rowSelect3.Next() {
				var tmpId3 int64
				var workflow Workflow
				scanErr3 := rowSelect3.Scan(&tmpId3, &workflow.Deleted, &workflow.Template, &workflow.Configuration, &workflow.Description, &workflow.Title)
				if scanErr3 != nil {
					log4plus.Error("%s Scan Error=[%s]", funName, scanErr3.Error())
					continue
				}
				workSpace.Workflows = append(workSpace.Workflows, workflow)
			}
			workspaceWorkflow.Workspace = workSpace
		}
		nodeWorkflowDB.NodeWorkflow = append(nodeWorkflowDB.NodeWorkflow, workspaceWorkflow)
		nodeWorkflowDBs = append(nodeWorkflowDBs, nodeWorkflowDB)
	}
	return nil, nodeWorkflowDBs
}

func marshalIndent(buffer string) string {
	var out bytes.Buffer
	err := json.Indent(&out, []byte(buffer), "", "    ") // 4 空格缩进
	if err != nil {
		return buffer
	}
	return out.String()
}

func (p *NodeDB) Task(taskID string) (error, MainTask) {
	funName := "Task"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect ---->>>>", funName)
		log4plus.Error(errString)
		return errors.New(errString), MainTask{}
	}
	sql := fmt.Sprintf(`select IFNULL(f_id,-1), IFNULL(f_task_id, ''),IFNULL(f_session, ''),IFNULL(f_publish_time, ''), IFNULL(f_completion_time, ''),
       IFNULL(f_title, ''),IFNULL(f_input, ''),IFNULL(f_state, 0),IFNULL(f_bill_mode, 0), IFNULL(f_duration, 0),
       IFNULL(f_sub_num, 0),IFNULL(f_workspace_id, ''),IFNULL(f_generated_data, ''),IFNULL(f_start_time, ''),IFNULL(f_delete, 0),
		IFNULL(f_request, '') from t_jobs where f_task_id='%s';`, taskID)
	rows, err := p.mysqlDb.Mysqldb.Query(sql)
	if err != nil {
		errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, err.Error(), sql)
		log4plus.Error(errString)
		return errors.New(errString), MainTask{}
	}
	defer rows.Close()

	var mainTask MainTask
	for rows.Next() {
		var tmpId int64
		scanErr := rows.Scan(&tmpId, &mainTask.TaskID, &mainTask.Session, &mainTask.PublishTime, &mainTask.CompletionTime,
			&mainTask.Title, &mainTask.Input, &mainTask.State, &mainTask.BillMode, &mainTask.Duration,
			&mainTask.SubNum, &mainTask.WorkspaceID, &mainTask.GeneratedData, &mainTask.StartTime, &mainTask.Delete,
			&mainTask.Request)
		if scanErr != nil {
			log4plus.Error("%s Scan Error=[%s]", funName, scanErr.Error())
			continue
		}
		//mainTask.Input = marshalIndent(mainTask.Input)
		//mainTask.Request = marshalIndent(mainTask.Request)
		//mainTask.GeneratedData = marshalIndent(mainTask.GeneratedData)

		//subTask
		sql = fmt.Sprintf(`select IFNULL(f_id, 0),
       IFNULL(f_sub_id, ''),
       IFNULL(f_start_time, ''),
       IFNULL(f_end_time, ''),
       IFNULL(f_estimate_time, 0),
       IFNULL(f_input, ''),
       IFNULL(f_system_uuid, ''),
       IFNULL(f_state, 0),
       IFNULL(f_output, ''),
       IFNULL(f_workspace_id, ''),
       IFNULL(f_publish_time, ''),
       IFNULL(f_urls, ''),
       IFNULL(f_oss_duration, ''),
       IFNULL(f_oss_size, ''),
       IFNULL(f_inference_duration, 0),
       IFNULL(f_create_time, ''),
       IFNULL(f_retry_count, 0),
       IFNULL(f_error_nodes, ''),
       IFNULL(f_delete, 0),
       IFNULL(f_failure_reason, '') from t_sub_jobs where f_task_id='%s';`, taskID)
		rowSelect2, errSelect2 := p.mysqlDb.Mysqldb.Query(sql)
		if errSelect2 != nil {
			errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, errSelect2.Error(), sql)
			log4plus.Error(errString)
			return errors.New(errString), MainTask{}
		}
		defer rowSelect2.Close()

		for rowSelect2.Next() {
			var tmpId2 int64
			var subTask SubTask
			scanErrSelect2 := rowSelect2.Scan(&tmpId2, &subTask.SubID, &subTask.StartTime, &subTask.EndTime, &subTask.EstimateTime,
				&subTask.Input, &subTask.SystemUUID, &subTask.State, &subTask.Output, &subTask.WorkspaceID, &subTask.PublishTime,
				&subTask.Urls, &subTask.OSSDuration, &subTask.OSSSize, &subTask.InferenceDuration, &subTask.CreateTime,
				&subTask.RetryCount, &subTask.ErrorNodes, &subTask.Delete, &subTask.FailureReason)
			if scanErrSelect2 != nil {
				log4plus.Error("%s Scan Error=[%s]", funName, scanErrSelect2.Error())
				continue
			}
			mainTask.SubTasks = append(mainTask.SubTasks, subTask)
		}
	}
	return nil, mainTask
}

func (p *NodeDB) Models() (error, []Model) {
	funName := "Models"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect", funName)
		log4plus.Error(errString)
		return errors.New(errString), []Model{}
	}
	sql := fmt.Sprintf(`select IFNULL(f_id,-1), 
       IFNULL(f_workflow_title,''), 
       IFNULL(f_workflow_cover,''), 
       IFNULL(f_badge,0), 
       IFNULL(f_author_name,''), 
       IFNULL(f_download_count,0), 
       IFNULL(f_like_count,0), 
       IFNULL(f_comment_count, 0),  
       IFNULL(f_model, ''), 
       IFNULL(f_group,'') from t_workflow_base order by f_id desc;`)
	rows, err := p.mysqlDb.Mysqldb.Query(sql)
	if err != nil {
		errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, err.Error(), sql)
		log4plus.Error(errString)
		return err, []Model{}
	}
	defer rows.Close()

	var models []Model
	for rows.Next() {
		model := Model{}
		var tmpId int
		scanErr := rows.Scan(&tmpId,
			&model.WorkflowTitle,
			&model.WorkflowCover,
			&model.Badge,
			&model.AuthorName,
			&model.DownloadCount,
			&model.LikeCount,
			&model.CommentCount,
			&model.Model,
			&model.Group)
		if scanErr != nil {
			log4plus.Error("%s Scan Error=[%s]", funName, scanErr.Error())
			continue
		}
		models = append(models, model)
	}
	return nil, models
}

func (p *NodeDB) Categories() (error, []Categorie) {
	funName := "Categories"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect", funName)
		log4plus.Error(errString)
		return errors.New(errString), []Categorie{}
	}
	sql := fmt.Sprintf(`select DISTINCT f_model from t_workflow_base;`)
	rows, err := p.mysqlDb.Mysqldb.Query(sql)
	if err != nil {
		errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, err.Error(), sql)
		log4plus.Error(errString)
		return err, []Categorie{}
	}
	defer rows.Close()

	var categories []Categorie
	var index int = 0
	for rows.Next() {
		var categorie Categorie
		scanErr := rows.Scan(&categorie.Name)
		if scanErr != nil {
			log4plus.Error("%s Scan Error=[%s]", funName, scanErr.Error())
			continue
		}
		categorie.ID = index
		index++
		categories = append(categories, categorie)
	}
	return nil, categories
}

func (p *NodeDB) Categorie(categorieName string) (error, []Model) {
	funName := "Categorie"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect", funName)
		log4plus.Error(errString)
		return errors.New(errString), []Model{}
	}
	sql := fmt.Sprintf(`select IFNULL(f_id,-1), 
       IFNULL(f_workflow_title,''), 
       IFNULL(f_workflow_cover,''), 
       IFNULL(f_badge,0), 
       IFNULL(f_author_name,''), 
       IFNULL(f_download_count,0), 
       IFNULL(f_like_count,0), 
       IFNULL(f_comment_count, 0),  
       IFNULL(f_model, ''), 
       IFNULL(f_group,'') from t_workflow_base where f_model='%s' order by f_id desc;`, categorieName)
	rows, err := p.mysqlDb.Mysqldb.Query(sql)
	if err != nil {
		errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, err.Error(), sql)
		log4plus.Error(errString)
		return err, []Model{}
	}
	defer rows.Close()

	var models []Model
	for rows.Next() {
		model := Model{}
		var tmpId int
		scanErr := rows.Scan(&tmpId,
			&model.WorkflowTitle,
			&model.WorkflowCover,
			&model.Badge,
			&model.AuthorName,
			&model.DownloadCount,
			&model.LikeCount,
			&model.CommentCount,
			&model.Model,
			&model.Group)
		if scanErr != nil {
			log4plus.Error("%s Scan Error=[%s]", funName, scanErr.Error())
			continue
		}
		models = append(models, model)
	}
	return nil, models
}

func (p *NodeDB) Tags() (error, []Tag) {
	funName := "Tags"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect", funName)
		log4plus.Error(errString)
		return errors.New(errString), []Tag{}
	}
	sql := fmt.Sprintf(`select DISTINCT f_group from t_workflow_base;`)
	rows, err := p.mysqlDb.Mysqldb.Query(sql)
	if err != nil {
		errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, err.Error(), sql)
		log4plus.Error(errString)
		return err, []Tag{}
	}
	defer rows.Close()

	var tags []Tag
	var index int = 0
	for rows.Next() {
		var tag Tag
		scanErr := rows.Scan(&tag.Name)
		if scanErr != nil {
			log4plus.Error("%s Scan Error=[%s]", funName, scanErr.Error())
			continue
		}
		tag.ID = index
		index++
		tags = append(tags, tag)
	}
	return nil, tags
}

func (p *NodeDB) WorkflowBase(title string) (error, WorkflowBase) {
	funName := "WorkflowBase"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect ---->>>>", funName)
		log4plus.Error(errString)
		return errors.New(errString), WorkflowBase{}
	}
	sql := fmt.Sprintf(`select IFNULL(a.f_id,-1), 
       	IFNULL(a.f_deleted, false),
       	IFNULL(a.f_template, ''),
       	IFNULL(a.f_configuration, ''),
    	IFNULL(a.f_description, ''),    	
    	IFNULL(b.f_workflow_cover, ''),
    	IFNULL(b.f_badge, 0),
    	IFNULL(b.f_author_name, ''),
    	IFNULL(b.f_download_count, 0),
    	IFNULL(b.f_like_count, 0),
    	IFNULL(b.f_comment_count, 0),
    	IFNULL(b.f_model, ''),
        IFNULL(a.f_title, '')
       	from t_workflows as a, t_workflow_base as b where a.f_title=b.f_workflow_title and a.f_title='%s';`, title)
	log4plus.Info(sql)
	rows, err := p.mysqlDb.Mysqldb.Query(sql)
	if err != nil {
		errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, err.Error(), sql)
		log4plus.Error(errString)
		return errors.New(errString), WorkflowBase{}
	}
	defer rows.Close()

	for rows.Next() {
		var tmpId int64
		var workflow WorkflowBase
		scanErr := rows.Scan(&tmpId,
			&workflow.Deleted,
			&workflow.Template,
			&workflow.Configuration,
			&workflow.Description,
			&workflow.Cover,
			&workflow.Badge,
			&workflow.AuthorName,
			&workflow.DownloadCount,
			&workflow.LikeCount,
			&workflow.CommentCount,
			&workflow.Model,
			&workflow.Title)
		if scanErr != nil {
			log4plus.Error("%s Scan Error=[%s]", funName, scanErr.Error())
			continue
		}
		if tmpId != -1 {
			return nil, workflow
		}
	}
	return errors.New("not found Workflow"), WorkflowBase{}
}

func (p *NodeDB) UserBase(userName string) (error, UserBase) {
	funName := "WorkflowBase"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect ---->>>>", funName)
		log4plus.Error(errString)
		return errors.New(errString), UserBase{}
	}
	sql := fmt.Sprintf(`select IFNULL(f_id,-1), 
       	IFNULL(f_user_name, ''),
       	IFNULL(f_cost, 0),
       	IFNULL(f_level, 0),
    	IFNULL(f_face, '')
       	from t_users where f_user_name='%s';`, userName)
	rows, err := p.mysqlDb.Mysqldb.Query(sql)
	if err != nil {
		errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, err.Error(), sql)
		log4plus.Error(errString)
		return errors.New(errString), UserBase{}
	}
	defer rows.Close()

	for rows.Next() {
		var tmpId int64
		var userBase UserBase
		scanErr := rows.Scan(&tmpId,
			&userBase.UserName,
			&userBase.Credits,
			&userBase.Level,
			&userBase.Face)
		if scanErr != nil {
			log4plus.Error("%s Scan Error=[%s]", funName, scanErr.Error())
			continue
		}
		if tmpId != -1 {
			userBase.Workflows.Used = 16
			userBase.Workflows.Total = 20
			userBase.Storage.Used = 1024 * 1024 * 34
			userBase.Storage.Total = 2048 * 1024 * 1024
			return nil, userBase
		}
	}
	return errors.New("not found Workflow"), UserBase{}
}

func (p *NodeDB) MyTask(userName string) (error, []SelfTask) {
	funName := "MyTask"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect", funName)
		log4plus.Error(errString)
		return errors.New(errString), []SelfTask{}
	}
	sql := fmt.Sprintf(`select IFNULL(f_id,-1), IFNULL(f_task_id,'') from t_user_job order by f_id desc;`)
	rows, err := p.mysqlDb.Mysqldb.Query(sql)
	if err != nil {
		errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, err.Error(), sql)
		log4plus.Error(errString)
		return err, []SelfTask{}
	}
	defer rows.Close()

	var tasks []string
	for rows.Next() {
		taskID := ""
		var tmpId int
		scanErr := rows.Scan(&tmpId, &taskID)
		if scanErr != nil {
			log4plus.Error("%s Scan Error=[%s]", funName, scanErr.Error())
			continue
		}
		tasks = append(tasks, taskID)
	}

	var mainTask []SelfTask
	for _, taskID := range tasks {
		//主任务
		sql = fmt.Sprintf(`select IFNULL(f_id,-1), 
       IFNULL(f_task_id, ''), 
       IFNULL(f_start_time, ''), 
       IFNULL(f_completion_time, ''), 
       IFNULL(f_title, ''),
       IFNULL(f_input, ''),
       IFNULL(f_sub_num, 0) from t_jobs where f_task_id='%s' order by f_id desc;`, taskID)
		rows, err = p.mysqlDb.Mysqldb.Query(sql)
		if err != nil {
			errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, err.Error(), sql)
			log4plus.Error(errString)
			return err, []SelfTask{}
		}
		defer rows.Close()
		var task SelfTask
		for rows.Next() {
			var tmpId int
			scanErr := rows.Scan(&tmpId, &task.TaskID, &task.StartTime, &task.CompletionTime, &task.Title, &task.Input, &task.SubCount)
			if scanErr != nil {
				log4plus.Error("%s Scan Error=[%s]", funName, scanErr.Error())
				continue
			}
		}

		//子任务
		sql = fmt.Sprintf(`select IFNULL(f_id,-1), 
       IFNULL(f_sub_id, ''), 
       IFNULL(f_start_time, ''), 
       IFNULL(f_end_time, ''), 
       IFNULL(f_system_uuid, ''),
       IFNULL(f_urls, ''),
       IFNULL(f_oss_size, 0),
       IFNULL(f_inference_duration, 0) from t_sub_jobs where f_task_id='%s' order by f_id desc;`, taskID)
		rows, err = p.mysqlDb.Mysqldb.Query(sql)
		if err != nil {
			errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, err.Error(), sql)
			log4plus.Error(errString)
			return err, []SelfTask{}
		}
		defer rows.Close()
		var subTask SelfSubTask
		index := 0
		for rows.Next() {
			var tmpId int
			scanErr := rows.Scan(&tmpId, &subTask.SubID, &subTask.StartTime, &subTask.EndTime, &subTask.SystemUUID,
				&subTask.Url, &subTask.OSSSize, &subTask.InferenceDuration)
			if scanErr != nil {
				log4plus.Error("%s Scan Error=[%s]", funName, scanErr.Error())
				continue
			}
			subTask.ID = index
			index++
			task.Subs = append(task.Subs, subTask)
		}
		mainTask = append(mainTask, task)
	}
	return nil, mainTask
}

func (p *NodeDB) TaskCount() (error, int64) {
	funName := "TaskCount"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect", funName)
		log4plus.Error(errString)
		return errors.New(errString), 0
	}
	sql := fmt.Sprintf(`select Count(f_id) from t_sub_jobs;`)
	rows, err := p.mysqlDb.Mysqldb.Query(sql)
	if err != nil {
		errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, err.Error(), sql)
		log4plus.Error(errString)
		return err, 0
	}
	defer rows.Close()

	for rows.Next() {
		var taskCount int64
		scanErr := rows.Scan(&taskCount)
		if scanErr != nil {
			log4plus.Error("%s Scan Error=[%s]", funName, scanErr.Error())
			continue
		}
		return nil, taskCount
	}
	return errors.New("not found From DB"), 0
}

func (p *NodeDB) NodeCount() (error, int64) {
	funName := "NodeCount"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect", funName)
		log4plus.Error(errString)
		return errors.New(errString), 0
	}
	sql := fmt.Sprintf(`select Count(f_id) from t_nodes;`)
	rows, err := p.mysqlDb.Mysqldb.Query(sql)
	if err != nil {
		errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, err.Error(), sql)
		log4plus.Error(errString)
		return err, 0
	}
	defer rows.Close()

	for rows.Next() {
		var nodeCount int64
		scanErr := rows.Scan(&nodeCount)
		if scanErr != nil {
			log4plus.Error("%s Scan Error=[%s]", funName, scanErr.Error())
			continue
		}
		return nil, nodeCount
	}
	return errors.New("not found From DB"), 0
}

func getGPU(gpu string) string {
	// Step 1: 定位起始点 (跳过"GPU 0: ")
	startIdx := strings.Index(gpu, ": ") + 2
	// Step 2: 定位结束点 (查找"(UUID:")
	endIdx := strings.Index(gpu, " (UUID:")
	if startIdx >= 0 && endIdx > startIdx {
		gpuInfo := gpu[startIdx:endIdx]
		return gpuInfo
	}
	return gpu
}

func (p *NodeDB) CurNodes() (error, []CurrentNode) {
	funName := "CurNodes"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect ---->>>>", funName)
		log4plus.Error(errString)
		return errors.New(errString), []CurrentNode{}
	}
	sql := fmt.Sprintf(`select IFNULL(f_id,-1), 
       	IFNULL(f_system_uuid, ''),
    	IFNULL(f_node_ip, ''),
        IFNULL(f_gpu, ''),
        IFNULL(f_gpu_driver, ''),
        IFNULL(f_memory, -1) from t_nodes order by f_heart_time desc limit 5;`)
	rows, err := p.mysqlDb.Mysqldb.Query(sql)
	if err != nil {
		errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, err.Error(), sql)
		log4plus.Error(errString)
		return errors.New(errString), []CurrentNode{}
	}
	defer rows.Close()

	var nodes []CurrentNode
	for rows.Next() {
		var tmpId, tmpMemory int64
		var node CurrentNode
		scanErr := rows.Scan(&tmpId,
			&node.SystemUUID,
			&node.NodeIP,
			&node.GPU,
			&node.GPUDriver,
			&tmpMemory)
		if scanErr != nil {
			log4plus.Error("%s Scan Error=[%s]", funName, scanErr.Error())
			continue
		}
		//GPU Info
		node.GPU = getGPU(node.GPU)
		node.Memory = tmpMemory / (1024 * 1024 * 1024)

		//得到当前在线时长
		sql = fmt.Sprintf(`select sum(f_hour) as hourNum  from t_node_online_time;`)
		rowsTime, errTime := p.mysqlDb.Mysqldb.Query(sql)
		if errTime != nil {
			errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, errTime.Error(), sql)
			log4plus.Error(errString)
			return errors.New(errString), []CurrentNode{}
		}
		defer rowsTime.Close()
		for rowsTime.Next() {
			scanTimeErr := rowsTime.Scan(&node.OnlineTime)
			if scanTimeErr != nil {
				log4plus.Error("%s Scan Error=[%s]", funName, scanTimeErr.Error())
				continue
			}
		}

		//得到当前在线时长
		sql = fmt.Sprintf(`select count(f_id) as taskNum  from t_sub_jobs where f_system_uuid='%s';`, node.SystemUUID)
		rowsTaskCount, errTaskCount := p.mysqlDb.Mysqldb.Query(sql)
		if errTaskCount != nil {
			errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, errTaskCount.Error(), sql)
			log4plus.Error(errString)
			return errors.New(errString), []CurrentNode{}
		}
		defer rowsTaskCount.Close()
		for rowsTaskCount.Next() {
			scanTaskCountErr := rowsTaskCount.Scan(&node.TasksCount)
			if scanTaskCountErr != nil {
				log4plus.Error("%s Scan Error=[%s]", funName, scanTaskCountErr.Error())
				continue
			}
		}
		if node.GPU != "" && node.GPUDriver != "" {
			nodes = append(nodes, node)
		}
	}
	return nil, nodes
}

func (p *NodeDB) Register(userName, eMail, password string) (error, bool, string) {
	funName := "Register"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect ---->>>>", funName)
		log4plus.Error(errString)
		return errors.New(errString), false, ""
	}
	sql := fmt.Sprintf(`select IFNULL(f_id,-1) from t_users where f_user_name='%s';`, userName)
	rows, err := p.mysqlDb.Mysqldb.Query(sql)
	if err != nil {
		errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, err.Error(), sql)
		log4plus.Error(errString)
		return errors.New(errString), false, ""
	}
	defer rows.Close()

	for rows.Next() {
		var tmpId int64
		scanErr := rows.Scan(&tmpId)
		if scanErr != nil {
			log4plus.Error("%s Scan Error=[%s]", funName, scanErr.Error())
			continue
		}
		if tmpId != -1 {
			return nil, false, "user already exists"
		}
	}
	sql = fmt.Sprintf(`insert into t_users (f_user_name, f_email, f_password, f_cost, f_level, f_face) values 
                                                        ('%s', '%s', '%s', 0, 0, 'http://54.238.152.179/images/mini_logo.png');`, userName, eMail, password)
	_, err = p.mysqlDb.Mysqldb.Exec(sql)
	if err != nil {
		log4plus.Error("%s insert Failed err=[%s] userName=[%s] eMail=[%s] password=[%s]", funName, err.Error(), userName, eMail, password)
		return err, false, ""
	}
	return nil, true, ""
}

func (p *NodeDB) Login(userName, password string) (error, UserBase) {
	funName := "Login"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect ---->>>>", funName)
		log4plus.Error(errString)
		return errors.New(errString), UserBase{}
	}
	sql := fmt.Sprintf(`select IFNULL(f_id,-1), 
       	IFNULL(f_user_name, ''),
       	IFNULL(f_cost, 0),
       	IFNULL(f_level, 0),
    	IFNULL(f_face, ''),
    	IFNULL(f_email, ''),
    	IFNULL(f_password, '') from t_users where f_user_name='%s';`, userName)
	rows, err := p.mysqlDb.Mysqldb.Query(sql)
	if err != nil {
		errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, err.Error(), sql)
		log4plus.Error(errString)
		return errors.New(errString), UserBase{}
	}
	defer rows.Close()

	for rows.Next() {
		var tmpId int64
		var tmpPassword string
		var userBase UserBase
		scanErr := rows.Scan(&tmpId,
			&userBase.UserName,
			&userBase.Credits,
			&userBase.Level,
			&userBase.Face,
			&userBase.EMail,
			&tmpPassword)
		if scanErr != nil {
			log4plus.Error("%s Scan Error=[%s]", funName, scanErr.Error())
			continue
		}
		if !strings.EqualFold(tmpPassword, password) {
			return errors.New("The password entered by the user is incorrect"), UserBase{}
		}
		if tmpId != -1 {
			userBase.Workflows.Used = 16
			userBase.Workflows.Total = 20
			userBase.Storage.Used = 1024 * 1024 * 34
			userBase.Storage.Total = 2048 * 1024 * 1024
			return nil, userBase
		}
	}
	return errors.New("not found Workflow"), UserBase{}
}

func SingletonNodeBaseDB() *NodeDB {
	funName := "SingletonNodeBaseDB"
	if gNodeDB == nil {
		log4plus.Info("%s ---->>>>", funName)
		gNodeDB = &NodeDB{}
		if gNodeDB.mysqlDb = NewMysql(configure.SingletonConfigure().Mysql.MysqlIp,
			configure.SingletonConfigure().Mysql.MysqlPort,
			configure.SingletonConfigure().Mysql.MysqlDBName,
			configure.SingletonConfigure().Mysql.MysqlDBCharset,
			configure.SingletonConfigure().Mysql.UserName,
			configure.SingletonConfigure().Mysql.Password); gNodeDB.mysqlDb == nil {
			log4plus.Error("%s NewMysql Failed", funName)
			return nil
		}
	}
	return gNodeDB
}
