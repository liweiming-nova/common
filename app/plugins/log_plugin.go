package plugins

import "github.com/liweiming-nova/common/xlog"

type LogPlugin struct {
}

func NewLogPlugin() *LogPlugin {
	return &LogPlugin{}
}
func (p *LogPlugin) Start(ctx *PluginContext) error {
	return xlog.Start()
}
func (p *LogPlugin) BeforeStart(ctx *PluginContext) error {
	return nil
}

func (p *LogPlugin) Stop() error {
	return nil
}
