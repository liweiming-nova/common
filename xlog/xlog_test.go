package xlog

import (
	"context"
	"testing"
)

func TestXlog(t *testing.T) {
	Infof(context.Background(), "hello world")
}
