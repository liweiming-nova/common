package plugins

import (
	"database/sql"
	"fmt"
	"github.com/liweiming-nova/common/config"
	"github.com/liweiming-nova/common/config/options"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"strings"
	"sync"
	"time"
)

type Cfg struct {
	// dial
	Engine   string `toml:"engine"` // postgressql/mysql
	DialUser string `toml:"user"`
	DialPawd string `toml:"pawd"`
	DialHost string `toml:"host"`
	DialPort string `toml:"port"`
	DialName string `toml:"name"`

	Debug bool `toml:"debug"` // 是否显示sql语句

	// pool
	PoolMaxOpenConn     int           `toml:"max_open_conn"`     // 最大连接数大小
	PoolMaxIdleConn     int           `toml:"max_idle_conn"`     // 最大空闲的连接的个数
	PoolConnMaxLifetime time.Duration `toml:"conn_max_lifetime"` // 连接的生命时间,超过此时间，连接将关闭后重新建立新的，0代表忽略相关判断,单位:second
}

var (
	once  sync.Once
	lock  sync.RWMutex
	pools map[string]*gorm.DB
)

type sqlConfig struct {
	Cfgs map[string]*Cfg `toml:"sql"`
}

func init() {
	pools = map[string]*gorm.DB{}
}

// Valid 参数names是实例的名称列表，如果为空则检测所有配置的实例
func Valid(names ...string) (err error) {
	if len(names) == 0 {
		var cfgs map[string]*Cfg
		if cfgs, err = loadCfgs(); err != nil {
			return
		}
		for k, _ := range cfgs {
			names = append(names, k)
		}
	}
	for _, name := range names {
		var cli *sql.DB
		cli, err = Client(name).DB()
		if err == nil {
			err = cli.Ping()
		}
		if err != nil {
			err = fmt.Errorf("mysql#%s is invalid, %s", name, err)
			return
		}
	}
	return
}

func Client(name string) (r *gorm.DB) {
	return Pool(name)
}

func Pool(name string) (r *gorm.DB) {
	var err error
	if r, err = getPool(name); err != nil {
		panic(err)
	}
	return
}

func getPool(name string) (r *gorm.DB, err error) {
	lock.RLock()
	r = pools[name]
	lock.RUnlock()
	if r == nil {
		r, err = addPool(name)
	}
	return
}

func addPool(name string) (r *gorm.DB, err error) {
	var cfg *Cfg
	if cfg, err = loadCfg(name); err != nil {
		return
	}
	r = NewSqlPool(cfg)

	lock.Lock()
	pools[name] = r
	lock.Unlock()
	return
}

func loadCfg(name string) (r *Cfg, err error) {
	var cfgs map[string]*Cfg
	if cfgs, err = loadCfgs(); err != nil {
		return
	}
	if r = cfgs[name]; r == nil {
		err = fmt.Errorf("mysql#%s not configed", name)
		return
	}
	return
}

func loadCfgs() (r map[string]*Cfg, err error) {
	r = map[string]*Cfg{}

	once.Do(func() {
		config.Get(&sqlConfig{}, options.WithOpOnChangeFn(func(cfg interface{}) {
			lock.Lock()
			defer lock.Unlock()
			pools = map[string]*gorm.DB{}
		}))
	})

	cfg := config.Get(&sqlConfig{}).(*sqlConfig)

	if err == nil && (cfg == nil || cfg.Cfgs == nil || len(cfg.Cfgs) == 0) {
		err = fmt.Errorf("not configed")
	}
	if err != nil {
		err = fmt.Errorf("mysql load cfgs error, %s", err)
		return
	}
	r = cfg.Cfgs
	return
}

func NewSqlPool(cfg *Cfg) *gorm.DB {
	gcfg := &gorm.Config{}
	if cfg.Debug == true {
		gcfg.Logger = logger.Default.LogMode(logger.Info)
	}

	var dialector gorm.Dialector
	cfg.Engine = strings.ToUpper(cfg.Engine)
	if cfg.Engine == "POSTGRESQL" {
		dialector = postgres.Open(fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC",
			cfg.DialHost, cfg.DialUser, cfg.DialPawd, cfg.DialName, cfg.DialPort))
	}
	if cfg.Engine == "MYSQL" {
		dialector = mysql.Open(fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=UTC",
			cfg.DialUser, cfg.DialPawd, cfg.DialHost, cfg.DialPort, cfg.DialName))
	}
	orm, err := gorm.Open(dialector, gcfg)
	if err != nil {
		panic(fmt.Sprintf("mysql connect errr: %s", err))
	}

	db, err := orm.DB()
	if err != nil {
		panic(fmt.Sprintf("Failed to get DB instance: %s", err))
	}

	if cfg.PoolMaxOpenConn > 0 {
		db.SetMaxOpenConns(cfg.PoolMaxOpenConn)
	}
	if cfg.PoolMaxIdleConn > 0 {
		db.SetMaxIdleConns(cfg.PoolMaxIdleConn)
	}
	if cfg.PoolConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(cfg.PoolConnMaxLifetime * time.Millisecond)
	}
	return orm
}

type SqlPlugin struct {
	names []string
}

func NewSqlPlugin(names ...string) (r *SqlPlugin) {
	return &SqlPlugin{names: names}
}

func (plugin *SqlPlugin) Start(ctx *PluginContext) (err error) {
	if err = Valid(plugin.names...); err != nil {
		err = fmt.Errorf("Sql valid error: %s\n", err)
	}
	return
}

func (plugin *SqlPlugin) Stop() (err error) {
	return
}

func (plugin *SqlPlugin) BeforeStart(ctx *PluginContext) (err error) {
	return
}
