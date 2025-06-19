package plugIn

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/nextGPU/ng-client/header"
	ws "github.com/nextGPU/ng-client/net/websocket"
	log4plus "github.com/nextGPU/include/log4go"
	"github.com/google/uuid"
	"github.com/jinzhu/copier"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type DockerInterface interface {
	GetClient() *ws.WSClient
	GetSystemUUID() string
	GetMeasureID() string

	SetState(state header.NodeStatus)
	RenameFile(saveDir, subFolder, fileName string) (error, string)
	PushOSS(ossDir, session, imagePath, filePath string) (string, int64, int64, error)
	SendPublishTask(workspaceID, subID, session string, success bool, reason string, allSlots []header.Slot)
	SendMeasureTask(workspaceID, subID, session string, success bool, reason string, allSlots []header.Slot)
	SendMeasureFailed(workspaceID, subID, failureReason string, resultData []byte, slot *header.Slot)
	SendMeasureComplete(workspaceID, subID string, outputs []*header.OSSOutput, resultData []byte, slot *header.Slot)
	SendTaskFailed(workspaceID, subID, failureReason string, resultData []byte, allSlots []header.Slot)
	SendTaskCompleted(workspaceID, subID string, outputs []*header.OSSOutput, resultData []byte, allSlots []header.Slot)
	SendComfyUIStartupSuccess(workSpaceID string)
	SendTaskStart(workspaceID, subID string, allSlots []header.Slot)
	SendMeasureStart(workspaceID, subID string, allSlots []header.Slot)
	SendInferenceCompleted(workspaceID, subID string)
}

type ComfyUI struct {
	queueLock     sync.Mutex
	queue         []*header.Slot
	curNodeID     string
	clientID      string
	WorkspaceID   string
	node          DockerInterface
	ComfyUIHttp   string
	ComfyUIWS     string
	SaveDir       string
	ComfyUILinker *ws.Linker
	OfflineCh     chan bool
	ExitCh        chan struct{}
	RecvMessageCh chan []byte
	SendMessageCh chan []byte
}

// OnEvent
func (n *ComfyUI) OnConnect(linker *ws.Linker, param string, Ip string, port string) {
	n.ComfyUILinker = linker
}

func (n *ComfyUI) OnRead(linker *ws.Linker, param string, message []byte) error {
	n.RecvMessageCh <- message
	return nil
}

func (n *ComfyUI) OnDisconnect(linker *ws.Linker, param string, Ip string, port string) {
	n.OfflineCh <- true
}

func (n *ComfyUI) Exist(param string) bool {
	return true
}

// slot
func (n *ComfyUI) addSlot(slot *header.Slot) {
	funName := "addSlot"
	n.queueLock.Lock()
	defer n.queueLock.Unlock()

	slot.InQueueTime = time.Now().UnixMilli()
	slot.InQueuePos = int64(len(n.queue))
	log4plus.Info(fmt.Sprintf("%s---->>>> subID=[%s]", funName, slot.SubID))
	n.queue = append(n.queue, slot)
	for index, value := range n.queue {
		value.CurQueuePos = int64(index)
	}
}

func (n *ComfyUI) delSlot(subID string) {
	n.queueLock.Lock()
	defer n.queueLock.Unlock()

	var newQueue []*header.Slot
	for _, value := range n.queue {
		if !strings.EqualFold(value.SubID, subID) {
			newQueue = append(newQueue, value)
		}
	}
	n.queue = newQueue
	for index, value := range n.queue {
		value.CurQueuePos = int64(index)
	}
}

func (n *ComfyUI) findSlot(promptID string) *header.Slot {
	n.queueLock.Lock()
	defer n.queueLock.Unlock()

	for _, value := range n.queue {
		if strings.EqualFold(promptID, value.PromptID) {
			return value
		}
	}
	return nil
}

func (n *ComfyUI) setStartTime(promptID string, startTime int64) {
	funName := "setStartTime"
	n.queueLock.Lock()
	defer n.queueLock.Unlock()

	log4plus.Info(fmt.Sprintf("%s---->>>> promptID=[%s] startTime=[%d]", funName, promptID, startTime))
	for _, value := range n.queue {
		if strings.EqualFold(promptID, value.PromptID) {
			value.StartTime = startTime
		}
	}
}

