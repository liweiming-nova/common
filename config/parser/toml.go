package parser

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/liweiming-nova/common/config/options"

	"path"
)

type TomlParser struct {
	modTime int64
}

func NewTomlParser() *TomlParser {
	o := &TomlParser{}
	return o
}

func (parser *TomlParser) Unmarshal(cfg interface{}, opts *options.Options) (err error) {
	var sources []string
	if sources, err = parser.parseSource(opts); err != nil {
		return
	}

	for _, source := range sources {
		if err = parser.decode(cfg, source); err != nil {
			return
		}
	}
	return
}

func (parser *TomlParser) GetLastModTime(opts *options.Options) (r int64, err error) {
	var sources []string
	if sources, err = parser.parseSource(opts); err != nil {
		return
	}

	for _, source := range sources {
		if IsLocalFile(source) == false {
			continue
		}
		var modTime int64
		if modTime, err = ParseFileLastModTime(source); err != nil {
			return
		}
		if modTime > parser.modTime {
			parser.modTime = modTime
		}
	}
	return parser.modTime, nil
}

func (parser *TomlParser) parseSource(opts *options.Options) (r []string, err error) {
	r = []string{}
	r = append(r, opts.Sources...)

	t := &TomlImport{}
	dir, _ := path.Split(opts.Sources[0])

	if err = parser.decode(t, opts.Sources[0]); err != nil {
		return
	}

	for _, v := range t.Import {
		r = append(r, fmt.Sprintf("%s%s.toml", dir, v))
	}
	return
}

func (parser *TomlParser) decode(cfg interface{}, source string) (err error) {
	if len(source) == 0 {
		err = fmt.Errorf("config source not specified")
		return
	}

	if IsLocalFile(source) == true {
		if _, err = toml.DecodeFile(source, cfg); err != nil {
			err = fmt.Errorf("local config source decode fail, %s", err)
		}
		return
	}

	err = fmt.Errorf("local config source[%s] not found", source)
	return
}
