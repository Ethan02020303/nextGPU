package db

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/nextGPU/ng-cloud/configure"
	log4plus "github.com/nextGPU/include/log4go"
	"strconv"
	"strings"
	"time"
)

type BillingMode int

const (
	NormalTask BillingMode = iota
	AdminTask
)

type TaskStatus int

const (
	CreateStatus TaskStatus = iota
	PublishStatus
	DealingStatus
	DealCompleted
	DealFailed
)

type TaskDB struct {
	mysqlDb *MysqlManager
}

var gTaskDB *TaskDB

func (p *TaskDB) InsertMainTask(taskID, session, title string,
	subNum int64,
	billMode BillingMode,
	input, workspaceID string,
	generatedData json.RawMessage,
	requestData string) error {
	funName := "InsertMainTask"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect", funName)
		log4plus.Error(errString)
		return errors.New(errString)
	}
	sql := fmt.Sprintf(`insert into t_jobs (f_task_id, f_session, f_publish_time, f_title, f_state, 
                    f_bill_mode, f_sub_num, f_input, f_workspace_id, 
                    f_generated_data, f_request) values 
                                                    ('%s', '%s', NOW(), '%s', %d, 
                                                     %d, %d, ?, '%s', 
                                                     ?, ?);`,
		taskID, session, title, int(CreateStatus), int(billMode), subNum, workspaceID)
	_, err := p.mysqlDb.Mysqldb.Exec(sql, input, generatedData, requestData)
	if err != nil {
		log4plus.Error("%s insert Failed err=[%s] taskId=[%s] session=[%s] workspaceID=[%s]", funName, err.Error(), taskID, session, workspaceID)
		return err
	}
	return nil
}

func (p *TaskDB) SetPublishTime(taskID string, publishTime string) error {
	funName := "SetPublishTime"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect", funName)
		log4plus.Error(errString)
		return errors.New(errString)
	}
	sql := fmt.Sprintf(`select IFNULL(f_id,-1), IFNULL(f_publish_time,'') from t_jobs where f_task_id='%s' limit 1;`, taskID)
	rows, err := p.mysqlDb.Mysqldb.Query(sql)
	if err != nil {
		errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, err.Error(), sql)
		log4plus.Error(errString)
		return err
	}
	defer rows.Close()

	tmpId := -1
	tmpPublishTime := ""
	for rows.Next() {
		scanErr := rows.Scan(&tmpId, &tmpPublishTime)
		if scanErr != nil {
			log4plus.Error("%s Scan Error=[%s]", funName, scanErr.Error())
			continue
		}
	}
	if tmpId == -1 {
		return errors.New(fmt.Sprintf("%s Not Found row taskID=[%s]", funName, taskID))
	}
	if tmpPublishTime == "" {
		sql = fmt.Sprintf(`update t_jobs set f_publish_time='%s', f_state=%d where f_task_id='%s';`,
			publishTime, int(PublishStatus), taskID)
		_, err = p.mysqlDb.Mysqldb.Exec(sql)
		if err != nil {
			log4plus.Error("%s update Failed err=[%s] taskID=[%s] publishTime=[%s]", funName, err.Error(), taskID, publishTime)
			return err
		}
	}
	return nil
}

