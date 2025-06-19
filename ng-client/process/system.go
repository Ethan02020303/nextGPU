package process

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	log4plus "github.com/nextGPU/include/log4go"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

func isWindows() bool {
	return runtime.GOOS == "windows"
}

func isLinux() bool {
	return runtime.GOOS == "linux"
}

func OSInfo() (error, string) {
	funName := "OSInfo"
	info, err := host.Info()
	if err != nil {
		errString := fmt.Sprintf("%s host.Info Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return err, ""
	}
	outString := fmt.Sprintf("%s %s (kernel: %s)", info.Platform, info.PlatformVersion, info.KernelVersion)
	return nil, outString
}

func CPUInfo() (error, string) {
	funName := "CPUInfo"
	cpuInfo, err := cpu.Info()
	if err != nil {
		errString := fmt.Sprintf("%s cpu.Info Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return err, ""
	}
	if len(cpuInfo) == 0 {
		errString := fmt.Sprintf("%s len(cpuInfo) == 0", funName)
		log4plus.Error(errString)
		return err, ""
	}
	physicalCores, _ := cpu.Counts(false)
	logicalCores, _ := cpu.Counts(true)
	outString := fmt.Sprintf("%s (physical cores:%d) (logical cores:%d)", cpuInfo[0].ModelName, physicalCores, logicalCores)
	return nil, outString
}

func MemoryInfo() (error, uint64) {
	funName := "MemoryInfo"
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		errString := fmt.Sprintf("%s mem.VirtualMemory Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return err, 0
	}
	return nil, memInfo.Total
}

// -----------------------
type GPUInfo struct {
	Model  string
	UUID   string
	Driver string
}

func findNvidiaSmi() (string, error) {
	funName := "findNvidiaSmi"
	if isWindows() {
		paths := []string{
			`C:\Windows\System32\nvidia-smi.exe`,
			`C:\Program Files\NVIDIA Corporation\NVSMI\nvidia-smi.exe`,
		}
		for _, path := range paths {
			if checkExecutable(path) {
				return path, nil
			}
		}
		errString := fmt.Sprintf("%s nvidia-smi not found", funName)
		return "", errors.New(errString)
	}
	// Linux系统使用原查找逻辑
	cmd := exec.Command("find", "/usr", "-path", "*/nvidia-smi", "-executable", "-print", "-quit")
	output, err := cmd.Output()
	if err != nil {
		errString := fmt.Sprintf("%s find nvidia-smi Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

type Win32VideoController struct {
	Name          string `json:"Name"`
	PNPDeviceID   string `json:"PNPDeviceID"`
	DriverVersion string `json:"DriverVersion"`
}

type DeviceInfo struct {
	Name          string `json:"Name"`
	PNPDeviceID   string `json:"PNPDeviceID"`
	DriverVersion string `json:"DriverVersion"`
}

func formatNvidiaUUID(uuid string) string {
	if strings.HasPrefix(uuid, "GPU-") {
		return strings.TrimPrefix(uuid, "GPU-")
	}
	return uuid
}

func windowsGPU() ([]GPUInfo, error) {
	funName := "windowsGPU"
	path, err := findNvidiaSmi()
	if err != nil {
		return []GPUInfo{}, err
	}
	cmd := exec.Command(path, "--query-gpu=name,uuid,driver_version", "--format=csv,noheader,nounits")
	output, errOutput := cmd.CombinedOutput()
	if errOutput != nil {
		errString := fmt.Sprintf("%s nvidia-smi running Failed err=[%s]", funName, errOutput.Error())
		log4plus.Error(errString)
		return nil, errOutput
	}
	var gpus []GPUInfo
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, ", ")
		if len(parts) != 3 {
			continue
		}
		gpus = append(gpus, GPUInfo{
			Model:  parts[0],
			UUID:   formatNvidiaUUID(parts[1]),
			Driver: parts[2],
		})
	}
	return gpus, nil
}

func parseCSVOutput(data string) ([]GPUInfo, error) {
	funName := "parseCSVOutput"
	reader := csv.NewReader(strings.NewReader(data))
	reader.TrimLeadingSpace = true // 处理字段前导空格
	records, err := reader.ReadAll()
	if err != nil {
		errString := fmt.Sprintf("%s ReadAll Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return nil, err
	}
	gpuInfos := make([]GPUInfo, 0)
	for _, record := range records {
		if len(record) != 3 {
			continue // 跳过格式异常行
		}
		info := GPUInfo{
			Model:  strings.TrimSpace(record[0]),
			UUID:   formatNvidiaUUID(record[1]),
			Driver: strings.TrimSpace(record[2]),
		}
		gpuInfos = append(gpuInfos, info)
	}
	if len(gpuInfos) == 0 {
		errString := fmt.Sprintf("%s not found GPU Information", funName)
		log4plus.Error(errString)
		return nil, errors.New(errString)
	}
	return gpuInfos, nil
}

func linuxGPU() ([]GPUInfo, error) {
	funName := "linuxGPU"
	// 执行nvidia-smi命令获取结构化数据
	cmd := exec.Command("nvidia-smi",
		"--query-gpu=name,uuid,driver_version",
		"--format=csv,noheader,nounits")

	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		errString := fmt.Sprintf("%s cmd.Run running failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return nil, errors.New(errString)
	}
	return parseCSVOutput(out.String())
}

func GetGPUInfo() ([]GPUInfo, error) {
	switch runtime.GOOS {
	case "linux":
		return linuxGPU()
	case "windows":
		return windowsGPU()
	default:
		return []GPUInfo{}, errors.New("not found system")
	}
}

func checkExecutable(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !fileInfo.IsDir()
}

func isValidUUID(uuid string) bool {
	if len(uuid) != 36 && len(uuid) != 32 {
		return false
	}
	validChars := "abcdefABCDEF0123456789-"
	for _, c := range uuid {
		if !strings.ContainsRune(validChars, c) {
			return false
		}
	}
	return true
}

func addEnvPath() {
	// 目标要添加的路径（WSL格式）
	targetPaths := []string{
		"/mnt/c/Windows/System32",
		"/mnt/c/Windows/System32/WindowsPowerShell/v1.0",
	}
	// 获取当前PATH
	oldPath, exists := os.LookupEnv("PATH")
	if !exists {
		oldPath = ""
	}
	// 分割现有PATH为切片
	pathSlice := strings.Split(oldPath, ":")
	// 检测缺失路径
	var missingPaths []string
	for _, target := range targetPaths {
		found := false
		for _, p := range pathSlice {
			if p == target {
				found = true
				break
			}
		}
		if !found {
			missingPaths = append(missingPaths, target)
		}
	}
	// 动态追加新路径
	if len(missingPaths) > 0 {
		newPath := strings.Join(append(pathSlice, missingPaths...), ":")
		_ = os.Setenv("PATH", newPath)
	}
}

func linuxBoardUUID() (error, string) {
	funName := "linuxBoardUUID"

	//动态添加环境变量
	addEnvPath()
	//查找powershell.exe
	runError := false
	powershellPath, err := exec.LookPath("powershell.exe")
	if err != nil {
		log4plus.Info(fmt.Sprintf("%s exec.LookPath failed err=[%s]", funName, err.Error()))
		powershellPath = "/mnt/c/Windows/System32/WindowsPowerShell/v1.0/powershell.exe"
	}
	log4plus.Info(fmt.Sprintf("%s exec.LookPath powershellPath=[%s]", funName, powershellPath))
	//cmd := exec.Command("powershell.exe", "-Command", "Get-CimInstance Win32_ComputerSystemProduct | Select-Object -ExpandProperty UUID")
	cmd := exec.Command(powershellPath, "-Command", "Get-CimInstance Win32_ComputerSystemProduct | Select-Object -ExpandProperty UUID")
	out, err := cmd.Output()
	if err != nil {
		log4plus.Info(fmt.Sprintf("%s cmd.Output failed err=[%s]", funName, err.Error()))
		runError = true
	}
	log4plus.Info(fmt.Sprintf("%s runError=[%t]", funName, runError))
	if !runError {
		uuid := strings.TrimSpace(string(out))
		log4plus.Info(fmt.Sprintf("%s strings.TrimSpace uuid=[%s]", funName, uuid))

		if isValidUUID(uuid) {
			return nil, uuid
		}
		lines := strings.Split(strings.ReplaceAll(string(out), "\r\n", "\n"), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "UUID") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					uuid = strings.TrimSpace(parts[1])
					if isValidUUID(uuid) {
						log4plus.Info(fmt.Sprintf("%s isValidUUID uuid=[%s]", funName, uuid))
						return nil, uuid
					}
				}
			}
		}
		errString := fmt.Sprintf("%s invalid UUID from PowerShell", funName)
		log4plus.Error(errString)
		return errors.New(errString), ""
	} else {
		dmidecodePath, errFile := exec.LookPath("dmidecode")
		if errFile != nil {
			possiblePaths := []string{
				"/usr/sbin/dmidecode",
				"/sbin/dmidecode",
				"/usr/local/sbin/dmidecode",
			}
			for _, path := range possiblePaths {
				if _, err := os.Stat(path); err == nil {
					dmidecodePath = path
					break
				}
			}
			if dmidecodePath == "" {
				errString := fmt.Sprintf("%s dmidecode not found in $PATH or default locations", funName)
				log4plus.Error(errString)
				runError = true
				return errors.New(errString), ""
			}
		}
		//cmd = exec.Command("sudo", "dmidecode", "-s", "system-uuid")
		cmd = exec.Command(dmidecodePath, "-s", "system-uuid")
		out, err = cmd.CombinedOutput()
		if err != nil {
			errString := fmt.Sprintf("%s Output Failed err=[%s]", funName, err.Error())
			log4plus.Error(errString)
			runError = true
			return err, ""
		}
		uuid := string(out[:len(out)-1])
		log4plus.Info(fmt.Sprintf("%s isValidUUID uuid=[%s]", funName, uuid))
		if isValidUUID(uuid) {
			return nil, uuid
		}
		errString := fmt.Sprintf("%s invalid UUID from PowerShell", funName)
		log4plus.Error(errString)
		return errors.New(errString), ""
	}
}

// Windows
func powershellBoardUUID() (error, string) {
	funName := "powershellBoardUUID"
	cmd := exec.Command("powershell", "-Command",
		"Get-CimInstance Win32_ComputerSystemProduct | select Name,Vendor,Version,IdentifyingNumber,UUID | fl")
	out, err := cmd.CombinedOutput()
	if err != nil {
		errString := fmt.Sprintf("%s CombinedOutput Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return err, ""
	}
	uuid := strings.TrimSpace(string(out))
	if isValidUUID(uuid) {
		return nil, uuid
	}
	lines := strings.Split(strings.ReplaceAll(string(out), "\r\n", "\n"), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "UUID") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				uuid := strings.TrimSpace(parts[1])
				if isValidUUID(uuid) {
					return nil, uuid
				}
			}
		}
	}
	errString := fmt.Sprintf("%s invalid UUID from PowerShell", funName)
	log4plus.Error(errString)
	return errors.New(errString), ""
}

