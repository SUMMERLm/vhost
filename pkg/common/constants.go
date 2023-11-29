package common

// default values
const (
	FrontendAliyunPkgExist    = true
	FrontendAliyunPkgNoExist  = false
	FrontendAliyunPkgUpdate   = true
	FrontendAliyunPkgNoUpdate = false

	FrontendAliyunCMExist   = true
	FrontendAliyunCMNoExist = false

	FrontendAliyunCdnVhostBasePath = "/var/www/hyperos/"
	FrontendAliyunCdnPkgBasePath   = "/etc/hyperos/frontend/"

	FrontendAliyunCdnPkgManageUrl = "http://ip+port/downloadFiles?"
	FrontendAliyunCdnVhostName    = "nginx-config"

	FrontendAliyunVhostFinalizers = "apps.gaia.io/vhostfinalizer"
)