func (p *TaskDB) SetStartTime(taskID string, startTime string) error {
	funName := "SetStartTime"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect", funName)
		log4plus.Error(errString)
		return errors.New(errString)
	}
	sql := fmt.Sprintf(`select IFNULL(f_id,-1), IFNULL(f_start_time,'') from t_jobs where f_task_id='%s' limit 1;`, taskID)
	rows, err := p.mysqlDb.Mysqldb.Query(sql)
	if err != nil {
		errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, err.Error(), sql)
		log4plus.Error(errString)
		return err
	}
	defer rows.Close()

	tmpId := -1
	tmpStartTime := ""
	for rows.Next() {
		scanErr := rows.Scan(&tmpId, &tmpStartTime)
		if scanErr != nil {
			log4plus.Error("%s Scan Error=[%s]", funName, scanErr.Error())
			continue
		}
	}
	if tmpId == -1 {
		return errors.New(fmt.Sprintf("%s Not Found row taskID=[%s]", funName, taskID))
	}
	if tmpStartTime == "" {
		sql = fmt.Sprintf(`update t_jobs set f_start_time='%s', f_state=%d where f_task_id='%s';`,
			startTime, int(DealingStatus), taskID)
		_, err = p.mysqlDb.Mysqldb.Exec(sql)
		if err != nil {
			log4plus.Error("%s update Failed err=[%s] taskId=[%s] starttime=[%s]", funName, err.Error(), taskID, startTime)
			return err
		}
	}
	return nil
}

func (p *TaskDB) SetTaskFailed(taskID, endTime string) {
	funName := "SetTaskFailed"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect", funName)
		log4plus.Error(errString)
		return
	}
	sql := fmt.Sprintf(`update t_jobs set f_completion_time='%s', f_state=%d where f_task_id='%s';`,
		endTime, int(DealFailed), taskID)
	_, err := p.mysqlDb.Mysqldb.Exec(sql)
	if err != nil {
		log4plus.Error("%s update Failed err=[%s] taskId=[%s] completiontime=[%s]", funName, err.Error(), taskID, endTime)
		return
	}
}

func (p *TaskDB) SetTaskResult(taskID string, completionTime string) error {
	funName := "SetTaskResult"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect", funName)
		log4plus.Error(errString)
		return errors.New(errString)
	}
	sql := fmt.Sprintf(`update t_jobs set f_completion_time='%s', f_state=%d where f_task_id='%s';`,
		completionTime, int(DealCompleted), taskID)
	_, err := p.mysqlDb.Mysqldb.Exec(sql)
	if err != nil {
		log4plus.Error("%s update Failed err=[%s] taskId=[%s] completiontime=[%s]", funName, err.Error(), taskID, completionTime)
		return err
	}
	return nil
}

func (p *TaskDB) SetTaskFailedResult(taskID string) error {
	funName := "SetTaskResult"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect", funName)
		log4plus.Error(errString)
		return errors.New(errString)
	}
	sql := fmt.Sprintf(`update t_jobs set f_completion_time=NOW(), f_state=%d where f_task_id='%s';`, int(DealFailed), taskID)
	_, err := p.mysqlDb.Mysqldb.Exec(sql)
	if err != nil {
		log4plus.Error("%s update Failed err=[%s] taskId=[%s] state=[%d]", funName, err.Error(), taskID, int(DealFailed))
		return err
	}
	return nil
}

func (p *TaskDB) InsertSubTask(taskID, subID, input, workspaceID string) error {
	funName := "InsertSubTask"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect", funName)
		log4plus.Error(errString)
		return errors.New(errString)
	}
	sql := fmt.Sprintf(`insert into t_sub_jobs (f_task_id, f_sub_id, f_create_time, f_state, f_input, f_workspace_id) values 
                                                                                                          ('%s', '%s', NOW(), %d, ?, '%s');`,
		taskID, subID, int(CreateStatus), workspaceID)
	_, err := p.mysqlDb.Mysqldb.Exec(sql, input)
	if err != nil {
		log4plus.Error("%s insert Failed err=[%s] taskId=[%s] subId=[%s] workspaceid=[%s]", funName, err.Error(), taskID, subID, workspaceID)
		return err
	}
	return nil
}

