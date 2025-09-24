package plugins

import (
	"github.com/liweiming-nova/common/config"
	"github.com/liweiming-nova/common/config/options"
	"github.com/liweiming-nova/common/config/parser"
)

type ConfigPlugin struct {
	file string
	data interface{}
}

func NewConfigPlugin(file string, data interface{}) *ConfigPlugin {
	return &ConfigPlugin{file: file, data: data}
}

func (plugin *ConfigPlugin) Name() (r string) {
	return "config"
}

func (plugin *ConfigPlugin) Start(ctx *PluginContext) (err error) {
	config.NewConfig(parser.NewTomlParser(),
		options.WithCfgSource(plugin.file),
		options.WithOpOnErrorFn(func(e error) { err = e }))
	return
}

func (plugin *ConfigPlugin) Stop() (err error) {
	return
}

func (plugin *ConfigPlugin) BeforeStart(ctx *PluginContext) error {
	return nil
}
