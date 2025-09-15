package options

type Options struct {
	Pwd            string
	Sources        []string          // config source
	MemoryVariable interface{}       // memory reference data
	CheckInterval  int64             // file update check interval
	OnChangeFn     func(interface{}) // call it when the file is modified
	OnErrorFn      func(error)       // call it when an error occurs
}

type Option func(o *Options)

type OpOption func(o *Options)

func WithOpOnErrorFn(inp func(error)) Option {
	return func(o *Options) {
		o.OnErrorFn = inp
	}
}

// WithPwd default, pwd: os.Getwd()
func WithPwd(pwd string) Option {
	return func(o *Options) {
		o.Pwd = pwd
	}
}

func WithOpOnChangeFn(inp func(cfg interface{})) Option {
	return func(o *Options) {
		o.OnChangeFn = inp
	}
}

func WithCfgSource(inp ...string) Option {
	return func(o *Options) {
		o.Sources = inp
	}
}

// WithCheckInterval 用户希望 每隔多少秒检查一次配置文件是否被修改。
func WithCheckInterval(seconds int64) Option {
	return func(o *Options) {
		o.CheckInterval = seconds
	}
}