func (p *TaskDB) SetSubTaskPublish(taskID, subID, systemUUID, publishTime string) error {
	funName := "SetSubTaskPublish"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect", funName)
		log4plus.Error(errString)
		return errors.New(errString)
	}
	sql := fmt.Sprintf(`update t_sub_jobs set f_system_uuid='%s', f_publish_time='%s', f_state=%d where f_task_id='%s' and f_sub_id='%s';`,
		systemUUID, publishTime, int(PublishStatus), taskID, subID)
	_, err := p.mysqlDb.Mysqldb.Exec(sql)
	if err != nil {
		log4plus.Error("%s update Failed err=[%s] taskId=[%s] subId=[%s] systemuuid=[%s] publishTime=[%s]",
			funName, err.Error(), taskID, subID, systemUUID, publishTime)
		return err
	}
	return nil
}

func (p *TaskDB) SetSubTaskStart(taskID, subID, systemUUID string, estimateTime int64) error {
	funName := "SetSubTaskStart"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect", funName)
		log4plus.Error(errString)
		return errors.New(errString)
	}
	sql := fmt.Sprintf(`update t_sub_jobs set f_estimate_time=%d, f_start_time=NOW(), f_state=%d where f_task_id='%s' and f_sub_id='%s';`,
		estimateTime, int(DealingStatus), taskID, subID)
	_, err := p.mysqlDb.Mysqldb.Exec(sql)
	if err != nil {
		log4plus.Error("%s update Failed err=[%s] taskId=[%s] subId=[%s] systemuuid=[%s]", funName, err.Error(), taskID, subID, systemUUID)
		return err
	}
	return nil
}

func (p *TaskDB) SubTaskResult(taskID, subID, startTime, endTime, output string, duration int64, urls []string, ossDurations []int64, ossSizes []int64) {
	funName := "SetSubTaskCompletion"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect", funName)
		log4plus.Error(errString)
		return
	}
	imageUrls := strings.Join(urls, "\n")
	var ossDurs []string
	for _, v := range ossDurations {
		ossDurs = append(ossDurs, strconv.Itoa(int(v)))
	}
	var imageSizes []string
	for _, v := range ossSizes {
		imageSizes = append(imageSizes, strconv.Itoa(int(v)))
	}
	durs := strings.Join(ossDurs, "\n")
	sizes := strings.Join(imageSizes, "\n")
	sql := fmt.Sprintf(`update t_sub_jobs as a, t_jobs as b set a.f_start_time='%s', 
                                        a.f_end_time='%s', 
                                        a.f_urls='%s', 
                                        a.f_output=?, 
                                        a.f_state=%d, 
                                        a.f_oss_duration='%s', 
                                        a.f_oss_size='%s',
                                        a.f_inference_duration=%d, 
                                        b.f_duration=b.f_duration + %d 
                                    where a.f_task_id='%s' and a.f_sub_id='%s' and a.f_task_id=b.f_task_id;`,
		startTime, endTime, imageUrls, int(DealCompleted), durs, sizes, duration, duration, taskID, subID)
	_, err := p.mysqlDb.Mysqldb.Exec(sql, output)
	if err != nil {
		log4plus.Error("%s update Failed err=[%s] taskId=[%s] subId=[%s]", funName, err.Error(), taskID, subID)
		return
	}
}

func (p *TaskDB) SetMeasureTaskFailed(taskID, subID string, startTime, endTime string, output string, duration int64, failureReason string) {
	funName := "SetMeasureTaskFailed"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect", funName)
		log4plus.Error(errString)
		return
	}
	sql := fmt.Sprintf(`update t_sub_jobs set f_state=%d, f_output=?, f_start_time='%s', f_end_time='%s', f_failure_reason='%s', f_inference_duration=%d 
                  where f_task_id='%s' and f_sub_id='%s';`,
		int(DealFailed), startTime, endTime, failureReason, duration, taskID, subID)
	_, err := p.mysqlDb.Mysqldb.Exec(sql, output)
	if err != nil {
		log4plus.Error("%s update Failed err=[%s] taskId=[%s] subId=[%s]", funName, err.Error(), taskID, subID)
		return
	}
}

