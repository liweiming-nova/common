package config

import (
	"encoding/json"
	"github.com/liweiming-nova/common/config/options"
	"github.com/liweiming-nova/common/config/parser"
	"github.com/liweiming-nova/common/utils"
	"reflect"
	"sync"
	"time"
)

var Instance *Config

type item struct {
	m          interface{}
	hash       string
	elem       reflect.Value
	onChangeFn func(interface{})
	onErrorFn  func(error)
}

type Config struct {
	lock   sync.RWMutex
	opts   *options.Options
	items  map[string]*item
	parser parser.Parser
}

func NewConfig(parser parser.Parser, opts ...options.Option) (r *Config) {
	options := &options.Options{
		OnChangeFn: func(cfg interface{}) {},
		OnErrorFn:  func(error) {}}
	for _, opt := range opts {
		opt(options)
	}

	Instance = &Config{
		opts:   options,
		items:  map[string]*item{},
		parser: parser}

	go Instance.changeChecker()

	return Instance
}

func (config *Config) changeChecker() {
	if config.opts.CheckInterval == 0 {
		return
	}

	var err error
	ticker := time.NewTicker(time.Second * time.Duration(config.opts.CheckInterval))
	for _ = range ticker.C {
		var lastModTime int64
		lastModTime, err = config.parser.GetLastModTime(config.opts)
		if err != nil {
			config.doError(err, "")
			continue
		}

		if time.Now().Unix()-lastModTime > config.opts.CheckInterval {
			continue
		}

		config.lock.Lock()
		for pointer, one := range config.items {
			if err := config.parser.Unmarshal(one.m, config.opts); err != nil {
				config.doError(err, pointer)
				continue
			}

			oldHash := one.hash

			b, _ := json.Marshal(one.m)
			config.items[pointer].m = one.m
			config.items[pointer].hash = utils.Md5String(string(b))
			config.items[pointer].elem = reflect.ValueOf(one.m).Elem()

			if oldHash != config.items[pointer].hash {
				config.opts.OnChangeFn(config.items[pointer].m)
				config.items[pointer].onChangeFn(config.items[pointer].m)
			}
		}
		config.lock.Unlock()
	}
	return
}

func (config *Config) doError(err error, pointer string) {
	if err == nil {
		return
	}
	config.opts.OnErrorFn(err)

	if _, ok := config.items[pointer]; ok {
		config.items[pointer].onErrorFn(err)
	}
}

func Get(cfg interface{}, opts ...options.Option) interface{} {
	Instance.lock.Lock()
	defer Instance.lock.Unlock()

	pointer := reflect.TypeOf(cfg).String()
	if _, ok := Instance.items[pointer]; !ok {
		options := &options.Options{
			OnChangeFn: func(cfg interface{}) {},
			OnErrorFn:  func(error) {}}
		for _, opt := range opts {
			opt(options)
		}

		Instance.items[pointer] = &item{
			onChangeFn: options.OnChangeFn,
			onErrorFn:  options.OnErrorFn}

		if err := Instance.parser.Unmarshal(cfg, Instance.opts); err != nil {
			Instance.doError(err, pointer)
			return nil
		}

		b, _ := json.Marshal(cfg)
		Instance.items[pointer].m = cfg
		Instance.items[pointer].hash = utils.Md5String(string(b))
		Instance.items[pointer].elem = reflect.ValueOf(cfg).Elem()
	}

	if v := Instance.items[pointer]; v != nil {
		reflect.ValueOf(cfg).Elem().Set(v.elem)
	}
	return cfg
}
