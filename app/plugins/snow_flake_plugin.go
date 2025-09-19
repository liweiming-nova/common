package plugins

import (
	"github.com/liweiming-nova/common/utils"
	"math/rand"
)

type SnowFlakePlugin struct {
	NodeID int64
}

func NewSnowFlakePlugin(nodeIds ...int64) *SnowFlakePlugin {
	var node int64
	if len(nodeIds) == 0 {
		node = int64(rand.Intn(1024))
	} else {
		node = nodeIds[0]
	}
	return &SnowFlakePlugin{
		NodeID: node,
	}
}
func (p *SnowFlakePlugin) Start(ctx *PluginContext) error {
	err := utils.InitSnowflake(p.NodeID)
	if err != nil {
		return err
	}
	return nil
}
func (p *SnowFlakePlugin) BeforeStart(ctx *PluginContext) error {
	return nil
}

func (p *SnowFlakePlugin) Stop() error {
	return nil
}
