package client

import (
	"context"
	"fmt"
	"github.com/liweiming-nova/common/app"
	"github.com/liweiming-nova/common/app/plugins"
	"testing"
)

type AppConfig struct {
}

func TestCall(t *testing.T) {
	app.NewApp().Use(plugins.NewConfigPlugin("app.toml", &AppConfig{})).Start()
	err := Call(context.Background(), "user_server", "GetUser", nil, nil)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
}