func (n *ComfyUI) getStartTime(promptID string) int64 {
	n.queueLock.Lock()
	defer n.queueLock.Unlock()

	for _, value := range n.queue {
		if strings.EqualFold(promptID, value.PromptID) {
			return value.StartTime
		}
	}
	return 0
}

func (n *ComfyUI) setEndTime(promptID string, endTime int64) {
	funName := "setEndTime"
	n.queueLock.Lock()
	defer n.queueLock.Unlock()

	log4plus.Info(fmt.Sprintf("%s---->>>> promptID=[%s] endTime=[%d]", funName, promptID, endTime))
	for _, value := range n.queue {
		if strings.EqualFold(promptID, value.PromptID) {
			value.EndTime = endTime
		}
	}
}

func (n *ComfyUI) getSlot() []header.Slot {
	n.queueLock.Lock()
	defer n.queueLock.Unlock()
	var allSlot []header.Slot
	for _, v := range n.queue {
		slot := header.Slot{
			SubID:       v.SubID,
			State:       v.State,
			Session:     v.Session,
			PromptID:    v.PromptID,
			InQueueTime: v.InQueueTime,
			InQueuePos:  v.InQueuePos,
			CurQueuePos: v.CurQueuePos,
			StartTime:   v.StartTime,
			EndTime:     v.EndTime,
		}
		allSlot = append(allSlot, slot)
	}
	return allSlot
}

func (n *ComfyUI) getPos(subID string) int {
	n.queueLock.Lock()
	defer n.queueLock.Unlock()
	for index, value := range n.queue {
		if strings.EqualFold(value.SubID, subID) {
			return index
		}
	}
	return -1
}

// recv message
func (n *ComfyUI) NewTask(msg *header.PublishMessage) {
	funName := "NewTask"
	var data header.RequestData
	if err := json.Unmarshal(msg.Data, &data); err == nil {
		errSend, promptID := n.sendWorkflow(msg.SubID, data.Data)
		if errSend != nil {
			log4plus.Error(fmt.Sprintf("%s sendWorkflow err=[%s]", funName, errSend.Error()))
			n.node.SendPublishTask(msg.WorkSpaceID, msg.SubID, msg.Session, false, errSend.Error(), nil)
		}
		slot := &header.Slot{
			SubID:     msg.SubID,
			State:     header.Running,
			Session:   msg.Session,
			ImagePath: msg.ImagePath,
			PromptID:  promptID,
		}
		_ = copier.Copy(&slot.TaskData, &data)
		n.addSlot(slot)

		log4plus.Info(fmt.Sprintf("%s sendWorkflow success promptID=[%s]", funName, promptID))
		allSlots := n.getSlot()
		n.node.SendPublishTask(msg.WorkSpaceID, msg.SubID, msg.Session, true, "", allSlots)
	}
}

func (n *ComfyUI) NewMeasureTask(msg *header.MeasureMessage) {
	funName := "NewMeasureTask"
	errSend, promptID := n.sendWorkflow(msg.SubID, msg.Data)
	if errSend != nil {
		log4plus.Error(fmt.Sprintf("%s sendWorkflow err=[%s]", funName, errSend.Error()))
		n.node.SendMeasureTask(msg.WorkSpaceID, msg.SubID, "", false, errSend.Error(), nil)
	}
	measurePath := fmt.Sprintf("meausre/%s", time.Now().Format("20060102"))
	slot := &header.Slot{
		SubID:     msg.SubID,
		State:     header.Running,
		Session:   msg.SubID,
		ImagePath: measurePath,
		PromptID:  promptID,
	}
	_ = copier.Copy(&slot.TaskData, &msg.Data)
	n.addSlot(slot)
	log4plus.Info(fmt.Sprintf("%s sendWorkflow success promptID=[%s]", funName, promptID))
	allSlot := n.getSlot()
	n.node.SendMeasureTask(msg.WorkSpaceID, msg.SubID, "", true, "", allSlot)
}

