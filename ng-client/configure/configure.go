package configure

import (
	"encoding/json"
	"fmt"
	"os"

	log4plus "github.com/nextGPU/include/log4go"
)

type ApplicationConfig struct {
	Name    string `json:"name"`
	Comment string `json:"comment"`
}

type WebConfig struct {
	Validate string `json:"validate"`
	Enroll   string `json:"enroll"`
}

type Config struct {
	Application ApplicationConfig `json:"application"`
	Web         WebConfig         `json:"web"`
}

type Configure struct {
	config Config
}

//func GetJsonFileName() string {
//	return "configCheck.json"
//}

var gConfigure *Configure

func (u *Configure) getConfig() error {
	funName := "getConfig"
	log4plus.Info(fmt.Sprintf("%s ---->>>>", funName))
	data, err := os.ReadFile("./config.json")
	if err != nil {
		log4plus.Error(fmt.Sprintf("%s os.ReadFile error=[%s]", funName, err.Error()))
		return err
	}
	log4plus.Info(fmt.Sprintf("%s data=[%s]", funName, string(data)))
	err = json.Unmarshal(data, &u.config)
	if err != nil {
		log4plus.Error(fmt.Sprintf("%s json.Unmarshal error=[%s]", funName, err.Error()))
		return err
	}
	//fileName := GetJsonFileName()
	//file, err := os.Create(fileName)
	//if err != nil {
	//	log4plus.Error(fmt.Sprintf("%s os.Create error=[%s]", funName, err.Error()))
	//	return err
	//}
	//defer file.Close()
	//encoder := json.NewEncoder(file)
	//encoder.SetIndent("", "  ")
	//err = encoder.Encode(u.config)
	//if err != nil {
	//	log4plus.Error(fmt.Sprintf("%s encoder.Encode error=[%s]", funName, err.Error()))
	//	return err
	//}
	log4plus.Info(fmt.Sprintf("%s success---->>>>", funName))
	return nil
}

func SingletonConfigure() Config {
	if gConfigure == nil {
		gConfigure = &Configure{}
		_ = gConfigure.getConfig()
	}
	return gConfigure.config
}
