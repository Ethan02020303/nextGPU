package db

import (
	"encoding/json"
	"errors"
	"fmt"
	log4plus "github.com/nextGPU/include/log4go"
	"github.com/nextGPU/ng-cloud/configure"
	"gopkg.in/yaml.v3"
	"regexp"
	"strings"
)

type AvailableMode int

const (
	GPU AvailableMode = iota
	Memory
	OS
	CPU
	GPUDriver
)

type NodeStatus int

const (
	Unavailable NodeStatus = iota
	Disconnect
	Enrolling
	Enrolled
	Connecting
	Connected
	Measuring
	MeasurementFailed
	MeasurementComplete
)

type DockerDeleted int

const (
	UnDelete DockerDeleted = iota
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
	Passed
	UnPassed
	DockerStatusMax
)

type RunMode int

const (
	DockerRun RunMode = iota
	ComposeMode
)

type (
	// 定义 docker-compose 顶层结构
	DockerCompose struct {
		Version  string             `yaml:"version"`
		Services map[string]Service `yaml:"services"`
		Networks map[string]Network `yaml:"networks,omitempty"`
		Volumes  map[string]Volume  `yaml:"volumes,omitempty"`
	}
	// 服务定义（覆盖常见字段）
	Service struct {
		Build         BuildConfig  `yaml:"build,omitempty"`
		Image         string       `yaml:"image,omitempty"`
		ContainerName string       `yaml:"container_name,omitempty"`
		Ports         []string     `yaml:"ports,omitempty"`       // 格式 "host:container"
		Volumes       []string     `yaml:"volumes,omitempty"`     // 格式 "./local:/container"
		Environment   []string     `yaml:"environment,omitempty"` // 环境变量列表
		DependsOn     []string     `yaml:"depends_on,omitempty"`  // 依赖服务
		Networks      []string     `yaml:"networks,omitempty"`    // 关联网络
		Deploy        DeployConfig `yaml:"deploy,omitempty"`      // 部署配置
		Command       interface{}  `yaml:"command,omitempty"`     // 字符串或数组
	}
	// 构建配置（支持本地路径）
	BuildConfig struct {
		Context    string `yaml:"context,omitempty"`
		Dockerfile string `yaml:"dockerfile,omitempty"`
	}
	// 部署资源配置（覆盖资源限制）
	DeployConfig struct {
		Resources struct {
			Reservations struct {
				Devices []struct {
					Driver       string   `yaml:"driver"`
					Count        int      `yaml:"count"`
					Capabilities []string `yaml:"capabilities"`
				} `yaml:"devices,omitempty"`
			} `yaml:"reservations,omitempty"`
		} `yaml:"resources,omitempty"`
	}
	// 网络配置（支持驱动类型）
	Network struct {
		Driver string `yaml:"driver,omitempty"` // bridge/overlay/host
	}
	// 数据卷配置（支持命名卷）
	Volume struct {
		Driver string `yaml:"driver,omitempty"`
	}
)

type (
	MonitorConfiguration struct {
		MonitorType string `json:"monitor_type"`
		Monitor     string `json:"monitor"`
	}
	MonitorConfigurations struct {
		Monitors []MonitorConfiguration `json:"monitors"`
	}
	WorkflowConfigure struct {
		Mapping map[string]interface{} `json:"mapping,omitempty"`
		Enums   map[string]interface{} `json:"enums,omitempty"`
		Fixeds  map[string]interface{} `json:"fixeds"`
		Outputs map[string]interface{} `json:"outputs,omitempty"`
	}
	DockerWorkflow struct {
		Title     string            `json:"title"`
		Deleted   DockerDeleted     `json:"deleted"`
		Template  string            `json:"template"`
		Configure WorkflowConfigure `json:"configure"`
	}
	DockerDB struct {
		WorkSpaceID          string        `json:"workSpaceID"`
		State                DockerStatus  `json:"state"`
		SaveDir              string        `json:"saveDir"`
		Progress             string        `json:"progress"`
		PullTimer            string        `json:"pullTimer"`
		CompleteTimer        string        `json:"completeTimer"`
		RunningMode          RunMode       `json:"runMode"`
		MajorCMD             string        `json:"majorCMD"`
		MinorCMD             string        `json:"minorCMD"`
		MajorCompose         DockerCompose `json:"majorCompose"`
		MinorCompose         DockerCompose `json:"minorCompose"`
		ComfyUIHttp          string        `json:"comfyUIHttp"`
		ComfyUIWS            string        `json:"comfyUIWS"`
		Delete               DockerDeleted `json:"delete"`
		MeasureTitle         string        `json:"measureTitle"`
		Measure              string        `json:"measure"`
		MeasureConfiguration string        `json:"measureConfiguration"`
	}
	NodeBaseDB struct {
		SystemUUID   string `json:"systemUUID"`
		OS           string `json:"os"`
		CPU          string `json:"cpu"`
		GPU          string `json:"gpu"`
		GpuDriver    string `json:"gpuDriver"`
		Memory       uint64 `json:"memory"`
		RegisterTime string `json:"registerTime"`
		EnrollTime   string `json:"enrollTime"`
		HeartTime    string `json:"heartTime"`
		Versions     string `json:"versions"`
		MeasureTime  string `json:"measureTime"`
	}
	WorkSpaceBase struct {
		WorkspaceID          string            `json:"workSpaceID"`
		MajorCMD             string            `json:"majorCMD"`
		MinorCMD             string            `json:"minorCMD"`
		MajorCompose         DockerCompose     `json:"majorCompose"`
		MinorCompose         DockerCompose     `json:"minorCompose"`
		DeployCount          int64             `json:"deployCount"`
		RedundantCount       int64             `json:"redundantCount"`
		ComfyUIHttp          string            `json:"comfyUIHttp"`
		ComfyUIWS            string            `json:"comfyUIWS"`
		Deleted              DockerDeleted     `json:"deleted"`
		Measure              string            `json:"measure"`
		MeasureConfiguration string            `json:"measureConfiguration"`
		RunningMode          RunMode           `json:"runMode"`
		MinGB                int64             `json:"minGBVRam"`
		SavePath             string            `json:"f_save_path"`
		Workflows            []*DockerWorkflow `json:"workflows"`
	}
)

