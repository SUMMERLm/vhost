package api

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"io"
	"log"
	"net/http"
	"os"
)

func GetIndex(c *gin.Context) {
	c.String(200, "Hello, get test")

}

func PostTest(c *gin.Context) {
	c.String(200, "Hello，post test")
}

func Upload(c *gin.Context) {
	file, _ := c.FormFile("file")
	fileDir := c.Query("filedir")
	log.Println(file.Filename)
	//path := "/var/frontend/"
	path := viper.GetString("frontEnd.pkgPath") + fileDir
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := os.Mkdir(path, os.ModePerm)
		if err != nil {
			fmt.Println(err)
		}
	}
	dst := path + file.Filename
	c.SaveUploadedFile(file, dst)
	c.String(http.StatusOK, fmt.Sprintf("'%s' uploaded!", file.Filename))
}

func VhostPkgManage(c *gin.Context) {
	var err error
	src := c.PostForm("src")
	dst := c.PostForm("dst")
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		c.String(400, err.Error())
	}
	if !sourceFileStat.Mode().IsRegular() {
		c.String(400, err.Error())
	}
	source, err := os.Open(src)
	if err != nil {
		c.String(400, err.Error())
	}
	defer source.Close()
	destination, err := os.Create(dst)
	if err != nil {
		c.String(400, err.Error())
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	c.String(200, string(nBytes))
}

func UploadMultiPkg(c *gin.Context) {
	// Multipart form
	form, _ := c.MultipartForm()
	files := form.File["upload[]"]

	for _, file := range files {
		log.Println(file.Filename)
		//TODO 结合nginx vhost管理配置
		dst := "./" + file.Filename
		// 上传文件至指定目录
		c.SaveUploadedFile(file, dst)
	}
	c.String(http.StatusOK, fmt.Sprintf("%d files uploaded!", len(files)))
}

func Recycle(c *gin.Context) {
	// Multipart form
	fileDir := viper.GetString("frontEnd.pkgPath") + c.Query("filedir")
	fileName := c.Query("fileName")
	_, errByOpenFile := os.Open(fileDir + "/" + fileName)
	//非空处理
	if errByOpenFile != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "失败",
			"error":   "资源不存在",
		})
		c.Redirect(http.StatusFound, "/404")
		return
	}
	// 删除指定目录下指定文件
	os.Remove(fileDir + "/" + fileName)
}

func DownloadFileService(c *gin.Context) {
	fileDir := viper.GetString("frontEnd.pkgPath") + c.Query("fileDir")
	fileName := c.Query("fileName")
	_, errByOpenFile := os.Open(fileDir + "/" + fileName)
	//非空处理
	if errByOpenFile != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "失败",
			"error":   "资源不存在",
		})
		c.Redirect(http.StatusFound, "/404")
		return
	}
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", "attachment; filename="+fileName)
	c.Header("Content-Transfer-Encoding", "binary")
	c.File(fileDir + "/" + fileName)
	return
}
