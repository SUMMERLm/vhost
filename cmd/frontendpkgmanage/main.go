package main

import (
	"github.com/SUMMERLm/vhost/pkg/frontendpkgmanage/config"
	"github.com/SUMMERLm/vhost/pkg/frontendpkgmanage/route"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

func main() {
	r := route.InitRouter()
	//config init
	config.InitConfig()
	//listen on 8000
	gin.SetMode(viper.GetString("server.run_mode"))
	r.Run(viper.GetString("server.addr"))
}
