package configs

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

type GlobalConfig struct {
	AppConfig AppConfig `yaml:"app"`
	DbConfig  DbConfig  `yaml:"db"`
	LogConfig LogConfig `yaml:"log"`
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
	CreateDB    bool   `yaml:"create_db"`
	AutoMigrate bool   `yaml:"auto_migrate"`
	AutoSeed    bool   `yaml:"auto_seed"`
	MaxIdleConn int    `yaml:"max_idle_conn"`
	MaxOpenConn int    `yaml:"max_open_conn"`
	MaxIdleTime int    `yaml:"max_idle_time"`
}

type LogConfig struct {
	Level         string `yaml:"level"`
	Format        string `yaml:"format"`
	Directory     string `yaml:"directory"`
	AppFile       string `yaml:"app_file"`
	AccessFile    string `yaml:"access_file"`
	AlsoStdout    bool   `yaml:"also_stdout"`
	SQLLevel      string `yaml:"sql_level"`
	SQLSlowMS     int    `yaml:"sql_slow_ms"`
	IncludeSource bool   `yaml:"include_source"`
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
			CreateDB:    true,
			AutoMigrate: true,
			AutoSeed:    true,
			MaxIdleConn: 10,
			MaxOpenConn: 25,
			MaxIdleTime: 1800,
		},
		LogConfig: LogConfig{
			Level:         "info",
			Format:        "json",
			Directory:     "logs",
			AppFile:       "app.log",
			AccessFile:    "access.log",
			AlsoStdout:    true,
			SQLLevel:      "warn",
			SQLSlowMS:     500,
			IncludeSource: false,
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
	if value, ok := envBool("MYSQL_CREATE_DATABASE"); ok {
		cfg.DbConfig.CreateDB = value
	}
	if value, ok := envBool("MYSQL_AUTO_MIGRATE"); ok {
		cfg.DbConfig.AutoMigrate = value
	}
	if value, ok := envBool("MYSQL_AUTO_SEED"); ok {
		cfg.DbConfig.AutoSeed = value
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

	if value := os.Getenv("LOG_LEVEL"); value != "" {
		cfg.LogConfig.Level = strings.ToLower(value)
	}
	if value := os.Getenv("LOG_FORMAT"); value != "" {
		cfg.LogConfig.Format = strings.ToLower(value)
	}
	if value := os.Getenv("LOG_DIR"); value != "" {
		cfg.LogConfig.Directory = value
	}
	if value := os.Getenv("LOG_APP_FILE"); value != "" {
		cfg.LogConfig.AppFile = value
	}
	if value := os.Getenv("LOG_ACCESS_FILE"); value != "" {
		cfg.LogConfig.AccessFile = value
	}
	if value, ok := envBool("LOG_TO_STDOUT"); ok {
		cfg.LogConfig.AlsoStdout = value
	}
	if value := os.Getenv("LOG_SQL_LEVEL"); value != "" {
		cfg.LogConfig.SQLLevel = strings.ToLower(value)
	}
	if value := os.Getenv("LOG_SQL_SLOW_MS"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			cfg.LogConfig.SQLSlowMS = parsed
		}
	}
	if value, ok := envBool("LOG_INCLUDE_SOURCE"); ok {
		cfg.LogConfig.IncludeSource = value
	}
}

func hasEnv(key string) bool {
	_, ok := os.LookupEnv(key)
	return ok
}

func envBool(key string) (bool, bool) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return false, false
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, false
	}
	return parsed, true
}