func (n *ComfyUI) startComfyUIWS(comfyUI string) {
	if linker := n.node.GetClient().StartClient(comfyUI, n.node.GetSystemUUID(), n); linker != nil {
		n.ComfyUIWS = comfyUI
	}
}

func (n *ComfyUI) reconnect() {
	if n.ComfyUILinker == nil || n.ComfyUILinker.State == ws.Disconnected {
		wsComfyUIAddr := fmt.Sprintf("%s/ws?clientId=%s", n.ComfyUIWS, n.clientID)
		if linker := n.node.GetClient().StartClient(wsComfyUIAddr, n.node.GetSystemUUID(), n); linker != nil {
			n.ComfyUILinker = linker
		}
	}
}

func (n *ComfyUI) offline() {
	if n.ComfyUILinker != nil {
		_ = n.ComfyUILinker.Close()
		n.ComfyUILinker = nil
	}
}

// send message
func (n *ComfyUI) httpNewTask(payload []byte) (error, string) {
	funName := "httpNewTask"
	now := time.Now().UnixMilli()
	defer func() {
		log4plus.Info(fmt.Sprintf("%s consumption time=%d(ms)", funName, time.Now().UnixMilli()-now))
	}()
	url := fmt.Sprintf("%s/prompt", n.ComfyUIHttp)
	log4plus.Info(fmt.Sprintf("%s parse url=[%s]", funName, url))
	client := &http.Client{
		Transport: &http.Transport{
			Dial: func(netw, addr string) (net.Conn, error) {
				c, err := net.DialTimeout(netw, addr, time.Minute*10)
				if err != nil {
					log4plus.Error(fmt.Sprintf("%s dail timeout err=%s", funName, err.Error()))
					return nil, err
				}
				return c, nil
			},
			MaxIdleConnsPerHost:   10,
			ResponseHeaderTimeout: time.Minute * 10,
		},
	}
	defer client.CloseIdleConnections()

	bodyBytes := bytes.NewReader(payload)
	request, err := http.NewRequest("POST", url, bodyBytes)
	if err != nil {
		log4plus.Error(fmt.Sprintf("%s http.NewRequest Failed url=[%s] err=[%s]", funName, url, err.Error()))
		return err, ""
	}
	response, err := client.Do(request)
	if err != nil {
		log4plus.Error(fmt.Sprintf("%s client.Do Failed url=[%s] err=[%s]", funName, url, err.Error()))
		return err, ""
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		log4plus.Error(fmt.Sprintf("%s ioutil.ReadAll Failed url=[%s] err=[%s]", funName, url, err.Error()))
		return err, ""
	}
	log4plus.Info(fmt.Sprintf("%s Check StatusCode response.StatusCode=[%d]", funName, response.StatusCode))
	if response.StatusCode != 200 {
		log4plus.Error(fmt.Sprintf("%s client.Do url=[%s] response.StatusCode=[%d] responseBody=[%s]", funName, url, response.StatusCode, string(responseBody)))
		return err, ""
	}
	var result map[string]interface{}
	if err := json.Unmarshal(responseBody, &result); err != nil {
		log4plus.Error(fmt.Sprintf("%s client.Do url=[%s] response.StatusCode=[%d] responseBody=[%s]", funName, url, response.StatusCode, string(responseBody)))
		return err, ""
	}
	promptID := result["prompt_id"].(string)
	log4plus.Info(fmt.Sprintf("%s Workflow Result promptID=[%s]", funName, promptID))
	return nil, promptID
}

