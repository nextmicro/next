package config

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"

	kConfig "github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/env"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"github.com/nextmicro/logger"
	"github.com/nextmicro/next/adapter/config/nacos"
	"github.com/nextmicro/next/adapter/logger/kratos"
	adapterNacos "github.com/nextmicro/next/adapter/logger/nacos"
	"github.com/nextmicro/next/api/config/v1"
	util "github.com/nextmicro/next/internal/pkg/file"
	kUtil "github.com/nextmicro/next/pkg/env"
)

const (
	_baseConf = "config.yaml"
)

var (
	_          kConfig.Config = &Config{}
	nextConfig *v1.Next
)

func init() {
	nextConfig = &v1.Next{}
}

type Config struct {
	kConfig.Config

	path     string
	filename string
}

func (c *Config) buildFileSource() []kConfig.Source {
	source := make([]kConfig.Source, 0, 3)

	// env source
	source = append(source, env.NewSource(kUtil.NextEnvPrefix))

	// base config file source
	baseFilename := filepath.Join(c.path, _baseConf)
	if exists, _ := util.Exists(baseFilename); exists {
		source = append(source, file.NewSource(baseFilename))
	}

	// custom config file source
	if exists, _ := util.Exists(c.filename); exists {
		source = append(source, file.NewSource(c.filename))
	}

	return source
}

// buildNacosSource 构建nacos配置源
func (c *Config) buildNacosSource() ([]kConfig.Source, error) {
	cfg := ApplicationConfig().GetNacos()
	if cfg == nil || len(cfg.Address) == 0 {
		return []kConfig.Source{}, nil
	}

	if cfg.GetCacheDir() == "" && kUtil.IsDev() {
		cfg.CacheDir = fmt.Sprintf("%s/runtime/nacos/cache", kUtil.WorkDir())
	} else if cfg.GetCacheDir() == "" {
		cfg.CacheDir = fmt.Sprintf("/data/nacos/%s/cache", cfg.DataId)
	}
	if cfg.GetLogDir() == "" && kUtil.IsDev() {
		cfg.LogDir = fmt.Sprintf("%s/runtime/nacos/log", kUtil.WorkDir())
	} else if cfg.GetLogDir() == "" {
		cfg.CacheDir = fmt.Sprintf("/data/nacos/%s/log", cfg.DataId)
	}

	if cfg.CacheDir != "" {
		// 判断备份目录是否存在
		exists, err := util.Exists(cfg.CacheDir)
		if err != nil {
			return nil, fmt.Errorf("failed to check backup path: %s, error: %s", cfg.CacheDir, err)
		}
		if !exists {
			if err := os.MkdirAll(cfg.CacheDir, 0755); err != nil {
				return nil, fmt.Errorf("failed to create backup path: %s, error: %s", cfg.CacheDir, err)
			}
		}
	}
	if cfg.LogDir != "" {
		// 判断备份目录是否存在
		exists, err := util.Exists(cfg.LogDir)
		if err != nil {
			return nil, fmt.Errorf("failed to check log dir path: %s, error: %s", cfg.LogDir, err)
		}
		if !exists {
			if err := os.MkdirAll(cfg.LogDir, 0755); err != nil {
				return nil, fmt.Errorf("failed to create log dir path: %s, error: %s", cfg.CacheDir, err)
			}
		}
	}

	serverConfigs := make([]constant.ServerConfig, 0)
	for _, addr := range cfg.Address {
		// check we have a port
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}

		p, err := strconv.ParseUint(port, 10, 64)
		if err != nil {
			return nil, err
		}
		serverConfigs = append(serverConfigs, *constant.NewServerConfig(host, p, constant.WithContextPath(cfg.ContextPath)))
	}

	var duration uint64 = 5000
	if cfg.GetTimeout().AsDuration() > 0 {
		duration = uint64(cfg.GetTimeout().AsDuration().Milliseconds())
	}
	if cfg.GetLogLevel() == "" && kUtil.IsDev() {
		cfg.LogLevel = "debug"
	} else if cfg.GetLogLevel() == "" {
		cfg.LogLevel = "info"
	}
	if cfg.Format == "" {
		cfg.Format = "yaml"
	}

	clientConfig := constant.NewClientConfig(
		constant.WithUsername(cfg.GetUsername()),
		constant.WithPassword(cfg.GetPassword()),
		constant.WithTimeoutMs(duration),
		constant.WithCacheDir(cfg.GetCacheDir()),
		constant.WithNamespaceId(cfg.GetNamespace()),
		constant.WithNotLoadCacheAtStart(cfg.NotLoadCacheAtStart),
		constant.WithLogLevel(cfg.LogLevel),
		constant.WithLogDir(cfg.LogDir),
		constant.WithOpenKMS(false),
	)
	client, err := clients.NewConfigClient(
		vo.NacosClientParam{
			ClientConfig:  clientConfig,
			ServerConfigs: serverConfigs,
		},
	)

	adapterNacos.NewNacos(logger.DefaultLogger).SetLogger() // adapter nacos logger
	if err != nil {
		return nil, fmt.Errorf("failed to create nacos client, error: %s", err)
	}

	return []kConfig.Source{
		nacos.NewConfigSource(client, nacos.WithDataID(cfg.DataId), nacos.WithGroup(cfg.Group), nacos.WithFormat(cfg.Format)),
	}, nil
}

// Init 初始化配置
func Init(filename string) (kConfig.Config, error) {
	cc := &Config{
		path:     filepath.Dir(filename),
		filename: filename,
	}

	kratos.New(logger.DefaultLogger).SetLogger() // adapter kratos logger

	// build file source
	source := cc.buildFileSource()

	cc.Config = kConfig.New(
		kConfig.WithSource(source...),
	)

	err := cc.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config filename: %s, error: %s", filename, err)
	}

	DefaultConfig = cc
	if err = AppScan(); err != nil {
		return nil, fmt.Errorf("failed to scan next config, error: %v", err)
	}

	// build Nacos source if needed
	sources, err := cc.buildNacosSource()
	if err != nil {
		return nil, err
	}

	if len(sources) > 0 {
		if err = cc.Close(); err != nil {
			return nil, err
		}
		source = append(source, sources...)
		cc.Config = kConfig.New(kConfig.WithSource(source...))
		err = cc.Load()
		if err != nil {
			return nil, fmt.Errorf("failed to load config filename: %s, error: %s", filename, err)
		}
	}

	DefaultConfig = cc
	return cc, nil
}

// AppScan 框架默认配置
func AppScan() error {
	out := &v1.Next{}
	DefaultConfig.Value("")
	if err := DefaultConfig.Scan(out); err != nil {
		return fmt.Errorf("failed to scan config: %s", err)
	}

	nextConfig = out
	return nil
}

// ApplicationConfig 获取框架配置
func ApplicationConfig() *v1.Next {
	return nextConfig
}

func (c *Config) Close() error {
	return c.Config.Close()
}