type (
	Md5FileContext struct {
		Url       string `json:"url"`
		CloudPath string `json:"cloudPath"`
		LocalPath string `json:"localPath"`
		MD5       string `json:"md5"`
		Start     bool   `json:"start"`
	}
	Md5Files struct {
		Md5File []Md5FileContext `json:"files"`
	}
)

type NodeDB struct {
	mysqlDb *MysqlManager
}

var gNodeDB *NodeDB

func (p *NodeDB) parseStartCMD(data []byte) DockerCompose {
	funName := "parseStartCMD"
	var compose DockerCompose
	if err := yaml.Unmarshal(data, &compose); err != nil {
		errString := fmt.Sprintf("%s yaml.Unmarshal Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return DockerCompose{}
	}
	return compose
}

func (p *NodeDB) WriteNode(systemUUID, OS, CPU, GPU, GpuDriver string, Memory uint64) (error, NodeBaseDB) {
	funName := "WriteNode"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect ---->>>>", funName)
		log4plus.Error(errString)
		return errors.New(errString), NodeBaseDB{}
	}
	//这里增加写入算力节点信息功能，避免非要进行一次注册步骤
	sql := fmt.Sprintf(`select IFNULL(f_id,-1) from t_nodes where f_system_uuid='%s' limit 1;`, systemUUID)
	rowsSelect, errSelect := p.mysqlDb.Mysqldb.Query(sql)
	if errSelect != nil {
		errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, errSelect.Error(), sql)
		log4plus.Error(errString)
		return errSelect, NodeBaseDB{}
	}
	defer rowsSelect.Close()

	exist := false
	for rowsSelect.Next() {
		var tmpId int64
		scanErr := rowsSelect.Scan(&tmpId)
		if scanErr != nil {
			log4plus.Error("%s Scan Error=[%s]", funName, scanErr.Error())
			continue
		}
		if tmpId != -1 {
			exist = true
		}
	}
	if !exist {
		sql = fmt.Sprintf(`insert into t_nodes (f_system_uuid, f_register_time, f_state, f_os, f_cpu, f_gpu, f_gpu_driver, f_memory) 
values ('%s', NOW(), 0, '%s', '%s', '%s', '%s', %d);`, systemUUID, OS, CPU, GPU, GpuDriver, Memory)
		_, errSelect = p.mysqlDb.Mysqldb.Exec(sql)
		if errSelect != nil {
			log4plus.Error("%s Insert Failed err=[%s] systemUUID=[%s] account=[%s] orgName=[%s]", funName, errSelect.Error(), systemUUID)
			return errSelect, NodeBaseDB{}
		}
	}
	//查询算力节点信息
	sql = fmt.Sprintf(`select IFNULL(f_id,-1), 
       IFNULL(f_system_uuid,''), IFNULL(f_os,''), IFNULL(f_cpu,''), IFNULL(f_gpu,''), IFNULL(f_gpu_driver,''), 
       IFNULL(f_memory,0), IFNULL(f_register_time,''), IFNULL(f_enroll_time,''),  IFNULL(f_heart_time,''),  IFNULL(f_versions,''), 
       IFNULL(f_measure_time, '') from t_nodes where f_system_uuid='%s' limit 1;`, systemUUID)
	rows, err := p.mysqlDb.Mysqldb.Query(sql)
	if err != nil {
		errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, err.Error(), sql)
		log4plus.Error(errString)
		return err, NodeBaseDB{}
	}
	defer rows.Close()
	var nodeBase NodeBaseDB
	for rows.Next() {
		var tmpId int64
		scanErr := rows.Scan(&tmpId,
			&nodeBase.SystemUUID, &nodeBase.OS, &nodeBase.CPU, &nodeBase.GPU, &nodeBase.GpuDriver,
			&nodeBase.Memory, &nodeBase.RegisterTime, &nodeBase.EnrollTime, &nodeBase.HeartTime, &nodeBase.Versions,
			&nodeBase.MeasureTime)
		if scanErr != nil {
			log4plus.Error("%s Scan Error=[%s]", funName, scanErr.Error())
			continue
		}
		return nil, nodeBase
	}
	errString := fmt.Sprintf("%s Query Failed SQL=[%s]", funName, sql)
	return errors.New(errString), NodeBaseDB{}
}

func (p *NodeDB) LoadNode(systemUUID string) (error, NodeBaseDB) {
	funName := "LoadNode"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect ---->>>>", funName)
		log4plus.Error(errString)
		return errors.New(errString), NodeBaseDB{}
	}
	//这里增加写入算力节点信息功能，避免非要进行一次注册步骤
	sql := fmt.Sprintf(`select IFNULL(f_id,-1) from t_nodes where f_system_uuid='%s' limit 1;`, systemUUID)
	rowsSelect, errSelect := p.mysqlDb.Mysqldb.Query(sql)
	if errSelect != nil {
		errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, errSelect.Error(), sql)
		log4plus.Error(errString)
		return errSelect, NodeBaseDB{}
	}
	defer rowsSelect.Close()

	exist := false
	for rowsSelect.Next() {
		var tmpId int64
		scanErr := rowsSelect.Scan(&tmpId)
		if scanErr != nil {
			log4plus.Error("%s Scan Error=[%s]", funName, scanErr.Error())
			continue
		}
		if tmpId != -1 {
			exist = true
		}
	}
	if !exist {
		errString := fmt.Sprintf("%s not found Node systemUUID=[%s] SQL=[%s]", funName, systemUUID, sql)
		log4plus.Error(errString)
		return errSelect, NodeBaseDB{}
	}
	//查询算力节点信息
	sql = fmt.Sprintf(`select IFNULL(f_id,-1), 
       IFNULL(f_system_uuid,''), IFNULL(f_os,''), IFNULL(f_cpu,''), IFNULL(f_gpu,''), IFNULL(f_gpu_driver,''), 
       IFNULL(f_memory,0), IFNULL(f_register_time,''), IFNULL(f_enroll_time,''),  IFNULL(f_heart_time,''),  IFNULL(f_versions,''), 
       IFNULL(f_measure_time, '') from t_nodes where f_system_uuid='%s' limit 1;`, systemUUID)
	rows, err := p.mysqlDb.Mysqldb.Query(sql)
	if err != nil {
		errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, err.Error(), sql)
		log4plus.Error(errString)
		return err, NodeBaseDB{}
	}
	defer rows.Close()
	var nodeBase NodeBaseDB
	for rows.Next() {
		var tmpId int64
		scanErr := rows.Scan(&tmpId,
			&nodeBase.SystemUUID, &nodeBase.OS, &nodeBase.CPU, &nodeBase.GPU, &nodeBase.GpuDriver,
			&nodeBase.Memory, &nodeBase.RegisterTime, &nodeBase.EnrollTime, &nodeBase.HeartTime, &nodeBase.Versions,
			&nodeBase.MeasureTime)
		if scanErr != nil {
			log4plus.Error("%s Scan Error=[%s]", funName, scanErr.Error())
			continue
		}
		return nil, nodeBase
	}
	errString := fmt.Sprintf("%s Query Failed SQL=[%s]", funName, sql)
	return errors.New(errString), NodeBaseDB{}
}