func (p *TaskDB) SetSubTaskFailed(systemUUID, taskID, subID string, startTime, endTime string, duration int64, failureReason string) {
	funName := "SetSubTaskFailed"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect", funName)
		log4plus.Error(errString)
		return
	}
	sql := fmt.Sprintf(`select IFNULL(f_id,-1), IFNULL(f_retry_count, 0), IFNULL(f_error_nodes, '') from t_sub_jobs where f_task_id='%s' and f_sub_id='%s';`,
		taskID, subID)
	rows, err := p.mysqlDb.Mysqldb.Query(sql)
	if err != nil {
		errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, err.Error(), sql)
		log4plus.Error(errString)
		return
	}
	defer rows.Close()

	tmpId := -1
	tmpRetry := 0
	tmpErrorNodes := ""
	for rows.Next() {
		scanErr := rows.Scan(&tmpId, &tmpRetry, &tmpErrorNodes)
		if scanErr != nil {
			log4plus.Error("%s Scan Error=[%s]", funName, scanErr.Error())
			continue
		}
	}
	if tmpId == -1 {
		errString := fmt.Sprintf("%s not found taskID=[%s] subID=[%s]", funName, taskID, subID)
		log4plus.Error(errString)
		return
	}
	tmpRetry++
	if tmpRetry == 1 {
		tmpErrorNodes = systemUUID
	} else {
		tmpErrorNodes = fmt.Sprintf("%s,%s", tmpErrorNodes, systemUUID)
	}
	sql = fmt.Sprintf(`update t_sub_jobs set f_retry_count='%d', f_error_nodes='%s', f_state=%d, f_end_time='%s', f_failure_reason='%s' 
                  where f_task_id='%s' and f_sub_id='%s';`,
		tmpRetry, tmpErrorNodes, int(DealFailed), endTime, failureReason, taskID, subID)
	_, err = p.mysqlDb.Mysqldb.Exec(sql)
	if err != nil {
		log4plus.Error("%s update Failed err=[%s] taskId=[%s] subId=[%s]", funName, err.Error(), taskID, subID)
		return
	}
}

func (p *TaskDB) GetSubTaskFailed(taskID, subID string) (error, int, string) {
	funName := "GetSubTaskFailed"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect ---->>>>", funName)
		log4plus.Error(errString)
		return errors.New(errString), 0, ""
	}
	sql := fmt.Sprintf(`select IFNULL(f_id,-1), IFNULL(f_retry_count, 0), IFNULL(f_error_nodes, '') from t_sub_jobs where f_task_id='%s' and f_sub_id='%s';`,
		taskID, subID)
	rows, err := p.mysqlDb.Mysqldb.Query(sql)
	if err != nil {
		errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, err.Error(), sql)
		log4plus.Error(errString)
		return errors.New(errString), 0, ""
	}
	defer rows.Close()

	tmpId := -1
	tmpRetry := 0
	tmpErrorNodes := ""
	for rows.Next() {
		scanErr := rows.Scan(&tmpId, &tmpRetry, &tmpErrorNodes)
		if scanErr != nil {
			log4plus.Error("%s Scan Error=[%s]", funName, scanErr.Error())
			continue
		}
	}
	if tmpId == -1 {
		errString := fmt.Sprintf("%s not found taskID=[%s] subID=[%s]", funName, taskID, subID)
		log4plus.Error(errString)
		return errors.New(errString), 0, ""
	}
	return nil, tmpRetry, tmpErrorNodes
}