func windowsBoardUUID() (error, string) {
	funName := "windowsBoardUUID"
	cmd := exec.Command("wmic", "csproduct", "get", "uuid")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return powershellBoardUUID()
	}
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "UUID") {
			if isValidUUID(line) {
				return nil, line
			}
		}
	}
	errString := fmt.Sprintf("%s invalid output format", funName)
	log4plus.Error(errString)
	return errors.New(errString), ""
}

// The motherboard UUID cannot be retrieved within Docker containers or WSL (Windows Subsystem for Linux) environments.
func BoardInfo() (error, string) {
	funName := "BoardInfo"
	switch runtime.GOOS {
	case "linux":
		log4plus.Info(fmt.Sprintf("%s linuxBoardUUID--->>>", funName))
		return linuxBoardUUID()
	case "windows":
		log4plus.Info(fmt.Sprintf("%s windowsBoardUUID--->>>", funName))
		return windowsBoardUUID()
	default:
		return fmt.Errorf("unsupported platform"), ""
	}
}

func GenerateEnvironmentID() string {
	funName := "GenerateEnvironmentID"
	var err error
	osInfo := ""
	if err, osInfo = OSInfo(); err != nil {
		osInfo = ""
	}
	cpuString := ""
	if err, cpuString = CPUInfo(); err != nil {
		cpuString = ""
	}
	memory := uint64(0)
	if err, memory = MemoryInfo(); err != nil {
		memory = 0
	}
	gpu := ""
	gpuDriver := ""
	var gpus []GPUInfo
	if gpus, err = GetGPUInfo(); err != nil {
		gpu = ""
		gpuDriver = ""
	} else {
		gpuModel := ""
		for _, singleGPU := range gpus {
			gpuModel += singleGPU.Model + "\n"
		}
		gpu = gpuModel
		gpuDriver = gpus[0].Driver
	}
	mainBoardUUID := ""
	if err, mainBoardUUID = BoardInfo(); err != nil {
		mainBoardUUID = ""
	}
	log4plus.Info(fmt.Sprintf("%s osInfo=[%s] cpu=[%s] memory=[%d] gpu=[%s] gpuDriver=[%s]\nUUID=[%s]", funName, osInfo, cpuString, memory, gpu, gpuDriver, mainBoardUUID))
	return mainBoardUUID
}
