package app

import (
	"fmt"
	"github.com/liweiming-nova/common/app/plugins"
	"testing"
)

func TestApp(t *testing.T) {
	NewApp().Use(&TestPlugin{}).SetContext("opt", "name").Start()
}

type TestPlugin struct {
}

func (plugin *TestPlugin) BeforeStart(ctx *plugins.PluginContext) error {
	fmt.Println("before start")
	return nil
}
func (plugin *TestPlugin) Start(ctx *plugins.PluginContext) error {
	fmt.Println("plugin start")
	opt := ctx.Get("opt").(string)
	fmt.Println("opt", opt)
	return nil
}

func (plugin *TestPlugin) Stop() error {
	fmt.Println("plugin stop")
	return nil
}
