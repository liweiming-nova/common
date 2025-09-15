package parser

import (
	"bytes"
	"fmt"
	"github.com/liweiming-nova/common/config/options"

	"github.com/spf13/viper"
	"os"
	"path/filepath"
)

// ViperParser 支持 JSON/YAML/TOML 的通用配置解析器
type ViperParser struct {
	modTime int64
}

func NewViperParser() *ViperParser {
	return &ViperParser{}
}

// Unmarshal 解析多个配置源到目标结构体
func (p *ViperParser) Unmarshal(cfg interface{}, opts *options.Options) error {
	sources, err := p.parseSource(opts)
	if err != nil {
		return err
	}

	v := viper.New()

	// 合并所有配置文件
	for _, source := range sources {
		if err := p.mergeConfig(v, source); err != nil {
			return fmt.Errorf("failed to merge config %s: %w", source, err)
		}
	}

	// 反序列化到结构体
	if err := v.Unmarshal(cfg); err != nil {
		return fmt.Errorf("failed to unmarshal into struct: %w", err)
	}

	return nil
}

// GetLastModTime 返回所有配置文件中最新的修改时间
func (p *ViperParser) GetLastModTime(opts *options.Options) (int64, error) {
	sources, err := p.parseSource(opts)
	if err != nil {
		return 0, err
	}

	var latest int64
	for _, source := range sources {
		if !IsLocalFile(source) {
			continue
		}
		info, err := os.Stat(source)
		if err != nil {
			return 0, err
		}
		modTime := info.ModTime().Unix()
		if modTime > latest {
			latest = modTime
		}
	}
	p.modTime = latest
	return latest, nil
}

// parseSource 解析主文件及其 import 列表
func (p *ViperParser) parseSource(opts *options.Options) ([]string, error) {
	if len(opts.Sources) == 0 {
		return nil, fmt.Errorf("config sources not specified")
	}

	var sources []string
	sources = append(sources, opts.Sources...)

	mainFile := opts.Sources[0]
	if !IsLocalFile(mainFile) {
		return sources, nil
	}

	dir := filepath.Dir(mainFile)

	// 用临时 Viper 读取 import 字段（不关心格式）
	tempV := viper.New()
	tempV.SetConfigFile(mainFile)
	ext := filepath.Ext(mainFile)
	configType := p.getConfigType(ext)
	if configType != "" {
		tempV.SetConfigType(configType)
	}

	if err := tempV.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read main config: %w", err)
	}

	var imports []string
	if err := tempV.UnmarshalKey("import", &imports); err != nil {
		_ = err // 忽略，没有 import 字段也正常
	}

	for _, imp := range imports {
		// 尝试常见后缀
		for _, suffix := range []string{".toml", ".yaml", ".yml", ".json"} {
			importPath := filepath.Join(dir, imp+suffix)
			if IsLocalFile(importPath) {
				sources = append(sources, importPath)
				break
			}
		}
	}

	return sources, nil
}

// mergeConfig 将单个配置文件内容合并到 viper 实例
func (p *ViperParser) mergeConfig(v *viper.Viper, source string) error {
	if !IsLocalFile(source) {
		return fmt.Errorf("unsupported remote source: %s", source)
	}

	data, err := os.ReadFile(source)
	if err != nil {
		return fmt.Errorf("read failed: %s: %w", source, err)
	}

	ext := filepath.Ext(source)
	configType := p.getConfigType(ext)
	if configType == "" {
		return fmt.Errorf("unsupported config type: %s", ext)
	}

	if err := v.MergeConfig(bytes.NewReader(data)); err != nil {
		return fmt.Errorf("merge failed: %w", err)
	}

	return nil
}

// getConfigType 根据文件扩展名返回 viper 配置类型
func (p *ViperParser) getConfigType(ext string) string {
	switch ext {
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	case ".toml":
		return "toml"
	default:
		return ""
	}
}