func (n *ComfyUI) httpHistory(promptID string) (err error, responseBody []byte) {
	funName := "httpHistory"
	url := fmt.Sprintf("%s/history/%s", n.ComfyUIHttp, promptID)
	log4plus.Info(fmt.Sprintf("%s parse url=[%s]", funName, url))
	client := &http.Client{
		Transport: &http.Transport{
			Dial: func(netw, addr string) (net.Conn, error) {
				c, errTimeout := net.DialTimeout(netw, addr, time.Minute*10)
				if errTimeout != nil {
					log4plus.Error(fmt.Sprintf("%s dail timeout err=%s", funName, errTimeout.Error()))
					return nil, errTimeout
				}
				return c, nil
			},
			MaxIdleConnsPerHost:   10,
			ResponseHeaderTimeout: time.Minute * 10,
		},
	}
	defer client.CloseIdleConnections()

	var request *http.Request
	request, err = http.NewRequest("GET", url, nil)
	if err != nil {
		log4plus.Error(fmt.Sprintf("%s http.NewRequest Failed url=[%s] err=[%s]", funName, url, err.Error()))
		return err, []byte("")
	}

	var response *http.Response
	response, err = client.Do(request)
	if err != nil {
		log4plus.Error(fmt.Sprintf("%s client.Do Failed url=[%s] err=[%s]", funName, url, err.Error()))
		return err, []byte("")
	}
	defer response.Body.Close()

	responseBody, err = io.ReadAll(response.Body)
	if err != nil {
		log4plus.Error(fmt.Sprintf("%s ioutil.ReadAll Failed url=[%s] err=[%s]", funName, url, err.Error()))
		return err, []byte("")
	}

	log4plus.Info(fmt.Sprintf("%s response.StatusCode=[%d] responseBody=[%s]", funName, response.StatusCode, string(responseBody)))
	if response.StatusCode != 200 {
		log4plus.Error(fmt.Sprintf("%s client.Do url=[%s] response.StatusCode=[%d] responseBody=[%s]", funName, url, response.StatusCode, string(responseBody)))
		return err, []byte("")
	}
	return nil, responseBody
}

