package configure

import (
	"encoding/json"
	"fmt"
	"os"

	log4plus "github.com/nextGPU/include/log4go"
)

type ApplicationConfig struct {
	Name      string `json:"name"`
	Comment   string `json:"comment"`
	UpdateUrl string `json:"updateUrl"`
}

type Config struct {
	Application ApplicationConfig `json:"application"`
}

type Configure struct {
	config Config
}

var gConfigure *Configure

func (u *Configure) getConfig() error {
	funName := "getConfig"
	log4plus.Info("%s ---->>>>", funName)
	data, err := os.ReadFile("update.json")
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
	return nil
}

func SingletonConfigure() Config {
	if gConfigure == nil {
		gConfigure = &Configure{}
		_ = gConfigure.getConfig()
	}
	return gConfigure.config
}
