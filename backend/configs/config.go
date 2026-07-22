package configs

import (
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"gopkg.in/yaml.v3"
)

type GlobalConfig struct {
	AppConfig AppConfig `yaml:"app"`
	DbConfig  DbConfig  `yaml:"db"`
}

type AppConfig struct {
	AppName string `yaml:"app_name"`
	Version string `yaml:"version"`
	Port    int    `yaml:"port"`
	RunMode string `yaml:"run_mode"`
}

type DbConfig struct {
	Host        string `yaml:"host"`
	Port        string `yaml:"port"`
	User        string `yaml:"user"`
	Password    string `yaml:"password"`
	DBName      string `yaml:"dbname"`
	MaxIdleConn int    `yaml:"max_idle_conn"`
	MaxOpenConn int    `yaml:"max_open_conn"`
	MaxIdleTime int    `yaml:"max_idle_time"`
}

var (
	globalConfig GlobalConfig
	once         sync.Once
)

func InitGlobalConfig() {
	once.Do(func() {
		cfg := defaultConfig()
		loadConfigFile(&cfg)
		overrideFromEnv(&cfg)
		globalConfig = cfg
	})
}

func GetGlobalConfig() GlobalConfig {
	InitGlobalConfig()
	return globalConfig
}

func defaultConfig() GlobalConfig {
	return GlobalConfig{
		AppConfig: AppConfig{
			AppName: "couple-mini-backend",
			Version: "v0.0.1",
			Port:    8080,
			RunMode: "release",
		},
		DbConfig: DbConfig{
			Host:        "127.0.0.1",
			Port:        "3306",
			User:        "root",
			Password:    "password",
			DBName:      "couple_mini",
			MaxIdleConn: 10,
			MaxOpenConn: 25,
			MaxIdleTime: 1800,
		},
	}
}

func loadConfigFile(cfg *GlobalConfig) {
	path := filepath.Join("configs", "config.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	_ = yaml.Unmarshal(data, cfg)
}

func overrideFromEnv(cfg *GlobalConfig) {
	if value := os.Getenv("APP_NAME"); value != "" {
		cfg.AppConfig.AppName = value
	}
	if value := os.Getenv("APP_VERSION"); value != "" {
		cfg.AppConfig.Version = value
	}
	if value := os.Getenv("APP_RUN_MODE"); value != "" {
		cfg.AppConfig.RunMode = value
	}
	if value := os.Getenv("PORT"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			cfg.AppConfig.Port = parsed
		}
	}

	if value := os.Getenv("MYSQL_HOST"); value != "" {
		cfg.DbConfig.Host = value
	}
	if value := os.Getenv("MYSQL_PORT"); value != "" {
		cfg.DbConfig.Port = value
	}
	if value := os.Getenv("MYSQL_USER"); value != "" {
		cfg.DbConfig.User = value
	}
	if value := os.Getenv("MYSQL_PASSWORD"); value != "" || hasEnv("MYSQL_PASSWORD") {
		cfg.DbConfig.Password = value
	}
	if value := os.Getenv("MYSQL_DATABASE"); value != "" {
		cfg.DbConfig.DBName = value
	}
	if value := os.Getenv("MYSQL_MAX_IDLE_CONN"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			cfg.DbConfig.MaxIdleConn = parsed
		}
	}
	if value := os.Getenv("MYSQL_MAX_OPEN_CONN"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			cfg.DbConfig.MaxOpenConn = parsed
		}
	}
	if value := os.Getenv("MYSQL_MAX_IDLE_TIME"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			cfg.DbConfig.MaxIdleTime = parsed
		}
	}
}

func hasEnv(key string) bool {
	_, ok := os.LookupEnv(key)
	return ok
}
