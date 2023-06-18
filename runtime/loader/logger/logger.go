package logger

import (
	"context"
	"errors"

	kconfig "github.com/go-kratos/kratos/v2/config"
	klog "github.com/go-kratos/kratos/v2/log"
	log "github.com/go-volo/logger"
	config "github.com/nextmicro/next/api/config"
	conf "github.com/nextmicro/next/config"
	"github.com/nextmicro/next/logger/kratos"
	"github.com/nextmicro/next/runtime/loader"
)

type logger struct {
	opt loader.Options
	cfg *config.Logger
}

func New(opts ...loader.Option) loader.Loader {
	options := loader.NewOptions(opts...)

	return &logger{
		opt: *options,
	}
}

// Init is a loader initializer.
func (loader *logger) Init(opts ...loader.Option) error {
	for _, opt := range opts {
		opt(&loader.opt)
	}

	logOpts := make([]log.Option, 0)
	cfg, ok := loader.opt.Context.Value(loggerKey{}).(*config.Logger)
	if cfg != nil && ok {
		loader.cfg = cfg
		logOpts = options(cfg)
	}

	log.DefaultLogger = log.New(logOpts...)            // 重写了log.DefaultLogger
	klog.DefaultLogger = kratos.New(log.DefaultLogger) // 重写了klog.DefaultLogger

	log.Infof("Loader [%s] init success", loader.String())

	return nil
}

func (loader *logger) Start(ctx context.Context) error {
	return nil
}

func (loader *logger) Watch() error {
	err := conf.Watch(loader.String(), func(key string, value kconfig.Value) {
		var cfg *config.Logger
		log.Info("logger config changed")

		err := value.Scan(&cfg)
		if err != nil {
			log.Errorf("logger watcher scan error: %s", err)
			return
		}

		log.DefaultLogger.SetLevel(log.ParseLevel(cfg.Level))
		log.Infof("logger config change, successfully loaded, old: %+v, new: %+v", loader.cfg, cfg)
	})
	if err != nil && !errors.Is(err, kconfig.ErrNotFound) {
		return err
	}

	log.Infof("Loader [%s] watch success", loader.String())
	return nil
}

func (loader *logger) Stop(ctx context.Context) error {
	_ = log.DefaultLogger.Sync()
	log.Infof("Loader [%s] stop success", loader.String())
	return nil
}

func (loader *logger) String() string {
	return "logger"
}