func (n *ComfyUI) sendWorkflow(subID string, data []byte) (error, string) {
	funName := "sendWorkflow"
	var workflow map[string]interface{}
	if err := json.Unmarshal(data, &workflow); err != nil {
		errString := fmt.Sprintf("%s json.Unmarshal Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return err, ""
	}
	var payloadBuf bytes.Buffer
	payloadEncoder := json.NewEncoder(&payloadBuf)
	payloadEncoder.SetEscapeHTML(false)
	_ = payloadEncoder.Encode(map[string]interface{}{
		"prompt":    workflow,
		"client_id": n.clientID,
	})
	payload := payloadBuf.Bytes()
	log4plus.Info(fmt.Sprintf("%s httpNewTask---->>>> payload=[%s]", funName, string(payload)))
	if err, promptID := n.httpNewTask(payload); err == nil {
		return nil, promptID
	} else {
		errString := fmt.Sprintf("%s httpWorkflow Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return err, ""
	}
}

type (
	ImageMeta struct {
		Filename  string `json:"filename"`
		Subfolder string `json:"subfolder"`
		Type      string `json:"type"` // 输出类型（如 "output", "temp"）
	}
	Output struct {
		NodeID string      `json:"nodeID"`
		Images []ImageMeta `json:"images"`
	}
	Outputs map[string]struct {
		Error       string      `json:"error"`        // 全局错误信息
		Images      []ImageMeta `json:"images"`       // 生成的图像元数据
		LatentShape []int       `json:"latent_shape"` // Latent空间维度
	}
	Status struct {
		StatusStr string          `json:"status_str"`
		Completed bool            `json:"completed"`
		Messages  [][]interface{} `json:"messages"`
	}
	HistoryResponse struct {
		Status     *Status           `json:"status,omitempty"`      // 任务状态（success/failed）
		Outputs    *Outputs          `json:"outputs,omitempty"`     // 输出数据
		NodeErrors map[string]string `json:"node_errors,omitempty"` // 节点级错误
	}
)

func parsePromptID(data []byte) []map[string]any {
	funName := "parseWorkflowResult"
	var results map[string]any
	if err := json.Unmarshal(data, &results); err != nil {
		errString := fmt.Sprintf("%s json.Unmarshal Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return nil
	}
	var outputs []map[string]any
	for uuid, value := range results {
		outerMap, ok := value.(map[string]any)
		if !ok {
			errString := fmt.Sprintf("%s Invalid structure uuid=[%s]", funName, uuid)
			log4plus.Error(errString)
			return nil
		}
		outputs = append(outputs, outerMap)
	}
	return outputs
}

func parseHistory(data []byte) (error, *Status, *Outputs) {
	funName := "consumption"
	var history HistoryResponse
	if err := json.Unmarshal(data, &history); err != nil {
		errString := fmt.Sprintf("%s json.Unmarshal Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return errors.New(errString), nil, nil
	}
	return nil, history.Status, history.Outputs
}

func parseDuration(startTime, endTime int64, status *Status) (int64, int64, int64) {
	if status == nil {
		return startTime, endTime, endTime - startTime
	} else {
		startTimestamp := int64(0)
		endTimestamp := int64(0)
		for _, message := range status.Messages {
			if len(message) >= 2 && message[0] == "execution_start" {
				dataMap := message[1].(map[string]interface{})
				startTimestamp = int64(dataMap["timestamp"].(float64))
			} else if len(message) >= 2 && message[0] == "execution_success" {
				dataMap := message[1].(map[string]interface{})
				endTimestamp = int64(dataMap["timestamp"].(float64))
			}
		}
		return startTimestamp, endTimestamp, endTimestamp - startTimestamp
	}
}

func checkSuccessStatus(status *Status) bool {
	if status != nil {
		if strings.EqualFold(status.StatusStr, "success") {
			return true
		} else if strings.EqualFold(status.StatusStr, "failed") {
			return false
		}
	}
	return true
}

func (n *ComfyUI) pushMeasureFile(promptID, session, subID, imagePath string, body []byte) {
	funName := "pushMeasureFile"
	log4plus.Info(fmt.Sprintf("%s session=[%s] subID=[%s] ", funName, session, subID))
	var ossOutputs []*header.OSSOutput
	bodyRaw := parsePromptID(body)
	for _, v := range bodyRaw {
		bodyData, _ := json.Marshal(v)
		log4plus.Info(fmt.Sprintf("%s --->>> bodyData=[%s]", funName, string(bodyData)))

		err, status, outputs := parseHistory(bodyData)
		if outputs == nil {
			errString := fmt.Sprintf("%s parseHistory failed result outputs is nil", funName)
			log4plus.Error(errString)
			continue
		}
		if status != nil {
			log4plus.Info(fmt.Sprintf("%s status=[%#v] outputs=[%#v] subID=[%s]", funName, *status, *outputs, subID))
		}
		if err == nil {
			if !checkSuccessStatus(status) {
				//failed
				errString := fmt.Sprintf("%s ____>>>> checkSuccessStatus failed subID=[%s] promptID=[%s]", funName, subID, promptID)
				log4plus.Error(errString)
				slot := n.findSlot(promptID)
				if slot != nil {
					n.node.SendMeasureFailed(n.WorkspaceID, subID, errString, body, slot)
					n.delSlot(slot.SubID)
				}
				return
			} else {
				//success
				var ossOutput header.OSSOutput
				failString := ""
				success := true
				for _, output := range *outputs {
					for _, image := range output.Images {
						errRaName, newFile := n.node.RenameFile(n.SaveDir, image.Subfolder, image.Filename)
						if errRaName != nil {
							errString := fmt.Sprintf("%s renameFile failed saveDir=[%s] subfolder=[%s] filename=[%s] errRaname=[%s]",
								funName, n.SaveDir, image.Subfolder, image.Filename, errRaName.Error())
							log4plus.Error(errString)
							success = false
							failString = errString
							continue
						}
						// push oss
						log4plus.Info(fmt.Sprintf("%s pushOSS session=[%s] newFile=[%s]", funName, session, newFile))
						if ossUrl, ossDuration, fileSize, errPush := n.node.PushOSS("", session, imagePath, newFile); errPush == nil {
							ossOutput.Images = append(ossOutput.Images, header.OssImage{
								OSSUrl:      ossUrl,
								OSSDuration: ossDuration,
								OSSSize:     fileSize,
							})
							//delete file
							log4plus.Info(fmt.Sprintf("%s Remove newFile=[%s]", funName, newFile))
							if errRemove := os.Remove(newFile); errRemove != nil {
								errString := fmt.Sprintf("%s Remove failed newFile=[%s] err=[%s]", funName, newFile, errRemove.Error())
								log4plus.Error(errString)
								success = false
								failString = errString
							}
						} else {
							errString := fmt.Sprintf("%s pushOSS Failed session=[%s] newFile=[%s] errPush=[%s]",
								funName, session, newFile, errPush.Error())
							log4plus.Error(errString)
							success = false
							failString = errString
						}
					}
				}
				if len(ossOutput.Images) > 0 {
					ossOutputs = append(ossOutputs, &ossOutput)
				}
				slot := n.findSlot(promptID)
				if slot != nil {
					//duration
					startTime, endTime, duration := parseDuration(slot.StartTime, slot.EndTime, status)
					if startTime > endTime {
						log4plus.Error(fmt.Sprintf("%s ____>>>> parseDuration subID=[%s] startTime=[%d] endTime=[%d] duration=[%d] promptID=[%s]",
							funName,
							subID,
							startTime,
							endTime,
							duration,
							promptID))
						startTime = endTime
						duration = 0
					}
					n.setStartTime(promptID, startTime)
					n.setEndTime(promptID, endTime)
					n.node.SetState(header.MeasurementComplete)
					log4plus.Info(fmt.Sprintf("%s ____>>>> parseDuration subID=[%s] startTime=[%d] endTime=[%d] duration=[%d] promptID=[%s]",
						funName,
						subID,
						startTime,
						endTime,
						duration,
						promptID))
					if success && len(ossOutputs) > 0 {
						n.node.SendMeasureComplete(n.WorkspaceID, subID, ossOutputs, body, slot)
					} else {
						n.node.SendMeasureFailed(n.WorkspaceID, subID, failString, body, slot)
					}
					n.delSlot(subID)
				}
			}
		}
	}
}

func (n *ComfyUI) pushAiTaskFile(session, promptID, subID, imagePath string, allSlots []header.Slot, body []byte) {
	funName := "pushAiTaskFile"
	log4plus.Info(fmt.Sprintf("%s session=[%s] subID=[%s] body=[%s]", funName, session, subID, string(body)))
	var ossOutputs []*header.OSSOutput
	bodyRaw := parsePromptID(body)
	for _, v := range bodyRaw {
		bodyData, _ := json.Marshal(v)
		err, status, outputs := parseHistory(bodyData)
		if outputs == nil {
			errString := fmt.Sprintf("%s parseHistory failed Result outputs is nil", funName)
			log4plus.Error(errString)
			continue
		}
		if status != nil {
			log4plus.Info(fmt.Sprintf("%s status=[%#v] outputs=[%#v] subID=[%s]", funName, *status, *outputs, subID))
		}
		if err == nil {
			if !checkSuccessStatus(status) {
				//failed
				errString := fmt.Sprintf("%s sendTaskFailed subID=[%s] promptID=[%s]", funName, subID, promptID)
				log4plus.Error(errString)
				n.node.SendTaskFailed(n.WorkspaceID, subID, errString, body, allSlots)
				n.delSlot(subID)
				return
			} else {
				//success
				var ossOutput header.OSSOutput
				failString := ""
				success := true
				for _, output := range *outputs {
					for _, image := range output.Images {
						errRaName, newFile := n.node.RenameFile(n.SaveDir, image.Subfolder, image.Filename)
						if errRaName != nil {
							errString := fmt.Sprintf("%s renameFile failed saveDir=[%s] subfolder=[%s] filename=[%s] errRaname=[%s]",
								funName, n.SaveDir, image.Subfolder, image.Filename, errRaName.Error())
							log4plus.Error(errString)
							success = false
							failString = errString
							continue
						}
						// push oss
						log4plus.Info(fmt.Sprintf("%s pushOSS session=[%s] newFile=[%s]", funName, session, newFile))
						if ossUrl, ossDuration, fileSize, errPush := n.node.PushOSS("", session, imagePath, newFile); errPush == nil {
							ossOutput.Images = append(ossOutput.Images, header.OssImage{
								OSSUrl:      ossUrl,
								OSSDuration: ossDuration,
								OSSSize:     fileSize,
							})
							//delete file
							log4plus.Info(fmt.Sprintf("%s Remove newFile=[%s]", funName, newFile))
							if errRemove := os.Remove(newFile); errRemove != nil {
								errString := fmt.Sprintf("%s Remove failed newFile=[%s] err=[%s]", funName, newFile, errRemove.Error())
								log4plus.Error(errString)
								success = false
								failString = errString
							}
						} else {
							errString := fmt.Sprintf("%s pushOSS Failed session=[%s] newFile=[%s] errPush=[%s]",
								funName, session, newFile, errPush.Error())
							log4plus.Error(errString)
							success = false
							failString = errString
						}
					}
				}
				if len(ossOutput.Images) > 0 {
					ossOutputs = append(ossOutputs, &ossOutput)
				}
				for index, value := range allSlots {
					if strings.EqualFold(value.SubID, subID) {
						startTime, endTime, _ := parseDuration(allSlots[index].StartTime, allSlots[index].EndTime, status)
						allSlots[index].StartTime = startTime
						allSlots[index].EndTime = endTime
						break
					}
				}
				if success && len(ossOutputs) > 0 {
					n.node.SendTaskCompleted(n.WorkspaceID, subID, ossOutputs, body, allSlots)
					log4plus.Info(fmt.Sprintf("%s --->>>sendTaskCompleted subID=[%s]", funName, subID))
				} else {
					n.node.SendTaskFailed(n.WorkspaceID, subID, failString, body, allSlots)
					log4plus.Info(fmt.Sprintf("%s --->>>sendTaskFailed subID=[%s]", funName, subID))
				}
				n.delSlot(subID)
			}
		}
	}
}

func (n *ComfyUI) listenProgress(buffer []byte) {
	funName := "listenProgress"
	var msg map[string]interface{}
	if err := json.Unmarshal(buffer, &msg); err != nil {
		errString := fmt.Sprintf("%s json.Unmarshal Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return
	}
	if strings.EqualFold(msg["type"].(string), "status") ||
		strings.EqualFold(msg["type"].(string), "execution_start") ||
		strings.EqualFold(msg["type"].(string), "executing") ||
		strings.EqualFold(msg["type"].(string), "progress") {
		n.node.SendComfyUIStartupSuccess(n.WorkspaceID)
		switch msg["type"] {
		case "status": //Task Change Notice
			log4plus.Info(fmt.Sprintf("%s type=[%s] buffer=[%s]", funName, msg["type"], string(buffer)))
		case "execution_start": //Task Execution Commences
			data := msg["data"].(map[string]interface{})
			promptID := data["prompt_id"].(string)
			n.setStartTime(promptID, time.Now().UnixMilli())
			allSlots := n.getSlot()
			for _, slot := range allSlots {
				if strings.EqualFold(slot.PromptID, promptID) {
					parts := strings.Split(slot.SubID, ".")
					if len(parts) <= 1 {
						log4plus.Info(fmt.Sprintf("%s sendTaskStart subID=[%s] --->>>", funName, slot.SubID))
						n.node.SendTaskStart(n.WorkspaceID, slot.SubID, allSlots)
					} else {
						if strings.EqualFold(parts[0], n.node.GetMeasureID()) {
							log4plus.Info(fmt.Sprintf("%s sendMeasureStart subID=[%s] --->>>", funName, slot.SubID))
							n.node.SendMeasureStart(n.WorkspaceID, slot.SubID, allSlots)
						}
					}
					break
				}
			}
		case "executing": //Steps of Task Execution
			log4plus.Info(fmt.Sprintf("%s buffer=[%s]", funName, string(buffer)))
			data := msg["data"].(map[string]interface{})
			if data["prompt_id"] == nil {
				return
			}
			promptID := data["prompt_id"].(string)
			if n.getStartTime(promptID) == 0 {
				n.setStartTime(promptID, time.Now().UnixMilli())
				allSlots := n.getSlot()
				for _, slot := range allSlots {
					if strings.EqualFold(slot.PromptID, promptID) {
						parts := strings.Split(slot.SubID, ".")
						if len(parts) <= 1 {
							log4plus.Info(fmt.Sprintf("%s sendTaskStart subID=[%s] --->>>", funName, slot.SubID))
							n.node.SendTaskStart(n.WorkspaceID, slot.SubID, allSlots)
						} else {
							if strings.EqualFold(parts[0], n.node.GetMeasureID()) {
								log4plus.Info(fmt.Sprintf("%s sendMeasureStart subID=[%s] --->>>", funName, slot.SubID))
								n.node.SendMeasureStart(n.WorkspaceID, slot.SubID, allSlots)
							}
						}
						break
					}
				}
			}
			if data["node"] == nil {
				n.setEndTime(promptID, time.Now().UnixMilli())
				_, body := n.httpHistory(promptID)
				slot := n.findSlot(promptID)
				if slot != nil {
					parts := strings.Split(slot.SubID, ".")
					if len(parts) <= 1 {
						//ai task
						allSlots := n.getSlot()
						log4plus.Info(fmt.Sprintf("%s pushAiTaskFile subID=[%s] Execution completed", funName, slot.SubID))
						n.node.SendInferenceCompleted(n.WorkspaceID, slot.SubID)
						go n.pushAiTaskFile(slot.Session, slot.PromptID, slot.SubID, slot.ImagePath, allSlots, body)
					} else {
						if strings.EqualFold(parts[0], n.node.GetMeasureID()) {
							//measure
							log4plus.Info(fmt.Sprintf("%s pushMeasureFile subID=[%s] Execution completed", funName, slot.SubID))
							go n.pushMeasureFile(slot.PromptID, slot.Session, slot.SubID, slot.ImagePath, body)
						}
					}
				}
			}
		}
	}
}

func (n *ComfyUI) recv() {
	reconnectTicker := time.NewTicker(30 * time.Second)
	defer reconnectTicker.Stop()

	for {
		select {
		case msg := <-n.SendMessageCh: // cloud -> comfyUI ws
			if err := n.ComfyUILinker.WriteMessage(msg); err != nil {
				return
			}
		case msg := <-n.RecvMessageCh:
			n.listenProgress(msg)
		case <-reconnectTicker.C:
			n.reconnect()
		case <-n.OfflineCh:
			n.offline()
		case <-n.ExitCh:
			n.offline()
			return
		}
	}
}

func (n *ComfyUI) Stop() {
	close(n.ExitCh)
}

func (n *ComfyUI) StartComfyUIWS() {
	funName := "StartComfyUIWS"
	log4plus.Info(fmt.Sprintf("%s ComfyUIUrl=[%s] ---->>>> ", funName, n.ComfyUIWS))
	if n.ComfyUILinker == nil {
		wsComfyUIAddr := fmt.Sprintf("%s/ws?clientId=%s", n.ComfyUIWS, n.clientID)
		n.startComfyUIWS(wsComfyUIAddr)
	}
}

func NewComfyUI(comfyUIHttp, comfyUIWS, saveDir string, workspaceID string, node DockerInterface) *ComfyUI {
	funName := "NewProxy"
	comfyUI := &ComfyUI{
		clientID:      uuid.New().String(),
		WorkspaceID:   workspaceID,
		ComfyUIWS:     comfyUIWS,
		ComfyUIHttp:   comfyUIHttp,
		SaveDir:       saveDir,
		node:          node,
		ExitCh:        make(chan struct{}),
		OfflineCh:     make(chan bool),
		SendMessageCh: make(chan []byte, 1024),
		RecvMessageCh: make(chan []byte, 1024),
	}
	log4plus.Info(fmt.Sprintf("%s ComfyUIWS=[%s] ComfyUIHttp=[%s] SaveDir=[%s]",
		funName, comfyUI.ComfyUIWS, comfyUI.ComfyUIHttp, comfyUI.SaveDir))
	go comfyUI.recv()
	return comfyUI
}
