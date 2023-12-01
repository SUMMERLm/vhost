package common

// default values
const (
	FrontendAliyunPkgExist    = true
	FrontendAliyunPkgNoExist  = false
	FrontendAliyunPkgUpdate   = true
	FrontendAliyunPkgNoUpdate = false

	FrontendAliyunCMExist            = true
	FrontendAliyunCMNoExist          = false
	FrontendAliyunFrontendDomainBase = "foolishtoohungry.com"

	//FrontendAliyunCdnVhostBasePath = "/var/www/hyperos/"
	FrontendAliyunCdnPkgBasePath = "/var/www/hyperos/frontend/"

	FrontendAliyunCdnPkgManageUrl = "http://127.0.0.1:8000/front/pkg/downloadFiles"
	FrontendAliyunCdnVhostName    = "nginx-config"

	FrontendAliyunVhostFinalizers = "apps.gaia.io/vhostfinalizer"
)
