package route

// 路由包

import (
	"github.com/SUMMERLm/vhost/pkg/frontendpkgmanage/api"
	"github.com/gin-gonic/gin"
)

func InitRouter() *gin.Engine {
	router := gin.Default()
	// get Test
	router.GET("/", api.GetIndex)
	// post Test
	router.POST("/test", api.PostTest)
	//pkg upload
	router.POST("/front/pkg/upload", api.Upload)
	//multi pkg upload
	router.POST("/front/multi/pkg/upload", api.UploadMultiPkg)
	//pkg recycle
	router.POST("/front/pkg/delete", api.Recycle)
	router.GET("/front/pkg/downloadFiles", api.DownloadFileService)
	router.POST("/front/pkg/vhost/pkg/manage", api.VhostPkgManage)
	return router
}
