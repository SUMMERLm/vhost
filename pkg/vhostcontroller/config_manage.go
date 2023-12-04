package pkg

import (
	"context"
	"fmt"
	vhostV1 "github.com/SUMMERLm/vhost/pkg/apis/frontend/v1"
	"github.com/SUMMERLm/vhost/pkg/common"
	"github.com/tufanbarisyildirim/gonginx"
	"github.com/tufanbarisyildirim/gonginx/parser"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
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
func (c *Controller) vhostConfig(vhost *vhostV1.Vhost) string {
	vhostString := (`server {
         listen 80;
         server_name vhost;
         root /var/www/vhost/;
         index index.html;
         location / {
         }
         }`)
	return vhostString
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
	//datamap, ok := configmap.Data[frontendDomainName]
	_, ok := configmap.Data[frontendDomainName]
	if !ok {
		//configmap.Data[frontendDomainName] = c.vhostConfig(vhost)
		//datamap, _ := configmap.Data[frontendDomainName]
		datamap := c.vhostConfig(vhost)
		p := parser.NewStringParser(datamap)
		cc := p.Parse()
		directives := cc.FindDirectives("server_name")
		for _, directive := range directives {
			fmt.Println("found a server_name :  ", directive.GetName(), directive.GetParameters())
			if directive.GetParameters()[0] == "vhost" {
				directive.GetParameters()[0] = vhost.Name + "." + vhost.Spec.DomainName
			}
		}
		datamap = gonginx.DumpBlock(cc.Block, gonginx.IndentedStyle)
		p = parser.NewStringParser(datamap)
		cc = p.Parse()
		directives = cc.FindDirectives("root")
		for _, directive := range directives {
			fmt.Println("found a root :  ", directive.GetName(), directive.GetParameters())
			if directive.GetParameters()[0] == "/var/www/vhost/" {
				directive.GetParameters()[0] = "/var/www/vhost/" + vhost.Name + "." + vhost.Spec.DomainName
			}
		}
		datamap = gonginx.DumpBlock(cc.Block, gonginx.IndentedStyle)
		configmap.Data[frontendDomainName+".conf"] = datamap
		_, err := c.kubeclientset.CoreV1().ConfigMaps(vhost.Namespace).Update(context.TODO(), configmap, metav1.UpdateOptions{})
		if err != nil {
			klog.Errorf("Failed to update  configmap of vhost %q, error == %v", vhost.Name, err)
			return err
		}
		return nil
	}

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
	configmap, err := c.kubeclientset.CoreV1().ConfigMaps(vhost.Namespace).Get(context.TODO(), common.FrontendAliyunCdnVhostName, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("Failed to get configmap of vhost %q, error == %v", vhost.Name, err)
		return err
	}
	frontendDomainName := vhost.Name + "." + vhost.Spec.DomainName+".conf"
	_, ok := configmap.Data[frontendDomainName]
	if ok {
		delete(configmap.Data, frontendDomainName)
		_, err := c.kubeclientset.CoreV1().ConfigMaps(vhost.Namespace).Update(context.TODO(), configmap, metav1.UpdateOptions{})
		if err != nil {
			klog.Errorf("Failed to update  configmap of vhost %q, error == %v", vhost.Name, err)
		}
	}
	vhostCopy := vhost.DeepCopy()
	vhostCopy.Finalizers = c.RemoveString(vhostCopy.Finalizers, common.FrontendAliyunVhostFinalizers)
	_, err = c.vhostclientset.FrontendsV1().Vhosts(vhost.Namespace).Update(context.TODO(), vhostCopy, metav1.UpdateOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		klog.WarningDepth(4, fmt.Sprintf("handleFrontend: failed to remove finalizer %s from frontend Descriptions %s: %v", common.FrontendAliyunVhostFinalizers, vhost, err))
	}

	return nil
}
func (c *Controller) RemoveString(slice []string, s string) []string {
	newSlice := make([]string, 0)
	for _, item := range slice {
		if item == s {
			continue
		}
		newSlice = append(newSlice, item)
	}
	if len(newSlice) == 0 {
		// Sanitize for unit tests so we don't need to distinguish empty array
		// and nil.
		newSlice = nil
	}
	return newSlice
}

func (c *Controller) offLine(vhost *vhostV1.Vhost) error {
	err := c.configRecycle(vhost)
	if err != nil {
		klog.Errorf("Failed to recycle  configmap of vhost %q, error == %v", vhost.Name, err)
		return err
	}
	err = c.pkgRecycle(vhost)
	if err != nil {
		klog.Errorf("Failed to recycle pkg of vhost %q, error == %v", vhost.Name, err)
		return err
	}

	return nil
}
