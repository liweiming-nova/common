package plugins

import "context"

type PluginContext struct {
	context.Context
	AppVersion string
	WorkDir    string
	Data       map[string]interface{} // 插件间共享数据
}

func NewPluginContext(appVersion, workDir string) *PluginContext {
	return &PluginContext{
		Context:    context.Background(),
		AppVersion: appVersion,
		WorkDir:    workDir,
		Data:       make(map[string]interface{}),
	}
}

// Set / Get 用于插件间共享数据
func (c *PluginContext) Set(key string, value interface{}) {
	c.Data[key] = value
}

func (c *PluginContext) Get(key string) interface{} {
	return c.Data[key]
}

type Plugin interface {
	Start(ctx *PluginContext) error
	Stop() error
	BeforeStart(ctx *PluginContext) error // 启动前缀钩子
}
