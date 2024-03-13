package nacos

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"

	"github.com/go-kratos/kratos/v2/config"
)

type Option func(*options)

type options struct {
	group  string
	dataID string
	format string
}

// WithGroup With nacos config group.
func WithGroup(group string) Option {
	return func(o *options) {
		o.group = group
	}
}

// WithDataID With nacos config data id.
func WithDataID(dataID string) Option {
	return func(o *options) {
		o.dataID = dataID
	}
}

// WithFormat with nacos config format.
func WithFormat(format string) Option {
	return func(o *options) {
		o.format = format
	}
}

type Config struct {
	opts   options
	client config_client.IConfigClient
}

func NewConfigSource(client config_client.IConfigClient, opts ...Option) config.Source {
	_options := options{}
	for _, o := range opts {
		o(&_options)
	}
	return &Config{client: client, opts: _options}
}

func (c *Config) Load() ([]*config.KeyValue, error) {
	content, err := c.client.GetConfig(vo.ConfigParam{
		DataId: c.opts.dataID,
		Group:  c.opts.group,
	})
	if err != nil {
		return nil, err
	}
	k := c.opts.dataID
	format := strings.TrimPrefix(filepath.Ext(k), ".")
	if format == "" && c.opts.format != "" {
		format = c.opts.format
	}
	return []*config.KeyValue{
		{
			Key:    k,
			Value:  []byte(content),
			Format: format,
		},
	}, nil
}

func (c *Config) Watch() (config.Watcher, error) {
	watcher := newWatcher(context.Background(), c.opts.dataID, c.opts.group, c.opts.format, c.client.CancelListenConfig)
	err := c.client.ListenConfig(vo.ConfigParam{
		DataId: c.opts.dataID,
		Group:  c.opts.group,
		OnChange: func(_, group, dataId, data string) {
			if dataId == watcher.dataID && group == watcher.group {
				watcher.content <- data
			}
		},
	})
	if err != nil {
		return nil, err
	}
	return watcher, nil
}
