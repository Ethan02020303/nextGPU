package process

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/google/shlex"
	log4plus "github.com/nextGPU/include/log4go"
	"github.com/nextGPU/ng-client/header"
	"github.com/nextGPU/ng-client/process/plugIn"
	"gopkg.in/yaml.v3"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

type (
	DockerBase struct {
		ClientVer    string               `json:"clientVer"`
		ServerVer    string               `json:"serverVer"`
		ContainerVer string               `json:"containerVer"`
		Params       string               `json:"params"`
		WorkSpaceID  string               `json:"workSpaceID"`
		RunningMode  header.RunMode       `json:"runMode"`
		StartCMD     []string             `json:"startCMD"`
		ComfyUIHttp  string               `json:"comfyUIHttp"`
		ComfyUIWS    string               `json:"comfyUIWS"`
		SaveDir      string               `json:"saveDir"`
		GPUMem       string               `json:"gpuMem"`
		Deleted      header.DockerDeleted `json:"deleted"`
	}
	NodeDocker struct {
		node            *Node
		WorkSpaceID     string
		State           header.DockerStatus
		Compose         DockerCompose
		Base            DockerBase
		ContainerID     string
		dockerClient    *client.Client
		ComfyUI         *plugIn.ComfyUI
		ExitCh          chan struct{}
		StartupDockerCh chan struct{}
		DelDockerCh     chan string
		MeasureCh       chan *header.MeasureMessage
		TaskCh          chan *header.PublishMessage
	}
)

type (
	DockerCompose struct {
		Version  string             `yaml:"version"`
		Services map[string]Service `yaml:"services"`
		Networks map[string]Network `yaml:"networks,omitempty"`
		Volumes  map[string]Volume  `yaml:"volumes,omitempty"`
	}
	Service struct {
		Build         BuildConfig  `yaml:"build,omitempty"`
		Image         string       `yaml:"image,omitempty"`
		ContainerName string       `yaml:"container_name,omitempty"`
		Ports         []string     `yaml:"ports,omitempty"`
		Volumes       []string     `yaml:"volumes,omitempty"`
		Environment   []string     `yaml:"environment,omitempty"`
		DependsOn     []string     `yaml:"depends_on,omitempty"`
		Networks      []string     `yaml:"networks,omitempty"`
		Deploy        DeployConfig `yaml:"deploy,omitempty"`
		Command       interface{}  `yaml:"command,omitempty"`
	}
	BuildConfig struct {
		Context    string `yaml:"context,omitempty"`
		Dockerfile string `yaml:"dockerfile,omitempty"`
	}
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
	Network struct {
		Driver string `yaml:"driver,omitempty"`
	}
	Volume struct {
		Driver string `yaml:"driver,omitempty"`
	}
)

func parseYaml(filePath string) DockerCompose {
	funName := "parseYaml"
	data, err := os.ReadFile(filePath)
	if err != nil {
		errString := fmt.Sprintf("%s ReadFile Failed filePath=[%s] err=[%s]", funName, filePath, err.Error())
		log4plus.Error(errString)
		return DockerCompose{}
	}
	var compose DockerCompose
	if errYaml := yaml.Unmarshal(data, &compose); errYaml != nil {
		errString := fmt.Sprintf("%s yaml.Unmarshal Failed filePath=[%s] err=[%s]", funName, filePath, errYaml.Error())
		log4plus.Error(errString)
		return DockerCompose{}
	}
	return compose
}

