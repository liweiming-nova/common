/* ######################################################################
# Author: (zfly1207@126.com)
# Created Time: 2020-10-30 22:01:02
# File Name: server_mgr.go
# Description:
####################################################################### */

package server

import (
	"context"
	"fmt"
	"github.com/liweiming-nova/common/config"
	"github.com/liweiming-nova/common/utils"
	"net/http"
	"sync"
	"time"
)

var (
	once    sync.Once
	lock    sync.RWMutex
	servers map[string]*http.Server
)

func init() {
	servers = map[string]*http.Server{}
}

type restConfig struct {
	Rest *struct {
		Cfgs map[string]*Cfg `toml:"server"`
	} `toml:"rest"`
}

type Cfg struct {
	// dial
	DialAddr         string        `toml:"addr"`
	DialReadTimeout  time.Duration `toml:"read_timeout"`
	DialWriteTimeout time.Duration `toml:"write_timeout"`
	DialIdleTimeout  time.Duration `toml:"idle_timeout"`
}

func StartDefaultServer(rcvr http.Handler) (err error) {
	return StartServe("default", rcvr)
}

func StopDefaultServer() (err error) {
	return StopServe("default")
}

func DefaultServer() (r *http.Server) {
	return Server("default")
}

func StartServe(name string, rcvr http.Handler) (err error) {
	utils.Lock(1)
	var srv *http.Server
	if srv, err = SafeServer(name); err == nil {
		srv.Handler = rcvr
	}
	return
}

func StopServe(name string) (err error) {
	defer utils.Unlock()
	var srv *http.Server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if srv, err = SafeServer(name); err == nil {
		err = srv.Shutdown(ctx)
	}
	return
}

func Server(name string) (r *http.Server) {
	var err error
	if r, err = getServer(name); err != nil {
		panic(err)
	}
	return
}

func SafeServer(name string) (r *http.Server, err error) {
	return getServer(name)
}

func getServer(name string) (r *http.Server, err error) {
	lock.RLock()
	r = servers[name]
	lock.RUnlock()
	if r == nil {
		r, err = addServer(name)
	}
	return
}

func addServer(name string) (r *http.Server, err error) {
	var cfg *Cfg
	if cfg, err = loadCfg(name); err != nil {
		return
	}
	if r, err = NewRestServer(cfg); err != nil {
		return
	}

	lock.Lock()
	servers[name] = r
	lock.Unlock()
	return
}

func loadCfg(name string) (r *Cfg, err error) {
	var cfgs map[string]*Cfg
	if cfgs, err = loadCfgs(); err != nil {
		return
	}
	if r = cfgs[name]; r == nil {
		err = fmt.Errorf("rest#%s not configed", name)
		return
	}
	return
}

func loadCfgs() (r map[string]*Cfg, err error) {
	r = map[string]*Cfg{}

	cfg := config.Get(&restConfig{}).(*restConfig)
	if err == nil && (cfg.Rest == nil || cfg.Rest.Cfgs == nil || len(cfg.Rest.Cfgs) == 0) {
		err = fmt.Errorf("not configed")
	}
	if err != nil {
		err = fmt.Errorf("rest load cfgs error, %s", err)
		return
	}
	r = cfg.Rest.Cfgs
	return
}
