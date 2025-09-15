/* ######################################################################
# Author: (zfly1207@126.com)
# Created Time: 2020-01-05 12:49:07
# File Name: parser.go
# Description:
####################################################################### */

package parser

import (
	"fmt"
	"github.com/liweiming-nova/common/config/options"

	"os"
)

type Parser interface {
	Unmarshal(cfg interface{}, opts *options.Options) error
	GetLastModTime(opts *options.Options) (int64, error)
}

func ParseFileLastModTime(file string) (r int64, err error) {
	fd, err := os.Stat(file)
	if err != nil {
		err = fmt.Errorf("parse file last modified time fail, %s", err)
		return
	}
	r = fd.ModTime().Unix()
	return
}

func IsLocalFile(file string) bool {
	//return strings.HasPrefix(file, ".") || strings.HasPrefix(file, "/")
	_, err := os.Stat(file)
	return os.IsNotExist(err) == false
}

type TomlImport struct {
	Import []string
}
