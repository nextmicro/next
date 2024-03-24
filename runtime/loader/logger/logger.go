package logger

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	kconfig "github.com/go-kratos/kratos/v2/config"
	log "github.com/nextmicro/logger"
	"github.com/nextmicro/next/adapter/logger/kratos"
	"github.com/nextmicro/next/adapter/logger/nacos"
	config "github.com/nextmicro/next/api/config/v1"
	conf "github.com/nextmicro/next/config"
	"github.com/nextmicro/next/pkg/env"
	"github.com/nextmicro/next/runtime/loader"
)

const (
	loggerPath = "/data/logs/%s/projlogs"
)

type logger struct {
	loader.BaseLoader
	opt loader.Options
	cfg *config.Logger
}

func New(opts ...loader.Option) loader.Loader {
	options := loader.NewOptions(opts...)

	return &logger{
		opt: *options,
	}
}

// Initialized returns the initialized status of the loader.
func (loader *logger) Initialized() bool {
	return loader.opt.Initialized
}

// Init is a loader initializer.
func (loader *logger) Init(opts ...loader.Option) error {
	cfg := conf.ApplicationConfig()
	logCfg := cfg.GetLogger()
	if logCfg == nil {
		logCfg = &config.Logger{
			Level:   "info",
			Console: true,
			File:    false,
		}
	}

	metadata := map[string]string{
		"app_id":      conf.ApplicationConfig().GetId(),
		"app_name":    conf.ApplicationConfig().GetName(),
		"app_version": conf.ApplicationConfig().GetVersion(),
		"env":         env.DeployEnvironment(),
		"instance_id": env.Hostname(),
	}
	cfg.Metadata = mergeMap(metadata, cfg.GetMetadata())

	if logCfg.Path == "" && env.DeployEnvironment() == env.Dev {
		logCfg.Path = filepath.Join(env.WorkDir(), "runtime", "logs")
	} else if logCfg.Path == "" {
		logCfg.Path = fmt.Sprintf(loggerPath, conf.ApplicationConfig().GetName())
	}

	log.DefaultLogger = log.New(options(logCfg)...) // adapter logger
	kratos.New(log.DefaultLogger).SetLogger()       // adapter kratos logger
	nacos.NewNacos(log.DefaultLogger).SetLogger()   // adapter nacos logger

	loader.cfg = logCfg
	loader.opt.Initialized = true
	log.Infof("Loader [%s] init success", loader.String())

	return nil
}

// 两个 map 合并
func mergeMap(m1, m2 map[string]string) map[string]string {
	for k, v := range m2 {
		m1[k] = v
	}
	return m1
}

func options(c *config.Logger) []log.Option {
	var opts []log.Option
	if c.FileName != "" {
		opts = append(opts, log.WithFilename(c.FileName))
	}
	if c.Path != "" {
		opts = append(opts, log.WithPath(c.Path))
	}
	if c.Level != "" {
		opts = append(opts, log.WithLevel(log.ParseLevel(c.Level)))
	}
	if c.File {
		opts = append(opts, log.WithMode(log.FileMode))
	}
	if c.MaxSize > 0 {
		opts = append(opts, log.WithMaxSize(int(c.MaxSize)))
	}
	if c.MaxBackups > 0 {
		opts = append(opts, log.WithMaxBackups(int(c.MaxBackups)))
	}
	if c.Compress {
		opts = append(opts, log.WithCompress(c.Compress))
	}
	if c.KeepHours > 0 {
		opts = append(opts, log.WithKeepHours(int(c.KeepHours)))
	}
	if c.KeepDays > 0 {
		opts = append(opts, log.WithKeepDays(int(c.KeepDays)))
	}
	if c.Rotation != "" {
		opts = append(opts, log.WithRotation(c.Rotation))
	}
	md := make(map[string]interface{})
	for k, v := range c.Metadata {
		md[k] = v
	}
	if len(md) > 0 {
		opts = append(opts, log.Fields(md))
	}

	return opts
}

func (loader *logger) Watch() error {
	err := conf.Watch("logger", func(key string, value kconfig.Value) {
		var cfg *config.Logger
		log.Info("logger config changed")

		err := value.Scan(&cfg)
		if err != nil {
			log.Errorf("logger watcher scan error: %s", err)
			return
		}

		log.DefaultLogger.SetLevel(log.ParseLevel(cfg.Level))
		log.Infof("logger config change, successfully loaded, old: %+v, new: %+v", loader.cfg, cfg)
		loader.cfg = cfg
	})
	if err != nil && !errors.Is(err, kconfig.ErrNotFound) {
		return err
	}

	log.Infof("Loader [%s] watch success", loader.String())
	return nil
}

func (loader *logger) Stop(ctx context.Context) error {
	log.Infof("Loader [%s] stop success", loader.String())
	_ = log.DefaultLogger.Sync()
	return nil
}

func (loader *logger) String() string {
	return "Logger"
}
