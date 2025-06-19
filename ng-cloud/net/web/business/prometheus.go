package business

import (
	"github.com/nextGPU/ng-cloud/process"
	log4plus "github.com/nextGPU/include/log4go"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"strings"
	"sync"
)

type (
	NodeGauge struct {
		Name  string
		Gauge prometheus.Gauge
	}
	NodeGaugeVec struct {
		Name       string
		LabelNames []string
		GaugeVec   *prometheus.GaugeVec
	}
	ServerMetrics struct {
		gauges    []*NodeGauge
		gaugesVec []*NodeGaugeVec
	}
)

type PrometheusServer struct {
	mutex      sync.Mutex
	serverData map[string]*ServerMetrics
	ExitCh     chan struct{}
}

var gPrometheusServer *PrometheusServer

func SingletonPrometheusServer() *PrometheusServer {
	if gPrometheusServer == nil {
		gPrometheusServer = &PrometheusServer{
			serverData: make(map[string]*ServerMetrics),
		}
		go gPrometheusServer.listenData()
	}
	return gPrometheusServer
}

func (a *PrometheusServer) listenData() {
	for {
		select {
		case msg := <-process.SingletonNodes().ExporterDataCh:
			a.SetData(msg)
		case <-a.ExitCh:
			return
		}
	}
}

func (a *PrometheusServer) init(systemUUID string) {
	if _, exists := a.serverData[systemUUID]; exists {
		return
	}
	a.serverData[systemUUID] = &ServerMetrics{}

	/****node_exporter****/
	a.addGaugeVec(systemUUID, "node_cpu_seconds_total", "CPU time spent in various modes", []string{"mode"})
	a.addGauge(systemUUID, "node_load1", "CPU Load Average (1 min)")
	a.addGauge(systemUUID, "node_load5", "CPU Load Average (5 min)")
	a.addGauge(systemUUID, "node_load15", "CPU Load Average (15 min)")

	/*memory*/
	a.addGauge(systemUUID, "node_memory_MemTotal_bytes", "Total memory in bytes")
	a.addGauge(systemUUID, "node_memory_MemAvailable_bytes", "Available memory in bytes")
	a.addGauge(systemUUID, "node_memory_SwapTotal_bytes", "Total swap memory in bytes")
	a.addGauge(systemUUID, "node_memory_SwapFree_bytes", "Free swap memory in bytes")

	/*disk*/
	a.addGaugeVec(systemUUID, "node_filesystem_size_bytes", "Filesystem size in bytes", []string{"device", "mountpoint", "fstype"})
	a.addGaugeVec(systemUUID, "node_filesystem_free_bytes", "Filesystem free space in bytes", []string{"device", "mountpoint", "fstype"})
	a.addGaugeVec(systemUUID, "node_disk_read_bytes_total", "Total number of bytes read from disk", []string{"device"})
	a.addGaugeVec(systemUUID, "node_disk_written_bytes_total", "Total number of bytes written to disk", []string{"device"})

	/*network*/
	a.addGaugeVec(systemUUID, "node_network_receive_bytes_total", "Total bytes received by network interface", []string{"device"})
	a.addGaugeVec(systemUUID, "node_network_transmit_bytes_total", "Total bytes transmitted by network interface", []string{"device"})

	/*file / process*/
	a.addGauge(systemUUID, "node_open_filedesc", "Number of open file descriptors")
	a.addGauge(systemUUID, "node_processes", "Total number of processes")

	/****gpu_exporter****/
	a.addGaugeVec(systemUUID, "gpu_memory_total_bytes", "Total GPU memory in bytes", []string{"gpu_id"})
	a.addGaugeVec(systemUUID, "gpu_memory_used_bytes", "Used GPU memory in bytes", []string{"gpu_id"})
	a.addGaugeVec(systemUUID, "gpu_memory_free_bytes", "Free GPU memory in bytes", []string{"gpu_id"})
	a.addGaugeVec(systemUUID, "gpu_utilization_percent", "GPU utilization percentage", []string{"gpu_id"})
	a.addGaugeVec(systemUUID, "gpu_temperature_celsius", "GPU temperature in Celsius", []string{"gpu_id"})
	a.addGaugeVec(systemUUID, "gpu_power_watts", "GPU power consumption in watts", []string{"gpu_id"})
	a.addGaugeVec(systemUUID, "gpu_clock_core_mhz", "GPU core clock speed in MHz", []string{"gpu_id"})
	a.addGaugeVec(systemUUID, "gpu_clock_memory_mhz", "GPU memory clock speed in MHz", []string{"gpu_id"})
	a.addGaugeVec(systemUUID, "gpu_fan_speed_percent", "GPU fan speed in percentage", []string{"gpu_id"})
}