type (
	HistorySubTask struct {
		TaskID       string `json:"taskID"`
		SubID        string `json:"subID"`
		StartTime    string `json:"startTime"`
		EndTime      string `json:"endTime"`
		EstimateTime string `json:"estimateTime"`
		Input        string `json:"input"`
		SystemUUID   string `json:"systemUUID"`
		State        int    `json:"state"`
		WorkspaceID  string `json:"workspaceID"`
		PublishTime  string `json:"publishTime"`
		CreateTime   string `json:"createTime"`
		RetryCount   int64  `json:"retryCount"`
		ErrorNodes   string `json:"errorNodes"`
		FailReason   string `json:"failReason"`
	}
	HistoryTask struct {
		TaskID                string            `json:"taskID"`
		Session               string            `json:"session"`
		PublishTime           string            `json:"publishTime"`
		WorkflowTitle         string            `json:"title"`
		Input                 string            `json:"input"`
		State                 int               `json:"state"`
		BillMode              int               `json:"billMode"`
		Duration              int64             `json:"duration"`
		SubNum                int64             `json:"subNum"`
		WorkspaceID           string            `json:"workspaceID"`
		StartTime             string            `json:"startTime"`
		Request               string            `json:"request"`
		InputImageScaledRatio int64             `json:"InputImageScaledRatio"`
		GenerateData          string            `json:"generatedData"`
		SubTasks              []*HistorySubTask `json:"subTasks"`
	}
	//任务查询的信息
	MinorTask struct {
		SubID             string `json:"subID"`
		CreateTime        string `json:"createTime"`
		PublishTime       string `json:"publishTime"`
		StartTime         string `json:"startTime"`
		EndTime           string `json:"endTime"`
		EstimateTime      int64  `json:"estimateTime"`
		SystemUUID        string `json:"systemUUID"`
		NodeIP            string `json:"nodeIP"`
		State             int    `json:"state"`
		OSSUrls           string `json:"ossUrls"`
		OSSDurations      string `json:"ossDurations"`
		OSSSizes          string `json:"ossSizes"`
		InferenceDuration int64  `json:"inferenceDuration"`
	}
	MajorTask struct {
		TaskID         string      `json:"taskID"`
		PublishTime    string      `json:"publishTime"`
		StartTime      string      `json:"startTime"`
		CompletionTime string      `json:"completionTime"`
		State          int         `json:"state"`
		Duration       int64       `json:"duration"`
		WorkspaceID    string      `json:"workspaceID"`
		SubNum         int64       `json:"subNum"`
		Title          string      `json:"title"`
		Minors         []MinorTask `json:"minors"`
	}
)

func (p *TaskDB) GetHistoryTasks(previousDay time.Time) (err error, tasks []*HistoryTask) {
	funName := "GetHistoryTasks"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect", funName)
		log4plus.Error(errString)
		return errors.New(errString), nil
	}
	sql := fmt.Sprintf(`select IFNULL(f_id,-1), 
       IFNULL(f_task_id,''), 
       IFNULL(f_session,''), 
       IFNULL(f_publish_time,''), 
       IFNULL(f_title,''), 
       IFNULL(f_input,''), 
       IFNULL(f_state, 0), 
       IFNULL(f_bill_mode, 0), 
       IFNULL(f_sub_num, 0), 
       IFNULL(f_workspace_id, ''), 
       IFNULL(f_generated_data, ''), 
       IFNULL(f_start_time, ''),
       IFNULL(f_request, '') from t_jobs where 
        f_publish_time>'%s' and 
        f_session <> '00000000' and 
        f_delete = 0 and 
        (f_state <> 3 && f_state <> 4) ;`,
		previousDay.Format("2006-01-02 15:04:05"))
	rows, err := p.mysqlDb.Mysqldb.Query(sql)
	if err != nil {
		errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, err.Error(), sql)
		log4plus.Error(errString)
		return err, nil
	}
	defer rows.Close()
	for rows.Next() {
		task := HistoryTask{}
		var tmpId int
		scanErr := rows.Scan(&tmpId,
			&task.TaskID,
			&task.Session,
			&task.PublishTime,
			&task.WorkflowTitle,
			&task.Input,
			&task.State,
			&task.BillMode,
			&task.SubNum,
			&task.WorkspaceID,
			&task.GenerateData,
			&task.StartTime,
			&task.Request)
		if scanErr != nil {
			log4plus.Error("%s Scan Error=[%s]", funName, scanErr.Error())
			continue
		}
		tasks = append(tasks, &task)
	}
	return nil, tasks
}

