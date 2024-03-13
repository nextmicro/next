package nacos

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/nacos-group/nacos-sdk-go/v2/vo"

	"github.com/go-kratos/kratos/v2/config"
)

type Watcher struct {
	dataID             string
	group              string
	format             string
	content            chan string
	cancelListenConfig cancelListenConfigFunc

	ctx    context.Context
	cancel context.CancelFunc
}

type cancelListenConfigFunc func(params vo.ConfigParam) (err error)

func newWatcher(ctx context.Context, dataID string, group string, format string, cancelListenConfig cancelListenConfigFunc) *Watcher {
	ctx, cancel := context.WithCancel(ctx)
	w := &Watcher{
		dataID:             dataID,
		group:              group,
		format:             format,
		cancelListenConfig: cancelListenConfig,
		content:            make(chan string, 100),

		ctx:    ctx,
		cancel: cancel,
	}
	return w
}

func (w *Watcher) Next() ([]*config.KeyValue, error) {
	format := strings.TrimPrefix(filepath.Ext(w.dataID), ".")
	if format == "" && w.format != "" {
		format = w.format
	}
	select {
	case <-w.ctx.Done():
		return nil, w.ctx.Err()
	case content := <-w.content:
		k := w.dataID
		return []*config.KeyValue{
			{
				Key:    k,
				Value:  []byte(content),
				Format: format,
			},
		}, nil
	}
}

func (w *Watcher) Close() error {
	err := w.cancelListenConfig(vo.ConfigParam{
		DataId: w.dataID,
		Group:  w.group,
	})
	w.cancel()
	return err
}

func (w *Watcher) Stop() error {
	return w.Close()
}