func (p *NodeDB) NodeRegister(systemUUID, account, orgName string) error {
	funName := "NodeRegister"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect ---->>>>", funName)
		log4plus.Error(errString)
		return errors.New(errString)
	}
	sql := fmt.Sprintf(`insert into t_nodes (f_system_uuid, f_account, f_org_name, f_register_time, f_state) values ('%s', '%s', '%s', NOW(), 0);`,
		systemUUID, account, orgName)
	_, err := p.mysqlDb.Mysqldb.Exec(sql)
	if err != nil {
		log4plus.Error("%s Insert Failed err=[%s] systemUUID=[%s] account=[%s] orgName=[%s]", funName, err.Error(), systemUUID, account, orgName)
		return err
	}
	return nil
}

func (p *NodeDB) NodeEnroll(systemUUID, nodeIp string, status NodeStatus) error {
	funName := "NodeEnroll"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect ---->>>>", funName)
		log4plus.Error(errString)
		return errors.New(errString)
	}
	sql := fmt.Sprintf(`update t_nodes set f_state=%d, f_node_ip='%s', f_enroll_time=NOW() where f_system_uuid='%s';`,
		int(status), nodeIp, systemUUID)
	_, err := p.mysqlDb.Mysqldb.Exec(sql)
	if err != nil {
		log4plus.Error("%s update Failed err=[%s] systemUUID=[%s]", funName, err.Error(), systemUUID)
		return err
	}
	return nil
}

func (p *NodeDB) NodeDockers(systemUUID string) (error, []DockerDB) {
	funName := "NodeDockers"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect ---->>>>", funName)
		log4plus.Error(errString)
		return errors.New(errString), []DockerDB{}
	}
	sql := fmt.Sprintf(`select IFNULL(a.f_id,-1), 
       	IFNULL(a.f_workspace_id,''), IFNULL(a.f_state,0), IFNULL(b.f_save_path,''), IFNULL(a.f_progress,''), IFNULL(a.f_pull_time,''), 
       	IFNULL(a.f_complete_time,''), IFNULL(b.f_major_cmd,''), IFNULL(b.f_minor_cmd,''), IFNULL(b.f_comfyui_http,''), IFNULL(b.f_comfyui_ws,''),  
       	IFNULL(b.f_run_mode, 0),  IFNULL(b.f_deleted, 0), IFNULL(b.f_measure_title, ''), IFNULL(b.f_measure, ''), 
       	IFNULL(b.f_measure_configuration, '')  
		from t_node_workspaces as a, t_workspaces as b where a.f_system_uuid='%s' and a.f_workspace_id=b.f_workspace_id;`, systemUUID)
	rows, err := p.mysqlDb.Mysqldb.Query(sql)
	if err != nil {
		errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, err.Error(), sql)
		log4plus.Error(errString)
		return err, []DockerDB{}
	}
	defer rows.Close()

	var dockerDBs []DockerDB
	for rows.Next() {
		var dockerDB DockerDB
		var tmpId int64
		scanErr := rows.Scan(&tmpId,
			&dockerDB.WorkSpaceID, &dockerDB.State, &dockerDB.SaveDir, &dockerDB.Progress, &dockerDB.PullTimer,
			&dockerDB.CompleteTimer, &dockerDB.MajorCMD, &dockerDB.MinorCMD, &dockerDB.ComfyUIHttp, &dockerDB.ComfyUIWS,
			&dockerDB.RunningMode, &dockerDB.Delete, &dockerDB.MeasureTitle, &dockerDB.Measure,
			&dockerDB.MeasureConfiguration)
		if scanErr != nil {
			log4plus.Error("%s Scan Error=[%s]", funName, scanErr.Error())
			continue
		}
		dockerDB.MajorCompose = p.parseStartCMD([]byte(dockerDB.MajorCMD))
		dockerDB.MinorCompose = p.parseStartCMD([]byte(dockerDB.MinorCMD))
		dockerDBs = append(dockerDBs, dockerDB)
	}
	return nil, dockerDBs
}

func (p *NodeDB) NodeHeart(systemUUID, nodeIp, os, cpu, gpu, gpuDriver, version string, memory uint64) error {
	funName := "NodeHeart"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect ---->>>>", funName)
		log4plus.Error(errString)
		return errors.New(errString)
	}
	sql := fmt.Sprintf(`update t_nodes set f_node_ip='%s', f_os='%s', f_cpu='%s', f_memory=%d, f_gpu='%s', f_gpu_driver='%s', 
                   f_versions='%s', f_heart_time=NOW() where f_system_uuid='%s';`,
		nodeIp, os, cpu, memory, gpu, gpuDriver, version, systemUUID)
	_, err := p.mysqlDb.Mysqldb.Exec(sql)
	if err != nil {
		log4plus.Error("%s update Failed err=[%s] systemUUID=[%s] nodeIp=[%s] os=[%s] cpu=[%s] memory=[%d] gpu=[%s] gpuDriver=[%s] version=[%s]",
			funName, err.Error(), systemUUID, nodeIp, os, cpu, memory, gpu, gpuDriver, version)
		return err
	}
	return nil
}

