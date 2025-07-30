package configure

import (
	"encoding/json"
	"os"
	"time"

	log4plus "github.com/nextGPU/include/log4go"
)

type ApplicationConfig struct {
	Name    string `json:"name"`
	Comment string `json:"comment"`
}

type MysqlConfig struct {
	MysqlIp        string `json:"Ip"`
	MysqlPort      int    `json:"Port"`
	MysqlDBName    string `json:"DBName"`
	MysqlDBCharset string `json:"Charset"`
	UserName       string `json:"UserName"`
	Password       string `json:"Password"`
}

type WebConfig struct {
	Listen       string `json:"listen"`
	Domain       string `json:"domain"`
	ResourcePath string `json:"resourcePath"`
}

type WebsocketConfig struct {
	RoutePath string `json:"routePath"`
	Listen    string `json:"listen"`
	Domain    string `json:"domain"`
}

type NetConfig struct {
	Web  WebConfig       `json:"web"`
	Node WebsocketConfig `json:"node"`
	User WebsocketConfig `json:"user"`
}

type MeasureConfig struct {
	MaxUnPassPercent float64 `json:"unPassPercent"`
	MeasureCount     int64   `json:"measureCount"`
	MeasureWidth     int64   `json:"measureWidth"`
	MeasureHeight    int64   `json:"measureHeight"`
}

type AITaskConfig struct {
	AITaskRecheckDays int `json:"aiTaskRecheckDays"`
	MaxRetryCount     int `json:"maxRetryCount"`
}

type MinRequestConfig struct {
	Memory  uint64 `json:"memory"`
	CoreNum int64  `json:"core"`
}

type ClientVersionConfig struct {
	Version    string `json:"version"`
	CreateTime string `json:"createTime"`
	Domain     string `json:"domain"`
	UpdateFile string `json:"updateFile"`
}

type OSSConfig struct {
	Endpoint        string `json:"endpoint"`
	AccessKeyID     string `json:"accessKeyID"`
	AccessKeySecret string `json:"accessKeySecret"`
	BucketName      string `json:"bucketName"`
}

type WechatPaymentConfig struct {
	MCHid             string `json:"mchid"`
	SerialNo          string `json:"serialNo"`
	ApiV3Key          string `json:"apiV3Key"`
	PrivateKeyFile    string `json:"privateKeyFile"`
	AppID             string `json:"appID"`
	NotifyUrl         string `json:"notifyUrl"`
	ContractNotifyUrl string `json:"contractNotifyUrl"`
}

type AliPaymentConfig struct {
	AppID         string `json:"appID"`
	APPPrivateKey string `json:"appPrivateKey"`
	APPPublicKey  string `json:"appPublicKey"`
	AliPublicKey  string `json:"aliPublicKey"`
	AliRootKey    string `json:"aliRootKey"`
	IsProd        bool   `json:"isProd"`
	NotifyUrl     string `json:"notifyUrl"`
}

type PaymentConfig struct {
	Wechat WechatPaymentConfig `json:"wechat"`
	Ali    AliPaymentConfig    `json:"ali"`
}

type Config struct {
	Application   ApplicationConfig   `json:"application"`
	Mysql         MysqlConfig         `json:"mysql"`
	Net           NetConfig           `json:"net"`
	AliOSS        OSSConfig           `json:"aliOSS"`
	AliLogOSS     OSSConfig           `json:"aliLogOSS"`
	Measure       MeasureConfig       `json:"measure"`
	AITask        AITaskConfig        `json:"aiTask"`
	MinRequest    MinRequestConfig    `json:"minRequest"`
	ClientVersion ClientVersionConfig `json:"clientVersion"`
	Payment       PaymentConfig       `json:"payment"`
}

type Configure struct {
	config Config
}

var gConfigure *Configure

func (u *Configure) getConfig() error {
	funName := "getConfig"
	log4plus.Info("%s ---->>>>", funName)
	data, err := os.ReadFile("./config.json")
	if err != nil {
		log4plus.Error("%s ReadFile error=[%s]", funName, err.Error())
		return err
	}
	//log4plus.Info("%s data=[%s]", funName, string(data))
	err = json.Unmarshal(data, &u.config)
	if err != nil {
		log4plus.Error("%s json.Unmarshal error=[%s]", funName, err.Error())
		return err
	}
	return nil
}

func (u *Configure) pollReload() {
	reloadTicker := time.NewTicker(30 * time.Second)
	defer reloadTicker.Stop()

	for {
		select {
		case <-reloadTicker.C:
			_ = u.getConfig()
		}
	}
}

func SingletonConfigure() Config {
	if gConfigure == nil {
		gConfigure = &Configure{}
		_ = gConfigure.getConfig()
		go gConfigure.pollReload()
	}
	return gConfigure.config
}