func (p *TaskDB) GetHistorySubTasks(tasks []*HistoryTask) error {
	funName := "GetHistorySubTasks"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect", funName)
		log4plus.Error(errString)
		return errors.New(errString)
	}
	for _, task := range tasks {
		sql := fmt.Sprintf(`select IFNULL(f_id,-1), 
       IFNULL(f_sub_id,''), 
       IFNULL(f_start_time,''), 
       IFNULL(f_end_time,''), 
       IFNULL(f_estimate_time,''), 
       IFNULL(f_input,''), 
       IFNULL(f_system_uuid,''), 
       IFNULL(f_state, 0),  
       IFNULL(f_workspace_id, ''), 
       IFNULL(f_publish_time,''), 
       IFNULL(f_create_time,''),
       IFNULL(f_retry_count, 0),
       IFNULL(f_error_nodes, ''),
       IFNULL(f_failure_reason, '') from t_sub_jobs where 
        f_task_id = '%s' and 
        f_delete = 0 and 
        (f_state <> 3 && f_state <> 4) order by f_sub_id;`, task.TaskID)
		rows, err := p.mysqlDb.Mysqldb.Query(sql)
		if err != nil {
			errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, err.Error(), sql)
			log4plus.Error(errString)
			return err
		}
		defer rows.Close()
		for rows.Next() {
			subTask := HistorySubTask{}
			var tmpId int
			scanErr := rows.Scan(&tmpId,
				&subTask.SubID,
				&subTask.StartTime,
				&subTask.EndTime,
				&subTask.EstimateTime,
				&subTask.Input,
				&subTask.SystemUUID,
				&subTask.State,
				&subTask.WorkspaceID,
				&subTask.PublishTime,
				&subTask.CreateTime,
				&subTask.RetryCount,
				&subTask.ErrorNodes,
				&subTask.FailReason)
			if scanErr != nil {
				log4plus.Error("%s Scan Error=[%s]", funName, scanErr.Error())
				continue
			}
			task.SubTasks = append(task.SubTasks, &subTask)
		}
	}
	return nil
}

func (p *TaskDB) SetSubTaskDelete(taskID string) error {
	funName := "SetSubTaskDelete"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect", funName)
		log4plus.Error(errString)
		return errors.New(errString)
	}
	sql := fmt.Sprintf(`update t_jobs, t_sub_jobs set t_jobs.f_delete=1, t_sub_jobs.f_delete=1 where t_jobs.f_task_id=t_sub_jobs.f_task_id and t_jobs.f_task_id='%s';`,
		taskID)
	_, err := p.mysqlDb.Mysqldb.Exec(sql)
	if err != nil {
		log4plus.Error("%s update Failed err=[%s] taskId=[%s]", funName, err.Error(), taskID)
		return err
	}
	return nil
}

func (p *TaskDB) LoadHistoryTasks(previousDay time.Time) []*HistoryTask {
	funName := "LoadHistoryTasks"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect", funName)
		log4plus.Error(errString)
		return nil
	}
	if err, tasks := p.GetHistoryTasks(previousDay); err != nil {
		log4plus.Error("%s GetHistoryTasks failed err=[%s] previousDay=[%s]", funName, err.Error(), previousDay.Format("2006-01-02 15:04:05"))
		return nil
	} else {
		if err = p.GetHistorySubTasks(tasks); err != nil {
			log4plus.Error("%s GetHistorySubTasks failed err=[%s] previousDay=[%s]", funName, err.Error(), previousDay.Format("2006-01-02 15:04:05"))
			return nil
		}
		return tasks
	}
}

