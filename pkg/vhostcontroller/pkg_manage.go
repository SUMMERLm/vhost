package pkg

import (
	"archive/zip"
	"bufio"
	"bytes"
	"compress/gzip"
	"fmt"
	vhostV1 "github.com/SUMMERLm/vhost/pkg/apis/frontend/v1"
	"github.com/SUMMERLm/vhost/pkg/common"
	"io"
	"io/ioutil"
	"k8s.io/klog/v2"
	"net/http"
	"os"
	"path"
	"path/filepath"
)

func (c *Controller) pkgState(vhost *vhostV1.Vhost) (string, bool, bool, error) {
	//包路径下当前包是否存在
	//存在，则返回state：exist
	//不存在，则返回state：NoExist
	//文件夹是否存在
	//存在情况下，文件夹是否为空
	path := common.FrontendAliyunCdnPkgBasePath + vhost.Name + "." + vhost.Spec.DomainName + "/"
	pathExist, err := c.pathExists(path)
	if err != nil {
		klog.Errorf("Failed to judge path exist status %q, error == %v", path, err)
		return "", false, false, err
	}

	pathIsEmpty, err := c.isEmpty(path)
	if err != nil {
		klog.Errorf("Failed to judge path empty status  %q, error == %v", path, err)
		return "", false, false, err
	}
	return path, pathExist, pathIsEmpty, nil
}