func (p *NodeDB) Configuration() (error, *MonitorConfigurations) {
	funName := "Configuration"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect ---->>>>", funName)
		log4plus.Error(errString)
		return errors.New(errString), nil
	}
	sql := `select IFNULL(f_id, -1), IFNULL(f_monitor_type, ''), IFNULL(f_monitor,'') from t_monitor_configuration;`
	rows, err := p.mysqlDb.Mysqldb.Query(sql)
	if err != nil {
		errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, err.Error(), sql)
		log4plus.Error(errString)
		return err, nil
	}
	defer rows.Close()

	monitors := &MonitorConfigurations{}
	for rows.Next() {
		var id int64
		var tmpMonitorType, tmpMonitor string
		scanErr := rows.Scan(&id, &tmpMonitorType, &tmpMonitor)
		if scanErr != nil {
			log4plus.Error("%s Scan Error=[%s]", funName, scanErr.Error())
			continue
		}
		monitor := MonitorConfiguration{
			MonitorType: tmpMonitorType,
			Monitor:     tmpMonitor,
		}
		monitors.Monitors = append(monitors.Monitors, monitor)
	}
	return nil, monitors
}

func removeLineBreaks(input string) string {
	var builder strings.Builder
	for _, char := range input {
		if char != '\r' && char != '\n' {
			builder.WriteRune(char)
		}
	}
	return builder.String()
}

func (p *NodeDB) extractGPUModel(line string) string {
	re := regexp.MustCompile(`NVIDIA\s+([A-Za-z0-9\s\-]+)`)
	matches := re.FindStringSubmatch(line)
	if len(matches) >= 2 {
		gpuType := matches[0]
		gpuType = removeLineBreaks(gpuType)
		return gpuType
	}
	return ""
}

func (p *NodeDB) SlotSize(systemUUID string) (error, int) {
	funName := "SlotSize"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect ---->>>>", funName)
		log4plus.Error(errString)
		return errors.New(errString), 0
	}
	sql := fmt.Sprintf(`select IFNULL(f_id,-1), IFNULL(f_gpu,'') from t_nodes where f_system_uuid='%s' limit 1;`, systemUUID)
	rows, err := p.mysqlDb.Mysqldb.Query(sql)
	if err != nil {
		errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, err.Error(), sql)
		log4plus.Warn(errString)
		return err, 1
	}
	defer rows.Close()

	gpu := ""
	for rows.Next() {
		var id int64
		scanErr := rows.Scan(&id, &gpu)
		if scanErr != nil {
			log4plus.Error(" %s Scan Error=[%s]", funName, scanErr.Error())
			continue
		}
		break
	}
	if gpu == "" {
		errString := fmt.Sprintf("%s node gpu not found systemUUID=[%s] slotSize=[1]", funName, systemUUID)
		log4plus.Error(errString)
		return nil, 1
	}
	gpuName := p.extractGPUModel(gpu)
	gpuName = strings.Trim(gpuName, " ")
	if gpuName == "" {
		return errors.New("not found gpu type"), 0
	}
	sql = fmt.Sprintf(`select IFNULL(b.f_id,-1), IFNULL(b.f_pool_size, 0) from t_gpu_family_gpu as a , t_gpu_family as b 
                                               where a.f_gpu_family_id=b.f_id and a.f_gpu_name='%s' limit 1;`, gpuName)
	rows, err = p.mysqlDb.Mysqldb.Query(sql)
	if err != nil {
		errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, err.Error(), sql)
		log4plus.Error(errString)
		return err, 0
	}
	defer rows.Close()

	for rows.Next() {
		var id, poolsize int
		scanErr := rows.Scan(&id, &poolsize)
		if scanErr != nil {
			log4plus.Error(" %s Scan Error=[%s]", funName, scanErr.Error())
			continue
		}
		return nil, poolsize
	}
	return nil, 1
}

func (p *NodeDB) WorkSpaces() (error, []*WorkSpaceBase) {
	funName := "WorkSpaces"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect ---->>>>", funName)
		log4plus.Error(errString)
		return errors.New(errString), nil
	}
	sql := `select IFNULL(f_id,-1), IFNULL(f_workspace_id,''), IFNULL(f_major_cmd,''), IFNULL(f_minor_cmd,''), IFNULL(f_deploy_count, 0),
    IFNULL(f_redundant_count, 0), IFNULL(f_comfyui_http, ''),  IFNULL(f_comfyui_ws, ''), IFNULL(f_deleted, 0), IFNULL(f_measure, ''), 
    IFNULL(f_measure_configuration, ''), IFNULL(f_run_mode, 0), IFNULL(f_min_gb_vram, 0), IFNULL(f_save_path, '') from t_workspaces order by f_workspace_id;`
	rows, err := p.mysqlDb.Mysqldb.Query(sql)
	if err != nil {
		errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, err.Error(), sql)
		log4plus.Warn(errString)
		return err, nil
	}
	defer rows.Close()

	var workspaces []*WorkSpaceBase
	for rows.Next() {
		var id int64
		var workspace WorkSpaceBase
		scanErr := rows.Scan(&id, &workspace.WorkspaceID, &workspace.MajorCMD, &workspace.MinorCMD, &workspace.DeployCount,
			&workspace.RedundantCount, &workspace.ComfyUIHttp, &workspace.ComfyUIWS, &workspace.Deleted, &workspace.Measure,
			&workspace.MeasureConfiguration, &workspace.RunningMode, &workspace.MinGB, &workspace.SavePath)
		if scanErr != nil {
			log4plus.Error(" %s Scan Error=[%s]", funName, scanErr.Error())
			continue
		}
		workspace.MajorCompose = p.parseStartCMD([]byte(workspace.MajorCMD))
		workspace.MinorCompose = p.parseStartCMD([]byte(workspace.MinorCMD))
		workspaces = append(workspaces, &workspace)
	}
	for _, workSpace := range workspaces {
		sql = fmt.Sprintf(`select IFNULL(a.f_id,-1), IFNULL(b.f_title,''), IFNULL(b.f_deleted,0), IFNULL(b.f_template,''), 
       	IFNULL(f_configuration, '') from t_workspace_workflow as a, t_workflows as b where a.f_title=b.f_title and a.f_workspace_id='%s';`, workSpace.WorkspaceID)
		rows, err = p.mysqlDb.Mysqldb.Query(sql)
		if err != nil {
			errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, err.Error(), sql)
			log4plus.Warn(errString)
			return err, nil
		}
		defer rows.Close()

		for rows.Next() {
			var id int64
			var deleted int
			var title, template, workflowConfigure string
			scanErr := rows.Scan(&id, &title, &deleted, &template, &workflowConfigure)
			if scanErr != nil {
				log4plus.Error(" %s Scan Error=[%s]", funName, scanErr.Error())
				continue
			}
			var workflowsConfig WorkflowConfigure
			_ = json.Unmarshal([]byte(workflowConfigure), &workflowsConfig)
			workflow := DockerWorkflow{
				Title:     title,
				Deleted:   DockerDeleted(deleted),
				Template:  template,
				Configure: workflowsConfig,
			}
			workSpace.Workflows = append(workSpace.Workflows, &workflow)
		}
	}
	return nil, workspaces
}