func isRunningContainer(cli *client.Client, containerName string) bool {
	funName := "getRunningContainer"
	containers, err := cli.ContainerList(
		context.Background(),
		types.ContainerListOptions{All: false},
	)
	if err != nil {
		errString := fmt.Sprintf("%s cli.ContainerList Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return false
	}
	for _, container := range containers {
		normalizedName := strings.TrimPrefix(container.Names[0], "/")
		if strings.EqualFold(containerName, normalizedName) {
			return true
		}
	}
	return false
}

func getExitedContainer(cli *client.Client, containerName string) bool {
	funName := "getExitedContainer"
	containers, err := cli.ContainerList(
		context.Background(),
		types.ContainerListOptions{All: true},
	)
	if err != nil {
		errString := fmt.Sprintf("%s cli.ContainerList Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return false
	}
	for _, container := range containers {
		switch strings.ToLower(container.State) {
		case "exited", "dead":
			normalizedName := strings.TrimPrefix(container.Names[0], "/")
			if strings.EqualFold(containerName, normalizedName) {
				return true
			}
		case "restarting":
			log4plus.Warn(fmt.Sprintf("容器 %s 处于重启状态", container.ID))
		}
	}
	return false
}

func startupContainer(cli *client.Client, containerName string) bool {
	funName := "startupContainer"
	containers, err := cli.ContainerList(
		context.Background(),
		types.ContainerListOptions{All: true},
	)
	if err != nil {
		errString := fmt.Sprintf("%s cli.ContainerList Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return false
	}
	for _, container := range containers {
		switch strings.ToLower(container.State) {
		case "exited", "dead":
			normalizedName := strings.TrimPrefix(container.Names[0], "/")
			if strings.EqualFold(containerName, normalizedName) {
				log4plus.Info(fmt.Sprintf("%s ContainerStart container.ID=[%s]", funName, container.ID))
				if err := cli.ContainerStart(context.Background(), container.ID, types.ContainerStartOptions{}); err != nil {
					errString := fmt.Sprintf("%s cli.ContainerStart Failed err=[%s]", funName, err.Error())
					log4plus.Error(errString)
					return false
				}
				return true
			}
		}
	}
	return false
}

func cleanupContainer(containerName string) {
	funName := "cleanupContainer"
	filter := "name=" + containerName
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("wsl", "docker", "ps", "-a", "--filter", filter, "--format", "{{.ID}}")
	} else {
		cmd = exec.Command("docker", "ps", "-a", "--filter", filter, "--format", "{{.ID}}")
	}
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		errString := fmt.Sprintf("%s docker cmd failed. Verify Docker in WSL is running. err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return
	}
	containerIDs := strings.Split(strings.TrimSpace(out.String()), "\n")
	for _, id := range containerIDs {
		if id == "" {
			continue
		}
		var rmCmd *exec.Cmd
		//stop
		if runtime.GOOS == "windows" {
			rmCmd = exec.Command("wsl", "docker", "stop", id)
		} else {
			rmCmd = exec.Command("docker", "stop", id)
		}
		if err := rmCmd.Run(); err != nil {
			errString := fmt.Sprintf("%s delete container failed err=[%s]", funName, err.Error())
			log4plus.Error(errString)
		}
		//rm
		if runtime.GOOS == "windows" {
			rmCmd = exec.Command("wsl", "docker", "rm", "-f", id)
		} else {
			rmCmd = exec.Command("docker", "rm", "-f", id)
		}
		if err := rmCmd.Run(); err != nil {
			errString := fmt.Sprintf("%s delete container failed err=[%s]", funName, err.Error())
			log4plus.Error(errString)
		}
	}
}

func checkStartup(cli *client.Client, containerName string) bool {
	funName := "checkStartup"
	ctx := context.Background()
	containerJSON, err := cli.ContainerInspect(ctx, containerName)
	if err != nil {
		errString := fmt.Sprintf("%s ContainerInspect status err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return false
	}
	return containerJSON.State.Running
}

func dockerVersion(cli *client.Client) (string, string) {
	serverVersion := ""
	clientVersion := ""
	if server, err := cli.ServerVersion(context.Background()); err == nil {
		serverVersion = server.Version
	}
	clientVersion = cli.ClientVersion()
	return serverVersion, clientVersion
}

// image
func isExistImage(cli *client.Client, imageName string) bool {
	funName := "getImages"
	images, err := cli.ImageList(context.Background(), types.ImageListOptions{All: true})
	if err != nil {
		errString := fmt.Sprintf("%s cli.ImageList Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return false
	}
	for _, image := range images {
		unTaggedName := extractUntaggedName(image)
		formatName := formatImageName(unTaggedName)
		if strings.EqualFold(imageName, formatName) {
			return true
		}
	}
	return false
}

func deleteImage(cli *client.Client, registryDomain, imageName, tag string) error {
	funName := "deleteImage"
	parts := strings.Split(imageName, "/")
	targetName := fmt.Sprintf("%s/%s/%s:%s", registryDomain, parts[len(parts)-2], parts[len(parts)-1], tag)
	_, err := cli.ImageRemove(context.Background(), targetName,
		types.ImageRemoveOptions{
			Force:         true,
			PruneChildren: true,
		},
	)
	if err != nil {
		errString := fmt.Sprintf("%s cli.ImageRemove Failed targetName=[%s] err=[%s]", funName, targetName, err.Error())
		log4plus.Error(errString)
		return err
	}
	return nil
}

func deleteImage2(cli *client.Client, imageName string) error {
	funName := "deleteImage2"
	_, err := cli.ImageRemove(context.Background(), imageName,
		types.ImageRemoveOptions{
			Force:         true,
			PruneChildren: true,
		},
	)
	if err != nil {
		errString := fmt.Sprintf("%s cli.ImageRemove Failed imageName=[%s] err=[%s]", funName, imageName, err.Error())
		log4plus.Error(errString)
		return err
	}
	return nil
}

func extractUntaggedName(img types.ImageSummary) string {
	funName := "extractUntaggedName"
	if len(img.RepoTags) == 0 {
		return "<none>" // 无标签的镜像
	}
	tagStr := img.RepoTags[0]
	if tagStr == "<none>:<none>" {
		return "<none>"
	}
	named, err := reference.ParseNormalizedNamed(tagStr)
	if err != nil {
		errString := fmt.Sprintf("%s parseNormalizedNamed Failed tagStr=[%#v] err=[%s]", funName, tagStr, err.Error())
		log4plus.Error(errString)
		return tagStr // 退回原始字符串
	}
	return named.Name()
}

func formatImageName(fullName string) string {
	named, err := reference.ParseNormalizedNamed(fullName)
	if err != nil {
		return fullName
	}
	path := reference.Path(named)
	if strings.HasPrefix(path, "library/") {
		return strings.TrimPrefix(path, "library/")
	}
	return path
}

func saveYamlFile(yaml string) string {
	funName := "saveYamlFile"
	fileName := "docker-compose.yaml"
	err := os.WriteFile(fileName, []byte(yaml), 0644)
	if err != nil {
		errString := fmt.Sprintf("%s WriteFile Failed fileName=[%s] err=[%s]", funName, fileName, err.Error())
		log4plus.Error(errString)
		return ""
	}
	absPath, err := filepath.Abs(fileName)
	if err != nil {
		errString := fmt.Sprintf("%s Abs Failed fileName=[%s] err=[%s]", funName, fileName, err.Error())
		log4plus.Error(errString)
		return ""
	}
	log4plus.Info(fmt.Sprintf("%s filepath.Abs absPath=[%s]", funName, absPath))
	return absPath
}

func replacePwdMarker(input string) (string, error) {
	if strings.Contains(input, "${PWD}") {
		currentDir, err := os.Getwd()
		if err != nil {
			return "", err
		}
		return strings.ReplaceAll(input, "${PWD}", currentDir), nil
	}
	return input, nil
}

func setWritableDir(path string) error {
	funName := "setWritableDir"
	// 检查目录是否存在
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return nil
	}
	// 创建目录
	if err := os.MkdirAll(path, 0777); err != nil {
		errString := fmt.Sprintf("%s MkdirAll Failed path=[%s] err=[%s]", funName, path, err.Error())
		log4plus.Error(errString)
		return errors.New(errString)
	}
	if err := os.Chmod(path, 0777); err != nil {
		errString := fmt.Sprintf("%s Chmod Failed path=[%s] err=[%s]", funName, path, err.Error())
		log4plus.Error(errString)
		return errors.New(errString)
	}
	if info, err := os.Stat(path); err == nil {
		errString := fmt.Sprintf("%s Stat mode=[%o]", funName, info.Mode().Perm())
		log4plus.Info(errString)
	}
	return nil
}

func (n *NodeDocker) parseStartCMD(startCMD, saveDir string) (DockerCompose, string) {
	funName := "parseStartCMD"
	yamlFile := saveYamlFile(startCMD)
	dockerCompose := parseYaml(yamlFile)
	if curSaveDir, err := replacePwdMarker(saveDir); err == nil {
		n.Base.SaveDir = curSaveDir
		_ = setWritableDir(n.Base.SaveDir)
		log4plus.Info(fmt.Sprintf("%s replacePwdMarker success saveDir=[%s]", funName, n.Base.SaveDir))
	} else {
		log4plus.Error(fmt.Sprintf("%s replacePwdMarker failed saveDir=[%s]", funName, saveDir))
	}
	return dockerCompose, yamlFile
}

func (n *NodeDocker) parseStartCMDArray(startCMD string) []string {
	funName := "parseStartCMDArray"
	cmd := struct {
		ComposeContent []string `json:"dockerComposes"`
	}{}
	if err := json.Unmarshal([]byte(startCMD), &cmd); err != nil {
		errString := fmt.Sprintf("%s json.Unmarshal Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return []string{}
	}
	return cmd.ComposeContent
}

func (n *NodeDocker) SetDocker(docker header.DockerArray) {
	funName := "SetDocker"
	n.Base.WorkSpaceID = docker.WorkSpaceID
	n.Base.RunningMode = docker.RunningMode
	n.Base.ComfyUIHttp = docker.ComfyUIHttp
	n.Base.ComfyUIWS = docker.ComfyUIWS
	n.Base.Deleted = docker.Deleted
	n.Base.SaveDir = docker.SaveDir
	n.State = docker.State
	n.Base.StartCMD = append(n.Base.StartCMD, docker.MajorCMD)
	if strings.Trim(docker.MinorCMD, " ") != "" {
		n.Base.StartCMD = append(n.Base.StartCMD, docker.MinorCMD)
	}
	//如何需要删除这个镜像，则轮询删除
	if n.Base.Deleted == header.Deleted {
		for _, startCMD := range n.Base.StartCMD {
			dockerCompose, _ := n.parseStartCMD(startCMD, n.Base.SaveDir)
			n.deleteImage(docker.WorkSpaceID, dockerCompose)
			return
		}
	}
	//检测容器已经启动
	for _, startCMD := range n.Base.StartCMD {
		dockerCompose, _ := n.parseStartCMD(startCMD, n.Base.SaveDir)
		for _, v := range dockerCompose.Services {
			log4plus.Info(fmt.Sprintf("%s isRunningContainer ContainerName=[%s]", funName, v.ContainerName))
			containerExist := isRunningContainer(n.dockerClient, v.ContainerName)
			if containerExist {
				//startup completed
				log4plus.Info(fmt.Sprintf("%s isRunningContainer containerExist is true ContainerName=[%s]", funName, v.ContainerName))
				n.State = header.StartupCompleted
				sendStartupComplete(n.Base.WorkSpaceID, true)
				if n.ComfyUI == nil {
					log4plus.Info(fmt.Sprintf("%s startup docker ComfyUIHttp=[%s] ComfyUIWS=[%s]", funName, n.Base.ComfyUIHttp, n.Base.ComfyUIWS))
					n.ComfyUI = plugIn.NewComfyUI(n.Base.ComfyUIHttp, n.Base.ComfyUIWS, n.Base.SaveDir, n.Base.WorkSpaceID, n.node)
				}
				log4plus.Info(fmt.Sprintf("%s startup docker", funName))
				return
			}
		}
	}
	//容器是否处于停止状态
	for _, startCMD := range n.Base.StartCMD {
		dockerCompose, _ := n.parseStartCMD(startCMD, n.Base.SaveDir)
		for _, v := range dockerCompose.Services {
			log4plus.Info(fmt.Sprintf("%s getExitedContainer ContainerName=[%s]", funName, v.ContainerName))
			containerExist := getExitedContainer(n.dockerClient, v.ContainerName)
			if containerExist {
				//startup
				log4plus.Info(fmt.Sprintf("%s getExitedContainer containerExist is true ContainerName=[%s]", funName, v.ContainerName))
				log4plus.Info(fmt.Sprintf("%s startupContainer ContainerName=[%s]", funName, v.ContainerName))
				if startupContainer(n.dockerClient, v.ContainerName) {
					sendStartupComplete(n.Base.WorkSpaceID, true)
					if n.ComfyUI == nil {
						log4plus.Info(fmt.Sprintf("%s startup docker ComfyUIHttp=[%s] ComfyUIWS=[%s]", funName, n.Base.ComfyUIHttp, n.Base.ComfyUIWS))
						n.ComfyUI = plugIn.NewComfyUI(n.Base.ComfyUIHttp, n.Base.ComfyUIWS, n.Base.SaveDir, n.Base.WorkSpaceID, n.node)
					}
					log4plus.Info(fmt.Sprintf("%s startup docker", funName))
					return
				}
			}
		}
	}
	//异常情况的处理
	if docker.State == header.Pulling || docker.State == header.PullFailed {
		for _, startCMD := range n.Base.StartCMD {
			dockerCompose, _ := n.parseStartCMD(startCMD, n.Base.SaveDir)
			n.deleteImage(docker.WorkSpaceID, dockerCompose)
		}
	} else if docker.State == header.StartupFailed {
		for _, startCMD := range n.Base.StartCMD {
			dockerCompose, _ := n.parseStartCMD(startCMD, n.Base.SaveDir)
			for _, v := range dockerCompose.Services {
				n.deleteContainer(docker.WorkSpaceID, v.ContainerName)
			}
		}
	}
	//启动
	n.State = header.Startuping
	n.StartupDockerCh <- struct{}{}
}

func checkDockerDaemon() error {
	funName := "checkDockerDaemon"
	switch runtime.GOOS {
	case "windows":
		if err := exec.Command("wsl", "--list", "--quiet").Run(); err != nil {
			errString := fmt.Sprintf("%s cmd.Run failed wsl not running", funName)
			log4plus.Error(errString)
			return errors.New(errString)
		}
		return nil
	case "linux", "darwin":
		if err := exec.Command("docker", "info").Run(); err != nil {
			errString := fmt.Sprintf("%s The Docker daemon is not running", funName)
			log4plus.Error(errString)
			return errors.New(errString)
		}
		return nil
	default:
		errString := fmt.Sprintf("%s The Docker daemon is not running", funName)
		log4plus.Error(errString)
		return fmt.Errorf("%s Unsupported operating system=[%s]", funName, runtime.GOOS)
	}
}

func validateImageReference(imageRef string) error {
	funName := "validateImageReference"
	pattern := `^([a-z0-9-]+\.?[a-z0-9-]+\/)?[a-z0-9_\/-]+(:[a-zA-Z0-9_.-]+)?$`
	matched, _ := regexp.MatchString(pattern, imageRef)
	if !matched {
		errString := fmt.Sprintf("%s regexp.MatchString Failed [registry/]repository/image:tag", funName)
		log4plus.Error(errString)
		return errors.New(errString)
	}
	return nil
}

func (n *NodeDocker) deleteContainer(workspaceID string, containerName string) {
	funName := "deleteContainer"
	log4plus.Info(fmt.Sprintf("%s deleteContainer workspaceID=[%s] containerName=[%s]", funName, workspaceID, containerName))
	cleanupContainer(containerName)

	log4plus.Info(fmt.Sprintf("%s proxy stop workspaceID=[%s] containerName=[%s]", funName, workspaceID, containerName))
	n.State = header.PullCompleted
	if n.ComfyUI != nil {
		n.ComfyUI.Stop()
		n.ComfyUI = nil
	}
	log4plus.Info(fmt.Sprintf("%s sendDeleteContainer workspaceID=[%s] containerName=[%s]", funName, workspaceID, containerName))
	sendDeleteContainer(workspaceID, "")
}

type PullMode int

const (
	noneMode PullMode = iota + 1
	dockerMode
	wgetMode
)

func (n *NodeDocker) pullMode(pullCMD string) PullMode {
	return dockerMode
}

type VolumeInfo struct {
	HostPath      string
	ContainerPath string
	Mode          string
}

func findImageIndex(args []string) int {
	optionWithValue := map[string]bool{
		"--name": true, "-p": true, "-e": true, "-v": true,
		"--gpus": true, "--volume": true, "--env": true,
		"--user": true, "--cap-add": true,
	}
	i := 2
	for i < len(args) {
		arg := args[i]
		cleanedArg := strings.Trim(strings.ReplaceAll(arg, `\\`, `\`), `"`)
		key := strings.SplitN(cleanedArg, "=", 2)[0]
		if optionWithValue[key] {
			if strings.Contains(cleanedArg, "=") {
				i++
			} else {
				if i+1 < len(args) {
					i += 2
				} else {
					i++
				}
			}
			continue
		}
		if strings.HasPrefix(arg, "-") {
			i++
			continue
		}
		if isValidImageName(cleanedArg) {
			return i
		} else {
			i++
		}
	}
	return -1
}

func isValidImageName(s string) bool {
	hasValidSeparator := strings.Contains(s, "/") || strings.Contains(s, ":")
	noSpaces := !strings.Contains(s, " ")
	return hasValidSeparator && noSpaces
}
func appendSystemArgs(args []string) []string {
	systemArgs := []string{"--user", "root:root", "--cap-add", "DAC_OVERRIDE"}
	return append(args[:2], append(systemArgs, args[2:]...)...)
}

func processOriginalArgs(newArgs []string, original []string) []string {
	for i := 0; i < len(original); {
		arg := original[i]
		if isVolumeArg(arg) {
			if strings.HasPrefix(arg, "--volume") {
				if eqIndex := strings.Index(arg, "="); eqIndex != -1 {
					i++
				} else {
					i += 2
				}
			} else {
				if len(arg) > 2 {
					i++
				} else {
					i += 2
				}
			}
		} else {
			newArgs = append(newArgs, arg)
			i++
		}
	}
	return newArgs
}

func isVolumeArg(arg string) bool {
	return strings.HasPrefix(arg, "-v") || strings.HasPrefix(arg, "--volume")
}

func appendVolumes(newArgs []string, volumes []VolumeInfo) []string {
	for _, vol := range volumes {
		volumeArg := buildVolumeArg(vol)
		newArgs = append(newArgs, "-v", volumeArg)
	}
	return newArgs
}

func cleanCommand(args []string) string {
	command := strings.Join(args, " ")
	command = regexp.MustCompile(`\\+`).ReplaceAllString(command, "")
	command = regexp.MustCompile(`\s+`).ReplaceAllString(command, " ")
	return strings.TrimSpace(command)
}

func splitCommandArgs(command string) []string {
	var args []string
	var currentArg strings.Builder
	var inSingleQuote, inDoubleQuote, escapeNext bool
	for _, r := range command {
		c := string(r)
		if escapeNext {
			currentArg.WriteString(c)
			escapeNext = false
			continue
		}
		switch {
		case c == `\` && inDoubleQuote:
			escapeNext = true
		case c == `"` && !inSingleQuote:
			inDoubleQuote = !inDoubleQuote
		case c == `'` && !inDoubleQuote:
			inSingleQuote = !inSingleQuote
		case (c == " " || c == "\t") && !inSingleQuote && !inDoubleQuote:
			if currentArg.Len() > 0 {
				args = append(args, currentArg.String())
				currentArg.Reset()
			}
		default:
			currentArg.WriteString(c)
		}
	}
	if currentArg.Len() > 0 {
		args = append(args, currentArg.String())
	}
	return args
}

func rebuildDockerCommand(volumes []VolumeInfo, originalCommand string) string {
	args := splitCommandArgs(originalCommand)
	imageIndex := findImageIndex(args)
	if imageIndex == -1 {
		return originalCommand
	}
	var newArgs []string
	newArgs = append(newArgs, args[0], args[1])
	newArgs = appendSystemArgs(newArgs)
	processedArgs := processOriginalArgs([]string{}, args[2:imageIndex])
	newArgs = append(newArgs, processedArgs...)
	newArgs = appendVolumes(newArgs, volumes)
	newArgs = append(newArgs, args[imageIndex:]...)
	return cleanCommand(newArgs)
}

func parseDockerVolumes(command string) []VolumeInfo {
	args := splitCommandArgs(command)
	var volumes []VolumeInfo
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg != "-v" && arg != "--volume" {
			continue
		}
		if i+1 >= len(args) {
			continue
		}
		value := args[i+1]
		i++
		vol := parseVolumeValue(value)
		if vol != nil {
			volumes = append(volumes, *vol)
		}
	}
	return volumes
}

func parseVolumeValue(value string) *VolumeInfo {
	value = strings.ReplaceAll(value, `\"`, `"`)
	value = strings.ReplaceAll(value, `\'`, `'`)
	value = strings.Trim(value, `"'`)
	parts := strings.SplitN(value, ":", 3)
	if len(parts) < 2 {
		return nil
	}
	vol := &VolumeInfo{
		HostPath:      parts[0],
		ContainerPath: parts[1],
	}
	if len(parts) >= 3 {
		vol.Mode = parts[2]
	}
	return vol
}

func resolveCommandSubstitutions(path string) (string, error) {
	path = strings.ReplaceAll(path, `"`, "")
	if strings.Contains(path, "$(pwd)") {
		currentDir, err := os.Getwd()
		if err != nil {
			return "", err
		}
		path = strings.ReplaceAll(path, "$(pwd)", currentDir)
	}
	return os.ExpandEnv(path), nil
}

func convertToWSLPath(hostPath string) (string, error) {
	resolvedPath, err := resolveCommandSubstitutions(hostPath)
	if err != nil {
		return "", err
	}
	resolvedPath = strings.TrimSpace(resolvedPath)
	//if runtime.GOOS == "windows" {
	//	volumeName := filepath.VolumeName(resolvedPath)
	//	if volumeName != "" {
	//		driveLetter := strings.ToLower(string(volumeName[0]))
	//		unixPath := fmt.Sprintf("/mnt/%s%s", driveLetter, filepath.ToSlash(resolvedPath[len(volumeName):]))
	//		return unixPath, nil
	//	}
	//}
	return filepath.ToSlash(resolvedPath), nil
}

func buildVolumeArg(vol VolumeInfo) string {
	mode := ""
	if vol.Mode != "" {
		mode = ":" + vol.Mode
	}
	return fmt.Sprintf(`"%s":"%s"%s`, vol.HostPath, vol.ContainerPath, mode)
}

func rebuildStartCMD(startCMD string) string {
	funName := "rebuildStartCMD"
	volumes := parseDockerVolumes(startCMD)
	for i := range volumes {
		wslPath, err := convertToWSLPath(volumes[i].HostPath)
		if err == nil {
			volumes[i].HostPath = wslPath
		}
	}
	newCommand := rebuildDockerCommand(volumes, startCMD)
	log4plus.Info(fmt.Sprintf("%s newCMD=[%s]", funName, newCommand))
	return newCommand
}

func parseDockerCommand(cmdStr string) ([]string, error) {
	funName := "parseDockerCommand"
	args, err := shlex.Split(cmdStr)
	if err != nil {
		errString := fmt.Sprintf("%s parse param failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return nil, errors.New(errString)
	}
	if len(args) < 2 || args[0] != "docker" || args[1] != "run" {
		errString := fmt.Sprintf("%s cmd is not docker command cmd=[%s]", funName, cmdStr)
		log4plus.Error(errString)
		return nil, errors.New(errString)
	}
	args = args[2:] // 保留后续参数
	var volumes []string
	for i := 0; i < len(args); i++ {
		if args[i] == "-v" && i+1 < len(args) {
			vol := strings.Trim(args[i+1], `"`)
			volumes = append(volumes, vol)
			i++
		}
	}
	return volumes, nil
}

func getSaveDir(startCMD string) string {
	funName := "getSaveDir"
	if volumes, err := parseDockerCommand(startCMD); err == nil {
		for _, vol := range volumes {
			parts := strings.SplitN(vol, ":", 2)
			if len(parts) == 2 {
				log4plus.Info(fmt.Sprintf("%s saveDir=[%s]", funName, parts[0]))
				return parts[0]
			} else {
				return ""
			}
		}
	}
	return ""
}

func (n *NodeDocker) waitStartup(cmd *exec.Cmd, containerName string) bool {
	funName := "waitStartup"
	checkTicker := time.NewTicker(time.Duration(5) * time.Second)
	defer checkTicker.Stop()

	if err := cmd.Wait(); err != nil {
		errString := fmt.Sprintf("%s cmd.Wait() Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return false
	}
	n.State = header.Startuping
	for {
		select {
		case <-checkTicker.C:
			if !checkStartup(n.dockerClient, containerName) {
				continue
			} else {
				n.State = header.StartupCompleted
				return true
			}
		}
	}
}

func findContainerByImage(imageName string) (string, error) {
	funName := "findContainerByImage"
	cmd := exec.Command(
		"docker", "ps", "-a",
		"--filter", "ancestor="+imageName,
		"--format", "{{.Names}}",
		"--latest",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		errString := fmt.Sprintf("%s run docker ps failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return "", errors.New(errString)
	}
	name := strings.TrimSpace(string(output))
	if name == "" {
		errString := fmt.Sprintf("%s not found container", funName)
		log4plus.Error(errString)
		return "", errors.New(errString)
	}
	return name, nil
}

func execDockerCompose(yamlFile string, index int64) (bool, *exec.Cmd) {
	funName := "execDockerCompose"
	cmdString := fmt.Sprintf("docker-compose -f %s up -d", yamlFile) //docker compose V1
	log4plus.Info(fmt.Sprintf("%s startup index=[%d] cmd=[%s]", funName, index, cmdString))
	argsV1 := strings.Fields(cmdString)
	cmdV1 := exec.Command(argsV1[0], argsV1[1:]...)
	cmdV1.Stdout = os.Stdout
	cmdV1.Stderr = os.Stderr
	if err := cmdV1.Start(); err == nil {
		return true, cmdV1
	}
	cmdString = fmt.Sprintf("docker-compose -f %s up -d", yamlFile)
	log4plus.Info(fmt.Sprintf("%s exec docker-compose failed index=[%d] cmd=[%s]", funName, index, cmdString))

	cmdString = fmt.Sprintf("docker compose -f %s up -d", yamlFile) //docker compose V1
	log4plus.Info(fmt.Sprintf("%s startup index=[%d] cmd=[%s]", funName, index, cmdString))
	argsV2 := strings.Fields(cmdString)
	cmdV2 := exec.Command(argsV2[0], argsV2[1:]...)
	cmdV2.Stdout = os.Stdout
	cmdV2.Stderr = os.Stderr
	if err := cmdV2.Start(); err == nil {
		return true, cmdV2
	}
	cmdString = fmt.Sprintf("docker compose -f %s up -d", yamlFile)
	log4plus.Info(fmt.Sprintf("%s exec docker compose failed index=[%d] cmd=[%s]", funName, index, cmdString))
	return false, nil
}

func (n *NodeDocker) startupDockerhub(compose DockerCompose, yamlFile string, index int64) bool {
	funName := "startupDockerhub"

	success, cmd := execDockerCompose(yamlFile, index)
	if !success {
		errString := fmt.Sprintf("%s cmd.Start() Failed ", funName)
		log4plus.Error(errString)
		sendStartupFail(n.Base.WorkSpaceID, errString)
		return false
	}

	//cmdString := fmt.Sprintf("docker-compose -f %s up -d", yamlFile)
	//log4plus.Info(fmt.Sprintf("%s startup index=[%d] cmd=[%s]", funName, index, cmdString))
	//args := strings.Fields(cmdString)
	//cmd := exec.Command(args[0], args[1:]...)
	//cmd.Stdout = os.Stdout
	//cmd.Stderr = os.Stderr
	//if err := cmd.Start(); err != nil {
	//	errString := fmt.Sprintf("%s cmd.Start() Failed err=[%s]", funName, err.Error())
	//	log4plus.Error(errString)
	//	sendStartupFail(n.Base.WorkSpaceID, errString)
	//	return false
	//}
	sendStartupWorkSpace(n.WorkSpaceID)
	containerName := ""
	for _, v := range compose.Services {
		containerName = v.ContainerName
		break
	}
	startUp := n.waitStartup(cmd, containerName)
	log4plus.Info(fmt.Sprintf("%s waitStartup Result startUp=[%t]", funName, startUp))
	sendStartupComplete(n.WorkSpaceID, startUp)
	if startUp {
		sendStartupComplete(n.WorkSpaceID, true)
		if n.ComfyUI == nil {
			log4plus.Info(fmt.Sprintf("%s startup docker ComfyUIHttp=[%s] ComfyUIWS=[%s]", funName, n.Base.ComfyUIHttp, n.Base.ComfyUIWS))
			n.ComfyUI = plugIn.NewComfyUI(n.Base.ComfyUIHttp, n.Base.ComfyUIWS, n.Base.SaveDir, n.Base.WorkSpaceID, n.node)
		}
	}
	log4plus.Info(fmt.Sprintf("%s startup docker success=[%t]", funName, true))
	return startUp
}

func (n *NodeDocker) startupContainer() {
	funName := "startupContainer"
	if n.Base.RunningMode == header.ComposeMode {
		//先从docker hub中拉取
		for index, startCMD := range n.Base.StartCMD {
			dockerCompose, yamlFile := n.parseStartCMD(startCMD, n.Base.SaveDir)
			if result := n.startupDockerhub(dockerCompose, yamlFile, int64(index)); result {
				log4plus.Info(fmt.Sprintf("%s startup docker success index=[%d] startCMD=[%s]", funName, index, startCMD))
				return
			}
		}
	}
}

func (n *NodeDocker) deleteImage(workspaceID string, compose DockerCompose) {
	funName := "deleteImage"
	for _, v := range compose.Services {
		imageName := v.Image
		containerName := v.ContainerName
		//delete container
		log4plus.Debug(fmt.Sprintf("%s deleteContainer containerName=[%s]", funName, containerName))
		n.deleteContainer(workspaceID, containerName)
		//delete image
		log4plus.Debug(fmt.Sprintf("%s deleteImage imageName=[%s]", funName, imageName))
		_ = deleteImage2(n.dockerClient, imageName)
	}
	n.State = header.UnInstalled
	n.Base.WorkSpaceID = ""
	n.Base.RunningMode = header.DockerRun
	n.Base.StartCMD = n.Base.StartCMD[:0]
	n.Base.SaveDir = ""
	n.Base.GPUMem = ""
	if n.ComfyUI != nil {
		n.ComfyUI.Stop()
		n.ComfyUI = nil
	}
	sendDeleteImage(workspaceID)
}

func (n *NodeDocker) checkContainer(cli *client.Client, containerName string) {
	funName := "checkContainer"
	containers, err := cli.ContainerList(
		context.Background(),
		types.ContainerListOptions{All: false},
	)
	if err != nil {
		errString := fmt.Sprintf("%s cli.ContainerList Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return
	}
	for _, container := range containers {
		normalizedName := strings.TrimPrefix(container.Names[0], "/")
		if strings.EqualFold(containerName, normalizedName) {
			n.ContainerID = container.ID[:12]
			info, errRes := n.dockerClient.ContainerInspect(context.Background(), n.ContainerID)
			if errRes != nil {
				errString := fmt.Sprintf("%s ContainerInspect err=[%s] ", funName, errRes.Error())
				log4plus.Error(errString)
				return
			}
			n.Base.SaveDir = info.GraphDriver.Data["MergedDir"]
			n.State = header.StartupCompleted
			gpuMem, _ := exec.Command("nvidia-smi", "--query-gpu=memory.used", "--format=csv,noheader,nounits").Output()
			n.Base.GPUMem = strings.TrimSpace(string(gpuMem))
			n.Base.ServerVer, n.Base.ClientVer = dockerVersion(cli)
			return
		}
	}
}

func (n *NodeDocker) checkTicker() {
	checkTicker := time.NewTicker(30 * time.Second)
	defer checkTicker.Stop()

	for {
		select {
		case <-checkTicker.C:
			if n.Base.RunningMode == header.ComposeMode {
				for _, v := range n.Compose.Services {
					n.checkContainer(n.dockerClient, v.ContainerName)
				}
			}
		case <-n.StartupDockerCh:
			n.startupContainer()
		case workspaceId := <-n.DelDockerCh:
			n.deleteImage(workspaceId, n.Compose)
		case message := <-n.MeasureCh:
			if n.ComfyUI != nil {
				n.ComfyUI.NewMeasureTask(message)
			}
		case message := <-n.TaskCh:
			if n.ComfyUI != nil {
				n.ComfyUI.NewTask(message)
			}
		case <-n.ExitCh:
			if n.ComfyUI != nil {
				n.ComfyUI.Stop()
				n.ComfyUI = nil
			}
			return
		}
	}
}

func (n *NodeDocker) Stop() {
	close(n.ExitCh)
}

func NewNodeDocker(workSpaceId string, node *Node) *NodeDocker {
	funName := "NewNodeDocker"
	nodeDocker := &NodeDocker{
		node:            node,
		WorkSpaceID:     workSpaceId,
		State:           header.UnInstalled,
		dockerClient:    nil,
		ComfyUI:         nil,
		ExitCh:          make(chan struct{}),
		StartupDockerCh: make(chan struct{}),
		DelDockerCh:     make(chan string, 1024),
		MeasureCh:       make(chan *header.MeasureMessage, 1024),
		TaskCh:          make(chan *header.PublishMessage, 1024),
	}
	if isLinux() {
		if _, err := os.Stat("/var/run/docker.sock"); os.IsNotExist(err) {
			errString := fmt.Sprintf("%s Docker socket not found. Please ensure the user has permission to access it err=[%s]", funName, err.Error())
			log4plus.Error(errString)
			return nil
		}
	}
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		errString := fmt.Sprintf("%s NewClientWithOpts Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return nil
	}
	nodeDocker.dockerClient = cli
	go nodeDocker.checkTicker()
	return nodeDocker
}
