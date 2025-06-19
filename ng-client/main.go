package main

import (
	"flag"
	"fmt"
	"github.com/beevik/ntp"
	"github.com/nextGPU/common"
	log4plus "github.com/nextGPU/include/log4go"
	"github.com/nextGPU/ng-client/configure"
	"github.com/nextGPU/ng-client/process"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"time"
)

const (
	Version = "1.0.0"
)

type Flags struct {
	Help    bool
	Version bool
}

func (f *Flags) Init() {
	flag.BoolVar(&f.Help, "h", false, "help")
	flag.BoolVar(&f.Version, "v", false, "show version")
}

func (f *Flags) Check() (needReturn bool) {
	flag.Parse()
	if f.Help {
		flag.Usage()
		needReturn = true
	} else if f.Version {
		verString := configure.SingletonConfigure().Application.Comment + " Version: " + Version + "\r\n"
		fmt.Println(verString)
		needReturn = true
	}
	return needReturn
}

var flags *Flags = &Flags{}

func init() {
	flags.Init()
}

func getExeName() string {
	ret := ""
	ex, err := os.Executable()
	if err == nil {
		ret = filepath.Base(ex)
	}
	return ret
}

func PathExist(fullPath string) bool {
	_, err := os.Stat(fullPath)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}

func setLog() {
	logJson := "log.json"
	set := false
	if bExist := PathExist(logJson); bExist {
		if err := log4plus.SetupLogWithConf(logJson); err == nil {
			set = true
		}
	}
	if !set {
		fileWriter := log4plus.NewFileWriter()
		exeName := getExeName()
		_ = fileWriter.SetPathPattern("./log/" + exeName + "-%Y%M%D.log")
		log4plus.Register(fileWriter)
		log4plus.SetLevel(log4plus.DEBUG)
	}
}

func syncTimeWithNTP() {
	funName := "syncTimeWithNTP"
	ntpServers := []string{"time.google.com", "pool.ntp.org", "ntp.aliyun.com"}
	var resp *ntp.Response
	var err error
	for _, server := range ntpServers {
		resp, err = ntp.QueryWithOptions(server, ntp.QueryOptions{Timeout: 3 * time.Second})
		if err == nil {
			break
		}
	}
	if err != nil {
		errString := fmt.Sprintf("%s All NTP servers are unavailable err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return
	}
	adjustedTime := time.Now().Add(resp.ClockOffset)
	log4plus.Info(fmt.Sprintf("%s NTP current time=[%s]", funName, adjustedTime.Format("2006-01-02 15:04:05.000")))
}

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

func parseYaml(filePath string) {
	funName := "parseYaml"
	data, err := os.ReadFile(filePath)
	if err != nil {
		errString := fmt.Sprintf("%s ReadFile Failed filePath=[%s] err=[%s]", funName, filePath, err.Error())
		log4plus.Error(errString)
		return
	}
	var compose DockerCompose
	if errYaml := yaml.Unmarshal(data, &compose); errYaml != nil {
		errString := fmt.Sprintf("%s yaml.Unmarshal Failed filePath=[%s] err=[%s]", funName, filePath, errYaml.Error())
		log4plus.Error(errString)
		return
	}
	service := compose.Services["comfyui"]
	fmt.Println("镜像名称:", service.Image)
	fmt.Println("端口映射:", service.Ports)
	fmt.Println("GPU 数量:", service.Deploy.Resources.Reservations.Devices[0].Count)
}

func main() {
	defer common.CrashDump()
	needReturn := flags.Check()
	if needReturn {
		return
	}
	setLog()

	syncTimeWithNTP()
	_, _ = process.GetGPUInfo()
	configure.SingletonConfigure()
	process.SingletonProcess()
	dir, _ := os.Getwd()
	log4plus.Info(fmt.Sprintf("当前工作目录 current dir=[%s]", dir))

	for {
		time.Sleep(time.Duration(10) * time.Second)
	}
}
