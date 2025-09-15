package plugins

import (
	"fmt"
	rest "github.com/liweiming-nova/common/httpx/server"
	"net/http"
)

type RestOptions func(plugins *RestPlugin)

type RestPlugin struct {
	name        string
	handlerFunc http.Handler
}

func WithHandlerFunc(handlerFunc http.Handler) RestOptions {
	return func(plugins *RestPlugin) {
		plugins.handlerFunc = handlerFunc
	}
}

func WithName(name string) RestOptions {
	return func(plugins *RestPlugin) {
		plugins.name = name
	}
}
func NewRestPlugin(opts ...RestOptions) (r *RestPlugin) {
	rest := &RestPlugin{}
	for _, opt := range opts {
		opt(rest)
	}
	return rest
}

func (plugin *RestPlugin) Start(ctx *PluginContext) (err error) {
	if err = rest.StartServe(plugin.name, plugin.handlerFunc); err != nil {
		err = fmt.Errorf("build rest service error: %s\n", err)
		return
	}
	fmt.Printf("App rest listening on %s\n", rest.Server(plugin.name).Addr)
	return
}

func (plugin *RestPlugin) Stop() (err error) {
	err = rest.StopServe(plugin.name)
	return
}

func (plugin *RestPlugin) BeforeStart(ctx *PluginContext) error {
	return nil
}