func (p *TaskDB) GetTaskID(taskID string) (error, MajorTask) {
	funName := "GetTaskID"
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect", funName)
		log4plus.Error(errString)
		return errors.New(errString), MajorTask{}
	}
	//得到主任务
	sql := fmt.Sprintf(`select IFNULL(f_id,-1), 
       IFNULL(f_title,''),
       IFNULL(f_publish_time,''),
       IFNULL(f_start_time,''),
       IFNULL(f_completion_time,''),
       IFNULL(f_state,''),
       IFNULL(f_duration,0),
       IFNULL(f_workspace_id,''),
       IFNULL(f_sub_num,0) from t_jobs where f_task_id='%s';`, taskID)
	rows, err := p.mysqlDb.Mysqldb.Query(sql)
	if err != nil {
		errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, err.Error(), sql)
		log4plus.Error(errString)
		return err, MajorTask{}
	}
	defer rows.Close()

	major := MajorTask{
		TaskID: taskID,
	}
	exist := false
	for rows.Next() {
		var tmpId int
		scanErr := rows.Scan(&tmpId,
			&major.Title,
			&major.PublishTime,
			&major.StartTime,
			&major.CompletionTime,
			&major.State,
			&major.Duration,
			&major.WorkspaceID,
			&major.SubNum)
		if scanErr != nil {
			log4plus.Error("%s Scan Error=[%s]", funName, scanErr.Error())
			continue
		}
		if tmpId != -1 {
			exist = true
		}

	}

	if !exist {
		return errors.New("not found taskID"), MajorTask{}
	}
	//得到子任务
	sql = fmt.Sprintf(`select IFNULL(a.f_id,-1), 
       IFNULL(a.f_sub_id,''),
       IFNULL(a.f_create_time,''),
       IFNULL(a.f_start_time,''),
       IFNULL(a.f_end_time,''),
       IFNULL(a.f_estimate_time,0),
       IFNULL(a.f_system_uuid,''),
       IFNULL(b.f_node_ip,''),
       IFNULL(a.f_state,0),
       IFNULL(a.f_publish_time,''),
       IFNULL(a.f_urls,''),
       IFNULL(a.f_oss_duration,''),
       IFNULL(a.f_oss_size,''),
       IFNULL(a.f_inference_duration,0) from t_sub_jobs as a, t_nodes as b where a.f_system_uuid=b.f_system_uuid and a.f_task_id='%s';`, taskID)
	rowsSubID, err := p.mysqlDb.Mysqldb.Query(sql)
	if err != nil {
		errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, err.Error(), sql)
		log4plus.Error(errString)
		return err, MajorTask{}
	}
	defer rowsSubID.Close()

	for rowsSubID.Next() {
		var tmpId int
		var minor MinorTask
		scanErr := rowsSubID.Scan(&tmpId,
			&minor.SubID,
			&minor.CreateTime,
			&minor.StartTime,
			&minor.EndTime,
			&minor.EstimateTime,
			&minor.SystemUUID,
			&minor.NodeIP,
			&minor.State,
			&minor.PublishTime,
			&minor.OSSUrls,
			&minor.OSSDurations,
			&minor.OSSSizes,
			&minor.InferenceDuration)
		if scanErr != nil {
			log4plus.Error("%s Scan Error=[%s]", funName, scanErr.Error())
			continue
		}
		if tmpId != -1 {
			major.Minors = append(major.Minors, minor)
		}
	}
	return nil, major
}

func SingletonTaskDB() *TaskDB {
	if gTaskDB == nil {
		log4plus.Info("SingletonTaskDB ---->>>>")
		gTaskDB = &TaskDB{}
		if gTaskDB.mysqlDb = NewMysql(configure.SingletonConfigure().Mysql.MysqlIp,
			configure.SingletonConfigure().Mysql.MysqlPort,
			configure.SingletonConfigure().Mysql.MysqlDBName,
			configure.SingletonConfigure().Mysql.MysqlDBCharset,
			configure.SingletonConfigure().Mysql.UserName,
			configure.SingletonConfigure().Mysql.Password); gTaskDB.mysqlDb == nil {
			log4plus.Error("SingletonTaskDB NewMysql Failed")
			return nil
		}
	}
	return gTaskDB
}
