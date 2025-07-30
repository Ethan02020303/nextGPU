package web

import (
	"github.com/gin-gonic/gin"
	log4plus "github.com/nextGPU/include/log4go"
	"github.com/nextGPU/ng-backend/configure"
	"github.com/nextGPU/ng-backend/middleware"
	"github.com/nextGPU/ng-backend/net/web/business"
)

type Web struct {
	webGin *gin.Engine
}

var gWeb *Web

func (w *Web) start() {
	funName := "start"
	log4plus.Info("%s user gin listen", funName)
	nodeGroup := w.webGin.Group("/backend")
	//nodeGroup.Use(business.SingletonBackend().SSOAuthMiddleware())
	{
		business.SingletonBackend().Start(nodeGroup)
	}
	log4plus.Info("%s start Run Listen=[%s]", funName, configure.SingletonConfigure().Net.Web.Listen)
	if err := w.webGin.Run(configure.SingletonConfigure().Net.Web.Listen); err != nil {
		log4plus.Error("%s Run Failed Not Use Http Error=[%s]", funName, err.Error())
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