func (p *PrometheusServer) addGauge(systemUUID, name, help string) {
	metrics := p.serverData[systemUUID]
	for _, v := range metrics.gauges {
		if strings.EqualFold(v.Name, name) {
			return
		}
	}
	nodeGauge := &NodeGauge{
		Name: name,
		Gauge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: name + "_" + systemUUID,
			Help: help,
		}),
	}
	prometheus.MustRegister(nodeGauge.Gauge)
	metrics.gauges = append(metrics.gauges, nodeGauge)
}

func (p *PrometheusServer) addGaugeVec(systemUUID, name, help string, labelNames []string) {
	metrics := p.serverData[systemUUID]
	for _, v := range metrics.gaugesVec {
		if strings.EqualFold(v.Name, name) {
			return
		}
	}
	nodeGaugeVec := &NodeGaugeVec{
		Name:       name,
		LabelNames: labelNames,
		GaugeVec: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: name + "_" + systemUUID,
			Help: help,
		}, labelNames),
	}
	prometheus.MustRegister(nodeGaugeVec.GaugeVec)
	metrics.gaugesVec = append(metrics.gaugesVec, nodeGaugeVec)
}

func (p *PrometheusServer) setGauge(systemUUID string, names []string, values []float64) {
	metrics := p.serverData[systemUUID]
	for i, name := range names {
		for _, v := range metrics.gauges {
			if v.Name == name {
				v.Gauge.Set(values[i])
				break
			}
		}
	}
}

func (p *PrometheusServer) setGaugeVec(systemUUID string, names []string, labels [][]string, values []float64) {
	metrics := p.serverData[systemUUID]
	for i, name := range names {
		for _, v := range metrics.gaugesVec {
			if strings.EqualFold(v.Name, name) {
				v.GaugeVec.WithLabelValues(labels[i]...).Set(values[i])
				break
			}
		}
	}
}

func (p *PrometheusServer) SetData(monitorData process.MonitorData) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if _, exists := p.serverData[monitorData.SystemUUID]; !exists {
		p.init(monitorData.SystemUUID)
	}
	var gaugeNames []string
	var gaugeValues []float64
	var gaugeVecNames []string
	var gaugeVecLabels [][]string
	var gaugeVecValues []float64
	metrics := p.serverData[monitorData.SystemUUID]
	for _, metric := range monitorData.NodeExporterData.Metrics {
		isGauge := false
		for _, g := range metrics.gauges {
			if strings.EqualFold(g.Name, metric.Name) {
				for _, sample := range metric.Samples {
					gaugeNames = append(gaugeNames, metric.Name)
					gaugeValues = append(gaugeValues, sample.Value)
				}
				isGauge = true
				break
			}
		}
		if isGauge {
			continue
		}
		isGaugeVec := false
		for _, gVec := range metrics.gaugesVec {
			if strings.EqualFold(gVec.Name, metric.Name) {
				isGaugeVec = true
				for _, sample := range metric.Samples {
					var labelValues []string
					for _, labelName := range gVec.LabelNames {
						value, exists := sample.Labels[labelName]
						if !exists {
							log4plus.Info("label %s not found for metric %s", labelName, metric.Name)
							labelValues = nil
							break
						}
						labelValues = append(labelValues, value)
					}
					if labelValues == nil {
						continue
					}
					if len(labelValues) != len(gVec.LabelNames) {
						log4plus.Info("mismatched labels for metric %s: expected %d, got %d", metric.Name, len(gVec.LabelNames), len(labelValues))
						continue
					}
					gaugeVecNames = append(gaugeVecNames, metric.Name)
					gaugeVecLabels = append(gaugeVecLabels, labelValues)
					gaugeVecValues = append(gaugeVecValues, sample.Value)
				}
				break
			}
		}
		if isGaugeVec {
			continue
		}
	}
	// 更新 Prometheus 指标
	p.setGauge(monitorData.SystemUUID, gaugeNames, gaugeValues)
	p.setGaugeVec(monitorData.SystemUUID, gaugeVecNames, gaugeVecLabels, gaugeVecValues)
}

func (p *PrometheusServer) Start(webGin *gin.RouterGroup) {
	webGin.GET("/metrics", gin.WrapH(promhttp.Handler()))
}