func (p *NodeDB) GetMeasure(workspaceID string) (error, string) {
	funName := "GetMeasure"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect ---->>>>", funName)
		log4plus.Error(errString)
		return errors.New(errString), ""
	}
	sql := fmt.Sprintf(`select IFNULL(f_id,-1), IFNULL(f_measure, '') from t_workspaces where f_workspace_id='%s' limit 1;`, workspaceID)
	rows, err := p.mysqlDb.Mysqldb.Query(sql)
	if err != nil {
		errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, err.Error(), sql)
		log4plus.Warn(errString)
		return err, ""
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var measure string
		scanErr := rows.Scan(&id, &measure)
		if scanErr != nil {
			log4plus.Error(" %s Scan Error=[%s]", funName, scanErr.Error())
			continue
		}
		return nil, measure
	}
	return errors.New("not found workspaceID"), ""
}

func (p *NodeDB) DispenseStart(systemUUID, workspaceID string, status DockerStatus) error {
	funName := "DispenseStart"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect ---->>>>", funName)
		log4plus.Error(errString)
		return errors.New(errString)
	}
	sql := fmt.Sprintf(`select IFNULL(f_id,-1), IFNULL(f_system_uuid, ''), IFNULL(f_workspace_id, '') from t_node_workspaces 
                                                                              where f_system_uuid='%s' and f_workspace_id='%s' limit 1;`,
		systemUUID, workspaceID)
	rows, err := p.mysqlDb.Mysqldb.Query(sql)
	if err != nil {
		errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, err.Error(), sql)
		log4plus.Error(errString)
		return err
	}
	defer rows.Close()

	exist := false
	for rows.Next() {
		var id int64
		var tmpSystemUUID, tmpWorkspaceID string
		scanErr := rows.Scan(&id, &tmpSystemUUID, &tmpWorkspaceID)
		if scanErr != nil {
			log4plus.Error(" %s Scan Error=[%s]", funName, scanErr.Error())
			continue
		}
		if id != -1 {
			exist = true
		}
	}
	if !exist {
		sql = fmt.Sprintf(`insert into t_node_workspaces (f_system_uuid, f_workspace_id, f_state, f_progress, f_pull_time) values 
                                                        ('%s', '%s', %d, '0', NOW());`,
			systemUUID, workspaceID, int(status))

	} else {
		sql = fmt.Sprintf(`update t_node_workspaces set f_state=%d, f_progress='0', f_pull_time=NOW() where f_system_uuid='%s' and f_workspace_id='%s' limit 1;`,
			int(status), systemUUID, workspaceID)
	}
	_, err = p.mysqlDb.Mysqldb.Exec(sql)
	if err != nil {
		log4plus.Error("%s Insert Into Failed err=[%s] systemUUID=[%s] workspaceId=[%s]", funName, err.Error(), systemUUID, workspaceID)
		return err
	}
	return nil
}

func (p *NodeDB) DispenseProgress(systemUUID, workspaceID, progress string, status DockerStatus) error {
	funName := "DispenseProgress"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect ---->>>>", funName)
		log4plus.Error(errString)
		return errors.New(errString)
	}
	sql := fmt.Sprintf(`update t_node_workspaces set f_progress='%s', f_state=%d where f_system_uuid='%s' and f_workspace_id='%s' limit 1;`,
		progress, int(status), systemUUID, workspaceID)
	_, err := p.mysqlDb.Mysqldb.Exec(sql)
	if err != nil {
		log4plus.Error("%s Update Failed err=[%s] systemUUID=[%s] workspaceId=[%s] progress=[%s]", funName, err.Error(), systemUUID, workspaceID, progress)
		return err
	}
	return nil
}

func (p *NodeDB) DispenseComplete(systemUUID, workspaceID, saveDir, progress string, status DockerStatus) error {
	funName := "DispenseComplete"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect ---->>>>", funName)
		log4plus.Error(errString)
		return errors.New(errString)
	}
	sql := fmt.Sprintf(`update t_node_workspaces set f_save_dir='%s', f_state=%d, f_progress='%s', f_complete_time=NOW() where 
                                                                                                     f_system_uuid='%s' and f_workspace_id='%s' limit 1;`,
		saveDir, int(status), progress, systemUUID, workspaceID)
	_, err := p.mysqlDb.Mysqldb.Exec(sql)
	if err != nil {
		log4plus.Error("%s Update Failed err=[%s] systemUUID=[%s] workspaceId=[%s] saveDir=[%s]", funName, err.Error(), systemUUID, workspaceID, saveDir)
		return err
	}
	return nil
}

