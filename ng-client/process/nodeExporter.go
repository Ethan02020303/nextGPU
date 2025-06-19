package process

import (
	"errors"
	"fmt"
	log4plus "github.com/nextGPU/include/log4go"
	"github.com/nextGPU/include/prometheus"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

type NodeSample struct {
	Labels map[string]string `json:"labels,omitempty"`
	Value  float64           `json:"value"`
}

type NodeIndicator struct {
	Name    string       `json:"name"`
	Type    string       `json:"type"`
	Help    string       `json:"help"`
	Samples []NodeSample `json:"samples"`
}

type NodeExporter struct {
	parent    *Node
	Timestamp string
	Metrics   []NodeIndicator
	MsgCh     chan *prometheus.ExporterData
	ExitCh    chan struct{}
}

func (n *NodeExporter) monitor() {
	for {
		select {
		case monitorData := <-n.MsgCh:
			sendNodeExporter(monitorData)
		case <-n.ExitCh:
			return
		}
	}
}

func startNodeExporter(path string) error {
	funName := "startNodeExporter"

	argv := []string{path}
	// 进程属性配置
	attr := &os.ProcAttr{
		Files: []*os.File{
			os.Stdin,
			os.Stdout,
			os.Stderr,
		},
		Sys: &syscall.SysProcAttr{
			// 可选：设置子进程组或会话特性[4](@ref)
		},
	}
	process, errProcess := os.StartProcess(path, argv, attr)
	if errProcess != nil {
		errString := fmt.Sprintf("%s os.StartProcess err=[%s]", funName, errProcess.Error())
		log4plus.Error(errString)
		return errors.New(errString)
	}
	log4plus.Info(fmt.Sprintf("%s os.StartProcess success pID=[%d]", funName, process.Pid))
	_ = process.Release()
	return nil
}

func isRunning(processName string) bool {
	funName := "isRunning"
	cmd := exec.Command("ps", "-e", "-o", "command")
	output, err := cmd.Output()
	if err != nil {
		errString := fmt.Sprintf("%s exec.Command Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return false
	}
	return strings.Contains(string(output), processName)
}

func NewNodeExporter(monitors []string, parent *Node) *NodeExporter {
	funName := "NewNodeExporter"
	exporter := &NodeExporter{
		parent: parent,
		ExitCh: make(chan struct{}),
		MsgCh:  make(chan *prometheus.ExporterData, 1024),
	}
	processName := "node_exporter"
	binaryPath := "./node_exporter/node_exporter"
	if !isRunning(processName) {
		if err := startNodeExporter(binaryPath); err != nil {
			errString := fmt.Sprintf("%s startNodeExporter Failed err=[%s]", funName, err.Error())
			log4plus.Error(errString)
		}
	}
	errString := fmt.Sprintf("%s monitors=[%#v]", funName, monitors)
	log4plus.Info(errString)
	prometheus.SingletonNodeExporter("http://localhost:9100/metrics", 10, exporter.MsgCh).AddQuerys(monitors)
	return exporter
}
