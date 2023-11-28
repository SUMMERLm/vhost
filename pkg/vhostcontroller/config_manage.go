package pkg

import (
	"context"
	"fmt"
	vhostV1 "github.com/SUMMERLm/vhost/pkg/apis/frontend/v1"
	"github.com/SUMMERLm/vhost/pkg/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	"github.com/tufanbarisyildirim/gonginx/parser"
)

type configOfSourceVhost struct {
}

func (c *Controller) configManage(vhost *vhostV1.Vhost) error {
	state, err := c.configState(vhost)
	if err != nil {
		klog.Errorf("Failed to get config state of vhost %q, error == %v", vhost.Name, err)
		return err
	}
	if !state {
		err := c.configVhost(vhost)
		if err != nil {
			klog.Errorf("Failed to config state of vhost %q, error == %v", vhost.Name, err)
			return err
		}
	}
	return nil
}

// https://github.com/tufanbarisyildirim/gonginx/blob/5dd06bb2938bd0e60810605960a1ae3f6cb273c3/examples/update-directive/main.go
func (c *Controller) configState(vhost *vhostV1.Vhost) (bool, error) {
	//vhost config检查

	configmap, err := c.kubeclientset.CoreV1().ConfigMaps(vhost.Namespace).Get(context.TODO(), common.FrontendAliyunCdnVhostName, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("Failed to get configmap of vhost %q, error == %v", vhost.Name, err)
		return common.FrontendAliyunCMNoExist, err
	}
	frontendDomainName := vhost.Name + "." + vhost.Spec.DomainName
	_, ok := configmap.Data[frontendDomainName]
	if !ok {
		return common.FrontendAliyunCMNoExist, nil
	}
	return common.FrontendAliyunCMExist, nil
}

func (c *Controller) configVhost(vhost *vhostV1.Vhost) error {
	//vhost config检查

	configmap, err := c.kubeclientset.CoreV1().ConfigMaps(vhost.Namespace).Get(context.TODO(), common.FrontendAliyunCdnVhostName, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("Failed to get configmap of vhost %q, error == %v", vhost.Name, err)
		return err
	}
	frontendDomainName := vhost.Name + "." + vhost.Spec.DomainName
	datamap, ok := configmap.Data[frontendDomainName]
	if !ok {
		return nil
	}
	p := parser.NewStringParser(datamap)
	cc := p.Parse()
	directives := cc.FindDirectives("server")
	for _, directive := range directives {
		fmt.Println("found a server :  ", directive.GetName(), directive.GetParameters())
		if directive.GetParameters()[0] == "http://www.google.com/" {
			directive.GetParameters()[0] = "http://www.duckduckgo.com/"
		}
	}
	return nil
}

func (c *Controller) configNew(vhost *vhostV1.Vhost) error {
	//vhost配置新建，对应nginx的vhost配置管理
	//	映射到configmap的配置
	//  configmap增加新建host的配置
	return nil
}

// TODO next s
func (c *Controller) configUpdate(vhost *vhostV1.Vhost) error {
	return nil
}

func (c *Controller) configRecycle(vhost *vhostV1.Vhost) error {
	//vhost配置删除，对应nginx的vhost配置管理
	//	映射到configmap的配置
	//  configmap删除host的配置
	return nil
}

func (c *Controller) offLine(vhost *vhostV1.Vhost) error {
	//TODO delete pkg
	//     config delete
	c.pkgRecycle(vhost)
	c.configRecycle(vhost)
	//TODO 延迟删除标志删除
	return nil
}
