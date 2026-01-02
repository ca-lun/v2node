package core

import (
	"fmt"

	panel "github.com/wyx2685/v2node/api/v2board"
	"github.com/wyx2685/v2node/conf"
)

func (v *V2Core) AddNode(tag string, info *panel.NodeInfo) error {
	var fallbacks []conf.Fallback
	for _, nodeConfig := range v.Config.NodeConfigs {
		if nodeConfig.NodeID == info.Id {
			fallbacks = nodeConfig.Fallbacks
			break
		}
	}
	inBoundConfig, err := buildInbound(info, tag, fallbacks)
	if err != nil {
		return fmt.Errorf("build inbound error: %s", err)
	}
	err = v.addInbound(inBoundConfig)
	if err != nil {
		return fmt.Errorf("add inbound error: %s", err)
	}
	return nil
}

func (v *V2Core) DelNode(tag string) error {
	err := v.removeInbound(tag)
	if err != nil {
		return fmt.Errorf("remove in error: %s", err)
	}
	return nil
}
