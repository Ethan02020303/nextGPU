package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/beevik/ntp"
	"github.com/nextGPU/common"
	log4plus "github.com/nextGPU/include/log4go"
	"github.com/nextGPU/ng-client/configure"
	"github.com/nextGPU/ng-client/process"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
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

// 获取主板UUID
func GetBoardUUID() (string, error) {
	// 首先尝试直接读取Linux系统文件（最可靠的方式）
	if uuid, err := getLinuxSysUUID(); err == nil {
		return uuid, nil
	}

	// 检查是否在WSL环境中
	if isWSL() {
		// 尝试使用Windows方法获取UUID
		if uuid, err := getWindowsUUID(); err == nil {
			return uuid, nil
		}
	}

	// 最后尝试使用dmidecode
	return getDMIDecodeUUID()
}

// 判断是否在WSL环境中
func isWSL() bool {
	// 检查WSL特定文件
	if _, err := os.Stat("/proc/sys/fs/binfmt_misc/WSLInterop"); !os.IsNotExist(err) {
		return true
	}

	// 检查内核版本是否包含Microsoft
	if content, err := ioutil.ReadFile("/proc/version"); err == nil {
		return strings.Contains(strings.ToLower(string(content)), "microsoft")
	}

	return false
}

// Linux系统文件方式获取UUID（首选）
func getLinuxSysUUID() (string, error) {
	// 尝试读取系统文件
	sysPaths := []string{
		"/sys/class/dmi/id/product_uuid",
		"/sys/devices/virtual/dmi/id/product_uuid",
		"/sys/firmware/dmi/tables/smbios_entry_point",
	}

	for _, path := range sysPaths {
		if data, err := ioutil.ReadFile(path); err == nil {
			uuid := normalizeUUID(string(data))
			if isValidUUID(uuid) {
				return uuid, nil
			}
		}
	}
	return "", errors.New("无法从系统文件读取UUID")
}

// dmidecode方式获取UUID
func getDMIDecodeUUID() (string, error) {
	// 尝试执行dmidecode命令，获取更详细输出
	cmd := exec.Command("sudo", "dmidecode", "-t", "system")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("dmidecode执行失败: %v", err)
	}

	// 从详细输出中提取UUID
	uuid := extractUUIDFromOutput(string(out))
	if isValidUUID(uuid) {
		return uuid, nil
	}

	return "", errors.New("从dmidecode输出中无法解析有效的UUID")
}

// 从dmidecode输出中提取UUID
func extractUUIDFromOutput(output string) string {
	lines := strings.Split(strings.ReplaceAll(output, "\r", ""), "\n")
	var uuid string

	for i, line := range lines {
		line = strings.TrimSpace(line)

		// 检查UUID行
		if strings.Contains(line, "UUID:") || strings.Contains(line, "UUID (Universally Unique ID):") {
			// 直接在同一行中查找UUID
			uuid = strings.TrimSpace(line[strings.Index(line, ":")+1:])
			if isValidUUID(uuid) {
				return uuid
			}

			// 或者在下一行查找UUID
			if i+1 < len(lines) {
				nextLine := strings.TrimSpace(lines[i+1])
				if isValidUUID(nextLine) {
					return nextLine
				}
			}
		}
	}

	// 搜索整个输出中的UUID模式
	uuidRegex := regexp.MustCompile(`(?i)[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
	if matches := uuidRegex.FindStringSubmatch(output); matches != nil {
		return matches[0]
	}

	return ""
}

// Windows方式获取UUID（在WSL中使用）
func getWindowsUUID() (string, error) {
	// 尝试使用wmic命令获取UUID
	if uuid, err := getWindowsUUIDViaWMIC(); err == nil {
		return uuid, nil
	}

	// 尝试使用注册表获取MachineGuid
	if uuid, err := getWindowsUUIDViaRegistry(); err == nil {
		return uuid, nil
	}

	// 尝试使用systeminfo命令
	if uuid, err := getWindowsUUIDViaSystemInfo(); err == nil {
		return uuid, nil
	}

	return "", errors.New("无法通过Windows方法获取UUID")
}

// 使用wmic命令获取UUID
func getWindowsUUIDViaWMIC() (string, error) {
	// 使用完整路径确保WSL中可以正常工作
	cmd := exec.Command("cmd.exe", "/c", "C:\\Windows\\System32\\wbem\\WMIC.exe csproduct get uuid")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("wmic执行失败: %v", err)
	}

	return extractUUIDFromOutput(string(out)), nil
}

// 使用注册表获取MachineGuid
func getWindowsUUIDViaRegistry() (string, error) {
	cmd := exec.Command("cmd.exe", "/c", "reg query HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Cryptography /v MachineGuid")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("reg查询失败: %v", err)
	}

	// 从输出中提取GUID
	return extractUUIDFromOutput(string(out)), nil
}

// 使用systeminfo命令获取系统信息
func getWindowsUUIDViaSystemInfo() (string, error) {
	cmd := exec.Command("cmd.exe", "/c", "systeminfo")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("systeminfo执行失败: %v", err)
	}

	systemInfo := string(out)
	if idx := strings.Index(systemInfo, "System UUID:"); idx != -1 {
		// 尝试提取UUID字段
		line := systemInfo[idx:]
		if newline := strings.Index(line, "\n"); newline != -1 {
			line = line[:newline]
		}

		if uuid := extractUUIDFromOutput(line); uuid != "" {
			return uuid, nil
		}
	}

	return "", errors.New("从systeminfo输出中无法解析UUID")
}

// 验证UUID格式
func isValidUUID(uuid string) bool {
	// 移除多余空格和非打印字符
	uuid = strings.TrimSpace(uuid)

	// 检查长度
	if len(uuid) < 32 || len(uuid) > 36 {
		return false
	}

	// 正则表达式验证UUID格式
	pattern := `(?i)^[0-9a-f]{8}-?[0-9a-f]{4}-?[0-9a-f]{4}-?[0-9a-f]{4}-?[0-9a-f]{12}$`
	matched, err := regexp.MatchString(pattern, uuid)
	return err == nil && matched
}

// 规范化UUID格式
func normalizeUUID(uuid string) string {
	// 移除空格和非打印字符
	uuid = strings.TrimSpace(uuid)
	uuid = strings.ToLower(uuid)

	// 移除换行符
	uuid = strings.ReplaceAll(uuid, "\r", "")
	uuid = strings.ReplaceAll(uuid, "\n", "")

	// 如果UUID不带横线，添加标准格式分隔符
	if len(uuid) == 32 {
		return fmt.Sprintf("%s-%s-%s-%s-%s",
			uuid[:8], uuid[8:12], uuid[12:16], uuid[16:20], uuid[20:32])
	}

	return uuid
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
