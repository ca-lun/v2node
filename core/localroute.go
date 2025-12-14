package core

import (
	"encoding/json"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/xtls/xray-core/core"
	coreConf "github.com/xtls/xray-core/infra/conf"
)

// LocalRouteConfig 本地路由配置结构
type LocalRouteConfig struct {
	Routing   *coreConf.RouterConfig          `json:"routing"`
	Outbounds []coreConf.OutboundDetourConfig `json:"outbounds"`
}

// LoadLocalRouteConfig 从本地文件加载路由配置
func LoadLocalRouteConfig(path string) (*LocalRouteConfig, error) {
	if path == "" {
		return nil, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			log.Warnf("Local route config file not found: %s", path)
			return nil, nil
		}
		return nil, fmt.Errorf("read local route config error: %w", err)
	}

	config := &LocalRouteConfig{}
	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("parse local route config error: %w", err)
	}

	outboundCount := len(config.Outbounds)
	ruleCount := 0
	if config.Routing != nil {
		ruleCount = len(config.Routing.RuleList)
	}

	log.Warnf("Loaded local route config from %s (outbounds: %d, rules: %d)", path, outboundCount, ruleCount)
	return config, nil
}

// BuildLocalOutbounds 构建本地outbound配置
func BuildLocalOutbounds(config *LocalRouteConfig) ([]*core.OutboundHandlerConfig, error) {
	if config == nil || len(config.Outbounds) == 0 {
		return nil, nil
	}

	var outbounds []*core.OutboundHandlerConfig
	for i := range config.Outbounds {
		built, err := config.Outbounds[i].Build()
		if err != nil {
			log.Warnf("Build local outbound %d error: %v", i, err)
			continue
		}
		outbounds = append(outbounds, built)
	}

	return outbounds, nil
}
