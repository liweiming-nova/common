package app

import (
	"fmt"
	"github.com/liweiming-nova/common/app/plugins"
	"log"
	"os"
	"reflect"
)

var (
	__version__ string // go build -ldflags "-X git.bestfulfill.tech/common/x/app.__version__=$VERSION"
)

type App struct {
	version        string
	workDir        string
	plugins        []plugins.Plugin
	pluginsContext *plugins.PluginContext
}

func (app *App) Use(plugins ...plugins.Plugin) *App {
	app.plugins = append(app.plugins, plugins...)
	return app
}
func (app *App) SetContext(name string, data interface{}) *App {
	app.pluginsContext.Set(name, data)
	return app
}

func NewApp(opts ...Option) (app *App) {
	app = &App{
		version: __version__,
		plugins: make([]plugins.Plugin, 0, 10)}
	app.workDir, _ = os.Getwd()

	for _, opt := range opts {
		opt(app)
	}

	// 创建插件上下文
	pluginCtx := plugins.NewPluginContext(app.version, app.workDir)
	app.pluginsContext = pluginCtx

	return
}

func (app *App) Start() (err error) {
	if len(os.Args) >= 2 && (os.Args[1] == "-v" || os.Args[1] == "--version") {
		fmt.Printf("Version: %s\n", app.version)
		return
	}

	log.Printf("Running in %s\n", app.workDir)
	for _, plugin := range app.plugins {
		if err = plugin.BeforeStart(app.pluginsContext); err != nil {
			log.Printf("Plugin %s start fail, %s\n", reflect.TypeOf(plugin).Elem().Name(), err)
			break
		}
		if err = plugin.Start(app.pluginsContext); err != nil {
			log.Printf("Plugin %s start fail, %s\n", reflect.TypeOf(plugin).Elem().Name(), err)
			break
		}
	}
	if err != nil {
		return
	}

	return
}
