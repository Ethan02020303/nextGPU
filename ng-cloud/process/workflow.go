package process

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jinzhu/copier"
	log4plus "github.com/nextGPU/include/log4go"
	"github.com/nextGPU/ng-cloud/db"
	"math/rand"
	"reflect"
	"strings"
)

const (
	SplitString = string("->")
)

// 通用节点结构
type (
	NodeItem struct {
		Inputs    json.RawMessage `json:"inputs"`
		ClassType string          `json:"class_type"`
		Meta      struct {
			Title string `json:"title"`
		} `json:"_meta"`
	}
	Option struct {
		Key   string        `json:"key"`
		Mode  string        `json:"mode"`
		Path  []string      `json:"paths"`
		Value []interface{} `json:"values"`
	}
	Binding struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	WorkflowConfigure struct {
		//Title   string                 `json:"title"`
		Mapping map[string]interface{} `json:"mapping"`
		Enums   map[string]interface{} `json:"enums,omitempty"`
		Fixeds  map[string]interface{} `json:"fixeds"`
		Outputs map[string]interface{} `json:"outputs,omitempty"`
	}
	Workflow struct {
		Title     string              `json:"title"`
		ImageNum  int64               `json:"imageNum"`
		Ratio     string              `json:"ratio"`
		Width     int64               `json:"width"`
		Height    int64               `json:"height"`
		Templates map[string]NodeItem `json:"nodes"`
		Configure WorkflowConfigure
	}
)

// 核心节点参数定义
type (
	KSamplerParams struct {
		Steps       int     `json:"steps"`
		CFG         float64 `json:"cfg"`
		SamplerName string  `json:"sampler_name"`
		Seed        int64   `json:"seed"`
	}
	CheckpointLoaderParams struct {
		CKPTName string `json:"ckpt_name"`
	}
	CLIPTextEncodeParams struct {
		Text string `json:"text"`
		Clip []any  `json:"clip"`
	}
	EmptyLatentParams struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	}
	VAEDecodeParams struct {
		Samples []any `json:"samples"` // 潜在空间数据输入
		VAE     []any `json:"vae"`     // VAE模型引用
	}
	SaveImageParams struct {
		Images    []any  `json:"images"`     // 输入图像数据
		OutputDir string `json:"output_dir"` // 输出目录
		Filename  string `json:"filename"`   // 文件名格式(含变量如%date%)
		Quality   int    `json:"quality"`    // JPG压缩质量[1-100]
		SaveJSON  bool   `json:"save_json"`  // 是否保存参数元数据
		Overwrite bool   `json:"overwrite"`  // 是否覆盖已存在文件
	}
)

// 用户请求格式
type (
	UserRequests struct {
		Users map[string]interface{} `json:"parameters"`
	}
)

type Workflows struct {
}

var gWorkflows *Workflows

func convertTemplateMap(templates map[string]NodeItem) (map[string]interface{}, error) {
	funName := "convertTemplateMap"
	result := make(map[string]interface{})
	for key, tmpl := range templates {
		data, err := json.Marshal(tmpl)
		if err != nil {
			errString := fmt.Sprintf("%s Marshal failed err=[%s]", funName, err.Error())
			log4plus.Error(errString)
			return nil, errors.New(errString)
		}
		var tmplMap map[string]interface{}
		if err = json.Unmarshal(data, &tmplMap); err != nil {
			errString := fmt.Sprintf("%s Unmarshal failed err=[%s]", funName, err.Error())
			log4plus.Error(errString)
			return nil, errors.New(errString)
		}
		result[key] = tmplMap
	}
	return result, nil
}

func setValueByPath(data map[string]interface{}, path string, newValue interface{}) error {
	funName := "setValueByPath"
	keys := strings.Split(path, SplitString)
	current := data
	for i := 0; i < len(keys)-1; i++ {
		key := keys[i]
		val, exists := current[key]
		if !exists {
			errString := fmt.Sprintf("%s key=[%s] not found", funName, key)
			log4plus.Error(errString)
			return errors.New(errString)
		}
		nextMap, ok := val.(map[string]interface{})
		if !ok {
			errString := fmt.Sprintf("%s key=[%s] is not a map", funName, key)
			log4plus.Error(errString)
			return errors.New(errString)
		}
		current = nextMap
	}
	lastKey := keys[len(keys)-1]
	current[lastKey] = newValue
	return nil
}