func (p *NodeDB) SetDockerStatus(systemUUID, workspaceID string, status DockerStatus) error {
	funName := "SetDockerStatus"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect ---->>>>", funName)
		log4plus.Error(errString)
		return errors.New(errString)
	}
	sql := fmt.Sprintf(`update t_node_workspaces set f_state=%d where f_system_uuid='%s' and f_workspace_id='%s' limit 1;`, int(status), systemUUID, workspaceID)
	_, err := p.mysqlDb.Mysqldb.Exec(sql)
	if err != nil {
		log4plus.Error("%s Update Failed err=[%s] systemUUID=[%s] workspaceId=[%s]", funName, err.Error(), systemUUID, workspaceID)
		return err
	}
	return nil
}

func (p *NodeDB) StartWorkSpace(systemUUID, workspaceID string, status DockerStatus) error {
	funName := "StartWorkSpace"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect ---->>>>", funName)
		log4plus.Error(errString)
		return errors.New(errString)
	}
	sql := fmt.Sprintf(`select IFNULL(f_id,-1), IFNULL(f_system_uuid, ''), IFNULL(f_workspace_id, '') from t_node_workspaces where 
                                                                                                         f_system_uuid='%s' and f_workspace_id='%s' limit 1;`,
		systemUUID, workspaceID)
	rows, err := p.mysqlDb.Mysqldb.Query(sql)
	if err != nil {
		errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, err.Error(), sql)
		log4plus.Error(errString)
		return err
	}
	defer rows.Close()

	exist := false
	for rows.Next() {
		var id int64
		var tmpSystemUUID, tmpWorkspaceID string
		scanErr := rows.Scan(&id, &tmpSystemUUID, &tmpWorkspaceID)
		if scanErr != nil {
			log4plus.Error(" %s Scan Error=[%s]", funName, scanErr.Error())
			continue
		}
		if id != -1 {
			exist = true
		}
	}
	if !exist {
		sql = fmt.Sprintf(`insert into t_node_workspaces (f_system_uuid, f_workspace_id, f_state, f_progress, f_pull_time) values 
                                                        ('%s', '%s', %d, '0', NOW());`,
			systemUUID, workspaceID, int(status))

	} else {
		sql = fmt.Sprintf(`update t_node_workspaces set f_state=%d, f_progress='0', f_pull_time=NOW() where f_system_uuid='%s' and f_workspace_id='%s' limit 1;`,
			int(status), systemUUID, workspaceID)
	}
	log4plus.Info("%s Insert Into SQL=[%s]", funName, sql)
	_, err = p.mysqlDb.Mysqldb.Exec(sql)
	if err != nil {
		log4plus.Error("%s Insert Into Failed err=[%s] systemUUID=[%s] workspaceId=[%s]", funName, err.Error(), systemUUID, workspaceID)
		return err
	}
	return nil
}

func (p *NodeDB) StartupWorkSpace(systemUUID, workspaceID string, state DockerStatus) error {
	funName := "StartupWorkSpace"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect ---->>>>", funName)
		log4plus.Error(errString)
		return errors.New(errString)
	}
	sql := fmt.Sprintf(`update t_node_workspaces set f_state=%d, f_complete_time=NOW()  where f_system_uuid='%s' and f_workspace_id='%s' limit 1;`,
		state, systemUUID, workspaceID)
	_, err := p.mysqlDb.Mysqldb.Exec(sql)
	if err != nil {
		log4plus.Error("%s Update Failed err=[%s] systemUUID=[%s] workspaceId=[%s]", funName, err.Error(), systemUUID, workspaceID)
		return err
	}
	return nil
}

func (p *NodeDB) DelWorkSpace(systemUUID, workspaceID string) error {
	funName := "DelWorkSpace"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect ---->>>>", funName)
		log4plus.Error(errString)
		return errors.New(errString)
	}
	sql := fmt.Sprintf(`delete  from t_node_workspaces where f_system_uuid='%s' and f_workspace_id='%s' limit 1;`,
		systemUUID, workspaceID)
	_, err := p.mysqlDb.Mysqldb.Exec(sql)
	if err != nil {
		log4plus.Error("%s delete Failed err=[%s] systemUUID=[%s] workspaceId=[%s]", funName, err.Error(), systemUUID, workspaceID)
		return err
	}
	return nil
}

func (p *NodeDB) SetMeasureCompletion(systemUUID string, state NodeStatus) error {
	funName := "SetMeasureCompletion"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect ---->>>>", funName)
		log4plus.Error(errString)
		return errors.New(errString)
	}
	sql := fmt.Sprintf(`update t_nodes set f_measure_time=NOW(), f_state=%d where f_system_uuid='%s';`, int(state), systemUUID)
	_, err := p.mysqlDb.Mysqldb.Exec(sql)
	if err != nil {
		log4plus.Error("%s update Failed err=[%s] systemUUID=[%s] standardConsuming=[%d]", funName, err.Error(), systemUUID)
		return err
	}
	return nil
}

func (p *NodeDB) DispenseFail(systemUUID, workspaceID string, state DockerStatus) error {
	funName := "DispenseFail"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect ---->>>>", funName)
		log4plus.Error(errString)
		return errors.New(errString)
	}
	sql := fmt.Sprintf(`update t_node_workspaces set f_state=%d where f_system_uuid='%s' and f_workspace_id='%s' limit 1;`,
		int(state), systemUUID, workspaceID)
	_, err := p.mysqlDb.Mysqldb.Exec(sql)
	if err != nil {
		log4plus.Error("%s update Failed err=[%s] systemUUID=[%s] workspaceId=[%s]", funName, err.Error(), systemUUID, workspaceID)
		return err
	}
	return nil
}