func (c *Controller) isEmpty(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdir(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err
}

func (c *Controller) pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (c *Controller) pkgManage(vhost *vhostV1.Vhost) error {
	//http请求拉取包
	//拉取到的包放置在对应目录下
	//目录需要映射到主机目录（deploy的volume映射管理）
	path, pathExist, pathIsEmpty, err := c.pkgState(vhost)
	if err != nil {
		klog.Errorf("Failed to get pkg state of  %q, error == %v", vhost.Name, err)
		return err
	}
	if pathExist && !pathIsEmpty {
		//路径存在，空文件夹
		err = os.RemoveAll(path)
		if err != nil {
			klog.Errorf("Failed to remove path: %q, error == %v", path, err)
			return err
		}
		err = c.pkgOnline(vhost, path)
		if err != nil {
			klog.Errorf("Failed  pkg online   %q, error == %v", vhost.Name, err)
			return err
		}
	} else if pathExist && pathIsEmpty {
		err = c.pkgOnline(vhost, path)
		if err != nil {
			klog.Errorf("Failed  pkg online   %q, error == %v", vhost.Name, err)
			return err
		}
	} else if !pathExist {
		//路径不存在,创建该路径
		err = os.Mkdir(path, os.ModePerm)
		if err != nil {
			klog.Errorf("Failed to mkdir  %q, error == %v", path, err)
			return err
		}
		err = c.pkgOnline(vhost, path)
		if err != nil {
			klog.Errorf("Failed  pkg online   %q, error == %v", vhost.Name, err)
			return err
		}
	}
	return nil
}

func (c *Controller) pkgOnlineTest(vhost *vhostV1.Vhost, pathOfPkg string) error {
	//pkgName包括了包的路径。如 /hyperos/usersystem/user.gz
	pkgUrl := common.FrontendAliyunCdnPkgManageUrl + vhost.Spec.PkgName
	vhostPkgFileName := path.Base(pkgUrl)
	res, err := http.Get(pkgUrl)
	if err != nil {
		klog.Errorf("Failed to download file from url %q, error == %v", pkgUrl, err)
		return err
	}
	defer res.Body.Close()
	// 获得get请求响应的reader对象
	reader := bufio.NewReaderSize(res.Body, 32*1024)
	//路径优化：对齐基本目录+包名称
	gzippedFile, err := os.Create(pathOfPkg + vhostPkgFileName)
	if err != nil {
		klog.Errorf("Failed to create file %q, error == %v", vhostPkgFileName, err)
		return err
	}
	defer gzippedFile.Close()
	gzipWriter := gzip.NewWriter(gzippedFile)
	defer gzipWriter.Close()
	// 获得文件的writer对象
	written, err := io.Copy(gzipWriter, reader)
	if err != nil {
		klog.Errorf("Failed to copy file %q, error == %v", vhostPkgFileName, err)
		return err
	}
	gzipWriter.Flush()
	klog.V(4).Infof("Total length: %d", written)

	//unzip到当前目录

	return nil
}

func (c *Controller) pkgOnline(vhost *vhostV1.Vhost, pathOfPkg string) error {
	//pkgName包括了包的路径。如 /hyperos/usersystem/user.gz
	//pkgUrl := common.FrontendAliyunCdnPkgManageUrl + vhost.Spec.PkgName
	err := c.gzipManage(vhost, pathOfPkg)
	if err != nil {
		klog.Errorf("Failed to online pkg from url %q, error == %v", vhost.Name, err)
		return err
	}
	return nil
	//vhostPkgFileName := path.Base(pkgUrl)
	//res, err := http.Get(pkgUrl)
	//if err != nil {
	//	klog.Errorf("Failed to download file from url %q, error == %v", pkgUrl, err)
	//	return err
	//}
	//defer res.Body.Close()
	//// 获得get请求响应的reader对象
	//reader := bufio.NewReaderSize(res.Body, 32*1024)
	////路径优化：对齐基本目录+包名称
	//zippedFile, err := os.Create(pathOfPkg + vhostPkgFileName)
	//if err != nil {
	//	klog.Errorf("Failed to create file %q, error == %v", vhostPkgFileName, err)
	//	return err
	//}
	//defer zippedFile.Close()
	////zipWriter := zip.NewWriter(zippedFile)
	////defer zipWriter.Close()
	//// 获得文件的writer对象
	//written, err := io.Copy(zippedFile, reader)
	//if err != nil {
	//	klog.Errorf("Failed to copy file %q, error == %v", vhostPkgFileName, err)
	//	return err
	//}
	////zippedFile.Flush()
	//klog.V(4).Infof("Total length: %d", written)
	//
	////unzip到当前目录
	//
	//return nil
}

func (c *Controller) gzipManage(vhost *vhostV1.Vhost, pathOfPkg string) error {
	//url := "https://example.com/archive.zip"
	url := common.FrontendAliyunCdnPkgManageUrl + vhost.Spec.PkgName
	//outputDir := "./output"
	outputDir := pathOfPkg
	// 创建输出目录
	err := os.MkdirAll(outputDir, os.ModePerm)
	if err != nil {
		klog.Errorf("Failed to mkdir  %q, error == %v", outputDir, err)
		return err
	}

	// 发起 HTTP 请求获取文件
	resp, err := http.Get(url)
	if err != nil {
		klog.Errorf("Failed to download file from url %q, error == %v", vhost.Name, err)
		return err
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	// 创建 zip.Reader
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		klog.Errorf("Failed to zip read %q, error == %v", vhost.Name, err)
		return err
	}

	// 遍历 zip 文件中的所有文件并解压缩
	for _, file := range reader.File {
		// 构造文件路径
		path := filepath.Join(outputDir, file.Name)

		if file.FileInfo().IsDir() {
			// 如果是目录，直接创建
			os.MkdirAll(path, file.Mode())
			continue
		}

		// 创建文件
		writer, err := os.Create(path)
		if err != nil {
			fmt.Println("创建文件失败:", err)
			klog.Errorf("Failed to create file %q, error == %v", vhost.Name, err)
			return err
		}
		defer writer.Close()

		// 打开 zip 文件中的文件
		reader, err := file.Open()
		if err != nil {
			fmt.Println("打开文件失败:", err)
			klog.Errorf("Failed to open file %q, error == %v", vhost.Name, err)

			return err
		}
		defer reader.Close()

		// 复制文件内容到输出文件中
		_, err = io.Copy(writer, reader)
		if err != nil {
			fmt.Println("复制文件内容失败:", err)
			klog.Errorf("Failed to copy file %q, error == %v", vhost.Name, err)

			return err
		}
	}

	fmt.Println("下载和解压缩完成!")
	klog.Info("download and unzip success of vhost: %q", vhost.Name)

	return nil
}

// TODO next s
func (c *Controller) pkgUpdate(vhost *vhostV1.Vhost) error {
	return nil
}

func (c *Controller) pkgRecycle(vhost *vhostV1.Vhost) error {
	//删除包
	//按需进行包删除，将相关包进行删除（deploy的volume映射管理）
	fileDir := common.FrontendAliyunCdnVhostBasePath
	vhostPkgFileName := vhost.Spec.PkgName
	_, err := os.Open(fileDir + "/" + vhostPkgFileName)
	//非空处理
	if err != nil {
		klog.Errorf("Failed to open file %q, error == %v", vhostPkgFileName, err)
	}
	// 删除指定目录下指定文件
	err = os.Remove(fileDir + "/" + vhostPkgFileName)
	if err != nil {
		klog.Errorf("Failed to remove file %q, error == %v", vhostPkgFileName, err)
		return err
	}
	return nil
}