func (w *Workflows) ParseWorkflow(title string, templateData json.RawMessage, configData db.WorkflowConfigure) (error, *Workflow) {
	funName := "ParseWorkflow"
	// parse template
	var template map[string]NodeItem
	if err := json.Unmarshal(templateData, &template); err != nil {
		errString := fmt.Sprintf("%s Unmarshal parsing failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return errors.New(errString), nil
	}
	workflow := &Workflow{
		Title: title,
	}
	_ = copier.CopyWithOption(&workflow.Templates, &template, copier.Option{DeepCopy: true})
	// Mapping
	workflow.Configure.Mapping = make(map[string]interface{})
	for k, v := range configData.Mapping {
		workflow.Configure.Mapping[k] = v
	}
	// Enums
	workflow.Configure.Enums = make(map[string]interface{})
	for k, v := range configData.Enums {
		workflow.Configure.Enums[k] = v
	}
	// Fixeds
	workflow.Configure.Fixeds = make(map[string]interface{})
	for k, v := range configData.Fixeds {
		workflow.Configure.Fixeds[k] = v
	}
	// Outputs
	workflow.Configure.Outputs = make(map[string]interface{})
	for k, v := range configData.Outputs {
		workflow.Configure.Outputs[k] = v
	}
	return nil, workflow
}

func (w *Workflows) setWorkflow(request *UserRequests, workflow *Workflow) (json.RawMessage, json.RawMessage, error) {
	funName := "setWorkflow"
	if mainTemplate, err := convertTemplateMap(workflow.Templates); err != nil {
		errString := fmt.Sprintf("%s convertTemplateMap failed Templates=[%#v]", funName, workflow.Templates)
		log4plus.Error(errString)
		return nil, nil, errors.New(errString)
	} else {
		for userKey, userValue := range request.Users {
			isDealed := false
			//Enums
			for enumKey, enumValue := range workflow.Configure.Enums {
				if !strings.EqualFold(enumKey, userKey) {
					continue
				}
				isDealed = true
				//check is slice
				t := reflect.TypeOf(enumValue)
				if t.Kind() != reflect.Slice {
					continue
				}
				rawSlice, _ := enumValue.([]interface{})
				for _, itemValue := range rawSlice {
					itemMap, _ := itemValue.(map[string]interface{})
					key, _ := itemMap["key"].(string)
					values, _ := itemMap["value"].([]interface{})
					if strings.EqualFold(key, userValue.(string)) {
						workflow.Ratio = userValue.(string)
						var sizes []float64
						for _, value := range values {
							valueMap, _ := value.(map[string]interface{})
							for keyItem, valueItem := range valueMap {
								sizes = append(sizes, valueItem.(float64))
								if err = setValueByPath(mainTemplate, keyItem, valueItem); err != nil {
									errString := fmt.Sprintf("%s setValueByPath failed err=[%s]", funName, err.Error())
									log4plus.Error(errString)
									return nil, nil, errors.New(errString)
								}
							}
						}
						if len(sizes) == 2 {
							workflow.Width = int64(sizes[0])
							workflow.Height = int64(sizes[1])
						}
						break
					}
				}
				break
			}
			// Mapping
			if !isDealed {
				for mappingKey, mappingValue := range workflow.Configure.Mapping {
					if !strings.EqualFold(mappingKey, userKey) {
						continue
					}
					log4plus.Debug(fmt.Sprintf("%s bindingValue=[%s] -> userValue=[%v]", funName, mappingValue, userValue))
					if err = setValueByPath(mainTemplate, mappingValue.(string), userValue); err != nil {
						errString := fmt.Sprintf("%s setValueByPath failed err=[%s]", funName, err.Error())
						log4plus.Error(errString)
						return nil, nil, errors.New(errString)
					}
					break
				}
			}
		}
		//set sub Template
		subTemplate := make(map[string]interface{})
		_ = copier.CopyWithOption(&subTemplate, &mainTemplate, copier.Option{DeepCopy: true})
		mainData, mainErr := json.Marshal(mainTemplate)
		subData, subErr := json.Marshal(subTemplate)
		if mainErr != nil || subErr != nil {
			errString := fmt.Sprintf("%s Marshal failed mainTemplate=[%s] subTemplate=[%s]", funName, mainTemplate, subTemplate)
			log4plus.Error(errString)
			return nil, nil, errors.New(errString)
		}
		return mainData, subData, nil
	}
}

func newSeed() int {
	return rand.Intn(1000000)
}

func (w *Workflows) SetSeed(data json.RawMessage, workflow db.DockerWorkflow) (json.RawMessage, error) {
	funName := "SetSeed"
	// parse template
	var seedData map[string]NodeItem
	if err := json.Unmarshal(data, &seedData); err != nil {
		errString := fmt.Sprintf("%s Unmarshal parsing failed err=[%s]", funName, err.Error())
		log4plus.Info(errString)
		return json.RawMessage{}, errors.New(errString)
	}
	if seedTemplate, err := convertTemplateMap(seedData); err != nil {
		errString := fmt.Sprintf("%s convertTemplateMap failed Templates=[%s]", funName, workflow.Template)
		log4plus.Error(errString)
		return nil, errors.New(errString)
	} else {
		//set seed
		for k, input := range workflow.Configure.Fixeds {
			if strings.EqualFold(k, "imageSeeds") {
				seed := newSeed()
				var seedPaths []string
				for _, item := range input.([]interface{}) {
					if s, ok := item.(string); ok {
						seedPaths = append(seedPaths, s)
					}
				}
				for _, seedPath := range seedPaths {
					//log4plus.Info("%s setValueByPath path=[%s] value=[%d]", funName, seedPath, seed)
					if err = setValueByPath(seedTemplate, seedPath, seed); err != nil {
						errString := fmt.Sprintf("%s setValueByPath failed err=[%s]", funName, err.Error())
						log4plus.Error(errString)
						return nil, errors.New(errString)
					}
				}
				break
			}
		}
		newWorkflowData, _ := json.Marshal(seedTemplate)
		return newWorkflowData, nil
	}
}

func (w *Workflows) SetMeasureSeed(data, measureConfiguration json.RawMessage) (json.RawMessage, error) {
	funName := "SetMeasureSeed"
	// parse template
	var seedData map[string]NodeItem
	if err := json.Unmarshal(data, &seedData); err != nil {
		errString := fmt.Sprintf("%s Unmarshal parsing failed err=[%s]", funName, err.Error())
		log4plus.Info(errString)
		return json.RawMessage{}, errors.New(errString)
	}
	if seedTemplate, err := convertTemplateMap(seedData); err != nil {
		errString := fmt.Sprintf("%s convertTemplateMap failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return nil, errors.New(errString)
	} else {
		// Shares
		var configure WorkflowConfigure
		if err := json.Unmarshal(measureConfiguration, &configure); err != nil {
			errString := fmt.Sprintf("%s Unmarshal parsing failed err=[%s]", funName, err.Error())
			log4plus.Info(errString)
			return json.RawMessage{}, errors.New(errString)
		}
		for k, input := range configure.Fixeds {
			if strings.EqualFold(k, "imageSeeds") {
				seed := newSeed()
				var seedPaths []string
				for _, item := range input.([]interface{}) {
					if s, ok := item.(string); ok {
						seedPaths = append(seedPaths, s)
					}
				}
				for _, seedPath := range seedPaths {
					if err = setValueByPath(seedTemplate, seedPath, seed); err != nil {
						errString := fmt.Sprintf("%s setValueByPath failed err=[%s]", funName, err.Error())
						log4plus.Error(errString)
						return nil, errors.New(errString)
					}
				}
				break
			}
		}
		newWorkflowData, _ := json.Marshal(seedTemplate)
		return newWorkflowData, nil
	}
}

func (w *Workflows) SetTemplateData(title string,
	templateData json.RawMessage,
	configData db.WorkflowConfigure,
	userData json.RawMessage) (json.RawMessage, json.RawMessage, string, int64, int64, error) {
	funName := "SetTemplateData"
	err, workflow := w.ParseWorkflow(title, templateData, configData)
	if err != nil {
		errString := fmt.Sprintf("%s ParseWorkflow not found jobType=[%s]", funName, title)
		log4plus.Error(errString)
		return json.RawMessage{}, json.RawMessage{}, "", 0, 0, errors.New(errString)
	}
	request := UserRequests{}
	if err = json.Unmarshal(userData, &request); err != nil {
		errString := fmt.Sprintf("%s Unmarshal parsing failed err=[%s]", funName, err.Error())
		log4plus.Info(errString)
		return json.RawMessage{}, json.RawMessage{}, "", 0, 0, err
	}
	mainRes, subRes, err := w.setWorkflow(&request, workflow)
	if err != nil {
		errString := fmt.Sprintf("%s setWorkflow failed err=[%s]", funName, err.Error())
		log4plus.Info(errString)
		return json.RawMessage{}, json.RawMessage{}, "", 0, 0, err
	}
	mainBody, err := json.Marshal(mainRes)
	if err != nil {
		errString := fmt.Sprintf("%s Marshal failed err=[%s]", funName, err.Error())
		log4plus.Info(errString)
		return json.RawMessage{}, json.RawMessage{}, "", 0, 0, err
	}
	subBody, err := json.Marshal(subRes)
	if err != nil {
		errString := fmt.Sprintf("%s Marshal failed err=[%s]", funName, err.Error())
		log4plus.Info(errString)
		return json.RawMessage{}, json.RawMessage{}, "", 0, 0, err
	}
	return mainBody, subBody, workflow.Ratio, workflow.Width, workflow.Height, nil
}

func SingletonWorkflows() *Workflows {
	if gWorkflows == nil {
		gWorkflows = &Workflows{}
	}
	return gWorkflows
}