func (p *NodeDB) WorkSpace2Title(workspaceID string) (error, []*DockerWorkflow) {
	funName := "WorkSpaceToJobType"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect ---->>>>", funName)
		log4plus.Error(errString)
		return errors.New(errString), nil
	}
	sql := fmt.Sprintf(`select IFNULL(a.f_id,-1), IFNULL(a.f_title,''), IFNULL(b.f_deleted, 0), IFNULL(b.f_template, ''), IFNULL(b.f_configuration, '')    
       from t_workspace_workflow as a, t_workflows as b where a.f_workspace_id='%s' and a.f_title = b.f_title;`, workspaceID)
	rows, err := p.mysqlDb.Mysqldb.Query(sql)
	if err != nil {
		errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, err.Error(), sql)
		log4plus.Warn(errString)
		return err, nil
	}
	defer rows.Close()

	var workflows []*DockerWorkflow
	for rows.Next() {
		var id int64
		var title, template, workflowConfigure string
		var deleted int
		scanErr := rows.Scan(&id, &title, &deleted, &template, &workflowConfigure)
		if scanErr != nil {
			log4plus.Error(" %s Scan Error=[%s]", funName, scanErr.Error())
			continue
		}
		var workflowsConfig WorkflowConfigure
		_ = json.Unmarshal([]byte(workflowConfigure), &workflowsConfig)
		workflow := DockerWorkflow{
			Title:     title,
			Deleted:   DockerDeleted(deleted),
			Template:  template,
			Configure: workflowsConfig,
		}
		workflows = append(workflows, &workflow)
	}
	return nil, workflows
}

func (p *NodeDB) Title2Workflow(workflowTitle string) (error, DockerWorkflow, []string) {
	funName := "Title2Workflow"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect ---->>>>", funName)
		log4plus.Error(errString)
		return errors.New(errString), DockerWorkflow{}, []string{}
	}
	sql := fmt.Sprintf(`select IFNULL(f_id, -1), IFNULL(f_deleted, 0), IFNULL(f_template, ''), IFNULL(f_configuration, '') from t_workflows 
                                                                                                   where f_title='%s';`, workflowTitle)
	rows, err := p.mysqlDb.Mysqldb.Query(sql)
	if err != nil {
		errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, err.Error(), sql)
		log4plus.Warn(errString)
		return err, DockerWorkflow{}, []string{}
	}
	defer rows.Close()
	var workFlow DockerWorkflow
	for rows.Next() {
		var id, deleted int64
		var tmpTemplate, tmpConfigure string
		scanErr := rows.Scan(&id, &deleted, &tmpTemplate, &tmpConfigure)
		if scanErr != nil {
			log4plus.Error(" %s Scan Error=[%s]", funName, scanErr.Error())
			continue
		}
		if id != -1 {
			workFlow.Title = workflowTitle
			workFlow.Deleted = DockerDeleted(deleted)
			workFlow.Template = tmpTemplate
			_ = json.Unmarshal([]byte(tmpConfigure), &workFlow.Configure)
		}
	}
	sql = fmt.Sprintf(`select IFNULL(a.f_id,-1), IFNULL(b.f_workspace_id,'') from t_workflows as a, t_workspace_workflow as b 
                                                      where a.f_title=b.f_title and a.f_title='%s';`, workflowTitle)
	rowsDB, errID := p.mysqlDb.Mysqldb.Query(sql)
	if errID != nil {
		errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, errID.Error(), sql)
		log4plus.Warn(errString)
		return err, DockerWorkflow{}, []string{}
	}
	defer rowsDB.Close()

	var workSpaceIDs []string
	for rowsDB.Next() {
		var id int64
		var workSpaceID string
		scanErr := rowsDB.Scan(&id, &workSpaceID)
		if scanErr != nil {
			log4plus.Error(" %s Scan Error=[%s]", funName, scanErr.Error())
			continue
		}
		if id != -1 {
			workSpaceIDs = append(workSpaceIDs, workSpaceID)
		}
	}
	return nil, workFlow, workSpaceIDs
}

func (p *NodeDB) GetCompletion(familyID int64, title string, width, height int64) (error, int64) {
	funName := "GetCompletion"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect ---->>>>", funName)
		log4plus.Error(errString)
		return errors.New(errString), 0
	}
	//Obtain based on familyID
	sql := fmt.Sprintf(`select IFNULL(f_id, -1), IFNULL(f_completion_time, 0) from t_workflow_completion_times 
            where f_gpu_family_id=%d and f_workflow_title='%s' and f_image_width=%d and f_image_height=%d;`, familyID, title, width, height)
	rows, err := p.mysqlDb.Mysqldb.Query(sql)
	if err != nil {
		errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, err.Error(), sql)
		log4plus.Warn(errString)
		return err, 0
	}
	defer rows.Close()
	for rows.Next() {
		var id, completionTime int64
		scanErr := rows.Scan(&id, &completionTime)
		if scanErr != nil {
			log4plus.Error(" %s Scan Error=[%s]", funName, scanErr.Error())
			continue
		}
		if id != -1 {
			return nil, completionTime
		}
	}
	return errors.New("not found "), 0
}

func (p *NodeDB) GetFamilyVRam(gpuName string) (err error, familyID int64, videoRam int64, familyWeight float64) {
	funName := "GetFamilyVRam"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect ---->>>>", funName)
		log4plus.Error(errString)
		return errors.New(errString), 0, 0, 0
	}
	sql := fmt.Sprintf(`select IFNULL(a.f_id,-1), IFNULL(a.f_gpu_family_id, 0), IFNULL(b.f_gb_vram, 0), IFNULL(b.f_weight, 0) 
    from t_gpu_family_gpu as a, t_gpu_family as b where a.f_gpu_name='%s' and a.f_gpu_family_id=b.f_id limit 1;`, gpuName)
	rows, errRows := p.mysqlDb.Mysqldb.Query(sql)
	if errRows != nil {
		errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, errRows.Error(), sql)
		log4plus.Warn(errString)
		return errRows, 0, 0, 0
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		scanErr := rows.Scan(&id, &familyID, &videoRam, &familyWeight)
		if scanErr != nil {
			log4plus.Error(" %s Scan Error=[%s]", funName, scanErr.Error())
			continue
		}
		return nil, familyID, videoRam, familyWeight
	}
	return errors.New("not found workspaceID"), 0, 0, 0
}

