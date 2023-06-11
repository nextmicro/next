package config

import (
	"context"
	"errors"

	kconfig "github.com/go-kratos/kratos/v2/config"
	"github.com/go-volo/logger"
	conf "github.com/nextmicro/next/config"
	"github.com/nextmicro/next/runtime/loader"
)

type config struct {
	opt         loader.Options
	ctx         context.Context
	watchCancel context.CancelFunc
}

func New(opts ...loader.Option) loader.Loader {
	o := loader.Options{}
	for _, opt := range opts {
		opt(&o)
	}

	watchCtx, cancel := context.WithCancel(context.Background())

	return &config{
		opt:         o,
		ctx:         watchCtx,
		watchCancel: cancel,
	}
}

func (loader config) Init(opts ...loader.Option) error {
	for _, opt := range opts {
		opt(&loader.opt)
	}

	filePath, ok := loader.opt.Context.Value(filePathKey{}).(string)
	if !ok || filePath == "" {
		return errors.New("config: file_path not empty")
	}

	c, err := conf.New(filePath)
	if err != nil {
		return err
	}
	conf.DefaultConfig = c

	logger.Infof("Loader [%s] init success", loader.String())

	return nil
}

func (loader config) Start(ctx context.Context) error {
	return nil
}

func (loader config) Watch() error {
	err := conf.Watch("config", func(key string, value kconfig.Value) {
		if err := conf.AppScan(value); err != nil {
			logger.Errorf("config watcher scan error: %s", err)
		}

		logger.Infof("config change, successfully loaded, new: %+v", conf.AppConfig())
	})
	if err != nil {
		return errors.New("config: watch error")
	}

	logger.Infof("Loader [%s] watch success", loader.String())

	return nil
}

func (loader config) Stop(ctx context.Context) error {
	loader.watchCancel()

	err := conf.Close()
	if err != nil {
		return errors.New("config: close error")
	}

	logger.Infof("Loader [%s] stop success", loader.String())

	return nil
}

func (loader config) String() string {
	return "config"
}
