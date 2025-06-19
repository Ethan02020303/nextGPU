package web

import (
	"github.com/nextGPU/ng-cloud/configure"
	"github.com/nextGPU/ng-cloud/middleware"
	"github.com/nextGPU/ng-cloud/net/web/business"
	log4plus "github.com/nextGPU/include/log4go"
	"github.com/gin-gonic/gin"
)

type Web struct {
	webGin *gin.Engine
}

var gWeb *Web

func (w *Web) start() {
	funName := "start"
	log4plus.Info("start user gin listen")

	nodeGroup := w.webGin.Group("/node")
	{
		business.SingletonNode().Start(nodeGroup)
		business.SingletonPrometheusServer().Start(nodeGroup)
	}
	userGroup := w.webGin.Group("/session")
	{
		business.SingletonUser().Start(userGroup)
	}
	workspaceGroup := w.webGin.Group("/workspace")
	{
		business.SingletonWorkSpace().Start(workspaceGroup)
	}
	filesGroup := w.webGin.Group("/files")
	{
		business.SingletonNode().FilesStart(filesGroup)
	}
	log4plus.Info("%s start Run Listen=[%s]", funName, configure.SingletonConfigure().Net.Web.Listen)
	if err := w.webGin.Run(configure.SingletonConfigure().Net.Web.Listen); err != nil {
		log4plus.Error("start Run Failed Not Use Http Error=[%s]", err.Error())
		return
	}
}

func SingletonWeb() *Web {
	if gWeb == nil {
		gWeb = &Web{}
		log4plus.Info("Create Web Manager")
		gWeb.webGin = gin.Default()
		gWeb.webGin.Use(middleware.Cors())
		gin.SetMode(gin.DebugMode)
		go gWeb.start()
	}
	return gWeb
}