func (p *NodeDB) GetClientVersion() (err error, version, createTime string, md5Files Md5Files) {
	funName := "GetClientVersion"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect ---->>>>", funName)
		log4plus.Error(errString)
		return errors.New(errString), "", "", Md5Files{}
	}
	sql := fmt.Sprintf(`select IFNULL(f_id,-1), IFNULL(f_version, ''), IFNULL(f_files, ''), IFNULL(f_createtime, '') from t_client_version limit 1;`)
	rows, errRows := p.mysqlDb.Mysqldb.Query(sql)
	if errRows != nil {
		errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, errRows.Error(), sql)
		log4plus.Warn(errString)
		return errRows, "", "", Md5Files{}
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var body string
		scanErr := rows.Scan(&id, &version, &body, &createTime)
		if scanErr != nil {
			log4plus.Error(" %s Scan Error=[%s]", funName, scanErr.Error())
			continue
		}
		if err = json.Unmarshal([]byte(body), &md5Files); err != nil {
			errString := fmt.Sprintf("%s Unmarshal Failed Error=[%s] SQL=[%s]", funName, err.Error(), sql)
			log4plus.Warn(errString)
			return errRows, "", "", Md5Files{}
		}
		return nil, version, createTime, md5Files
	}
	return errors.New("not found version"), "", "", Md5Files{}
}

func (p *NodeDB) GetAvailableGPU() (err error, gpus []string) {
	funName := "GetAvailableGPU"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect ---->>>>", funName)
		log4plus.Error(errString)
		return errors.New(errString), []string{}
	}
	sql := fmt.Sprintf(`select IFNULL(f_id,-1), IFNULL(f_gpu_model, '') from t_available_gpu;`)
	rows, errRows := p.mysqlDb.Mysqldb.Query(sql)
	if errRows != nil {
		errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, errRows.Error(), sql)
		log4plus.Warn(errString)
		return errRows, []string{}
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var gpu string
		scanErr := rows.Scan(&id, &gpu)
		if scanErr != nil {
			log4plus.Error(" %s Scan Error=[%s]", funName, scanErr.Error())
			continue
		}
		gpus = append(gpus, gpu)
	}
	return nil, gpus
}

func (p *NodeDB) GetGPUPriority() (err error, gpus []string, levels []int64) {
	funName := "GetGPUPriority"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect ---->>>>", funName)
		log4plus.Error(errString)
		return errors.New(errString), []string{}, []int64{}
	}
	sql := fmt.Sprintf(`select IFNULL(f_id,-1), IFNULL(f_gpu_model, ''), IFNULL(f_scheduling_priority, -1) from t_available_gpu;`)
	rows, errRows := p.mysqlDb.Mysqldb.Query(sql)
	if errRows != nil {
		errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, errRows.Error(), sql)
		log4plus.Warn(errString)
		return errRows, []string{}, []int64{}
	}
	defer rows.Close()

	for rows.Next() {
		var id, level int64
		var gpu string
		scanErr := rows.Scan(&id, &gpu, &level)
		if scanErr != nil {
			log4plus.Error(" %s Scan Error=[%s]", funName, scanErr.Error())
			continue
		}
		gpus = append(gpus, gpu)
		levels = append(levels, level)
	}
	return nil, gpus, levels
}

func (p *NodeDB) WriteStartCMD(workspaceID, startCMD string) error {
	funName := "WriteStartCMD"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect ---->>>>", funName)
		log4plus.Error(errString)
		return errors.New(errString)
	}
	sql := fmt.Sprintf(`update t_workspaces set f_start_cmd='%s' where f_workspace_id='%s' limit 1;`, startCMD, workspaceID)
	_, err := p.mysqlDb.Mysqldb.Exec(sql)
	if err != nil {
		log4plus.Error("%s Update Failed err=[%s] workspaceID=[%s]", funName, err.Error(), workspaceID)
		return err
	}
	return nil
}

func (p *NodeDB) SetOnlineTime(systemUUID, Day string, hour int, onlineTime int) error {
	funName := "SetOnlineTime"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect ---->>>>", funName)
		log4plus.Error(errString)
		return errors.New(errString)
	}
	sql := fmt.Sprintf(`select IFNULL(f_id,-1), IFNULL(f_online_time, 0) from t_node_online_time 
                                                                              where f_system_uuid='%s' and f_date='%s' and f_hour=%d limit 1;`,
		systemUUID, Day, hour)
	rows, err := p.mysqlDb.Mysqldb.Query(sql)
	if err != nil {
		errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, err.Error(), sql)
		log4plus.Error(errString)
		return err
	}
	defer rows.Close()

	exist := false
	var id, curOnlineTime int
	for rows.Next() {
		scanErr := rows.Scan(&id, &curOnlineTime)
		if scanErr != nil {
			log4plus.Error(" %s Scan Error=[%s]", funName, scanErr.Error())
			continue
		}
		if id != -1 {
			exist = true
		}
	}
	if !exist {
		sql = fmt.Sprintf(`insert into t_node_online_time (f_system_uuid, f_date, f_hour, f_online_time) values 
                                                        ('%s', '%s', %d, %d);`, systemUUID, Day, hour, onlineTime)
	} else {
		newOnlineTime := curOnlineTime + onlineTime
		if newOnlineTime > 60*60 {
			newOnlineTime = 60 * 60
			log4plus.Error("%s update online time systemUUID=[%s] day=[%s] hour=[%d] curOnlineTime=[%d] onlineTime=[%d]",
				funName, systemUUID, Day, hour, curOnlineTime, onlineTime)
		}
		sql = fmt.Sprintf(`update t_node_online_time set f_online_time=%d where f_system_uuid='%s' and f_date='%s' and f_hour=%d limit 1;`,
			newOnlineTime, systemUUID, Day, hour)
	}
	_, err = p.mysqlDb.Mysqldb.Exec(sql)
	if err != nil {
		log4plus.Error("%s mysqlDb.Mysqldb.Exec Failed err=[%s] systemUUID=[%s] Day=[%s] hour=[%d]", funName, err.Error(), systemUUID, Day, hour)
		return err
	}
	return nil
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
