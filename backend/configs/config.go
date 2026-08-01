package configs

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

type GlobalConfig struct {
	AppConfig    AppConfig    `yaml:"app"`
	DbConfig     DbConfig     `yaml:"db"`
	LogConfig    LogConfig    `yaml:"log"`
	AdminConfig  AdminConfig  `yaml:"admin"`
	AuthConfig   AuthConfig   `yaml:"auth"`
	WeChatConfig WeChatConfig `yaml:"wechat"`
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
	ErrorFile     string `yaml:"error_file"`
	AlsoStdout    bool   `yaml:"also_stdout"`
	SQLLevel      string `yaml:"sql_level"`
	SQLSlowMS     int    `yaml:"sql_slow_ms"`
	IncludeSource bool   `yaml:"include_source"`
}

type AdminConfig struct {
	Enabled           bool   `yaml:"enabled"`
	Username          string `yaml:"username"`
	Password          string `yaml:"password"`
	Title             string `yaml:"title"`
	SampleIntervalSec int    `yaml:"sample_interval_sec"`
	HistoryLimit      int    `yaml:"history_limit"`
}

type AuthConfig struct {
	TokenSecret   string `yaml:"token_secret"`
	TokenTTLHours int    `yaml:"token_ttl_hours"`
}

type WeChatConfig struct {
	AppID                       string `yaml:"app_id"`
	Secret                      string `yaml:"secret"`
	SubscribeNoticeTemplateID   string `yaml:"subscribe_notice_template_id"`
	SubscribeScheduleTemplateID string `yaml:"subscribe_schedule_template_id"`
	SubscribeTitleKey           string `yaml:"subscribe_title_key"`
	SubscribeContentKey         string `yaml:"subscribe_content_key"`
	SubscribePage               string `yaml:"subscribe_page"`
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

func ValidateForStartup(cfg GlobalConfig) error {
	if strings.ToLower(strings.TrimSpace(cfg.AppConfig.RunMode)) != "release" {
		return nil
	}
	if cfg.AuthConfig.TokenSecret == "" || cfg.AuthConfig.TokenSecret == "change-me-token-secret" {
		return fmt.Errorf("AUTH_TOKEN_SECRET must be set to a non-default value in release mode")
	}
	if cfg.AdminConfig.Enabled && (cfg.AdminConfig.Password == "" || cfg.AdminConfig.Password == "admin123456") {
		return fmt.Errorf("ADMIN_PASSWORD must be set to a non-default value in release mode")
	}
	if !hasEnv("MYSQL_DSN") && cfg.DbConfig.Password == "password" {
		return fmt.Errorf("MYSQL_PASSWORD must be set to a non-default value in release mode")
	}
	return nil
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
			Directory:     "logs/backend",
			AppFile:       "app.log",
			AccessFile:    "access.log",
			ErrorFile:     "error.log",
			AlsoStdout:    true,
			SQLLevel:      "warn",
			SQLSlowMS:     500,
			IncludeSource: false,
		},
		AdminConfig: AdminConfig{
			Enabled:           true,
			Username:          "admin",
			Password:          "admin123456",
			Title:             "Couple Mini Admin",
			SampleIntervalSec: 5,
			HistoryLimit:      120,
		},
		AuthConfig: AuthConfig{
			TokenSecret:   "change-me-token-secret",
			TokenTTLHours: 168,
		},
		WeChatConfig: WeChatConfig{
			AppID:                       "",
			Secret:                      "",
			SubscribeNoticeTemplateID:   "",
			SubscribeScheduleTemplateID: "",
			SubscribeTitleKey:           "thing1",
			SubscribeContentKey:         "thing2",
			SubscribePage:               "pages/home/home",
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
	if value := os.Getenv("LOG_ERROR_FILE"); value != "" {
		cfg.LogConfig.ErrorFile = value
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

	if value, ok := envBool("ADMIN_ENABLED"); ok {
		cfg.AdminConfig.Enabled = value
	}
	if value := os.Getenv("ADMIN_USERNAME"); value != "" {
		cfg.AdminConfig.Username = value
	}
	if value := os.Getenv("ADMIN_PASSWORD"); value != "" || hasEnv("ADMIN_PASSWORD") {
		cfg.AdminConfig.Password = value
	}
	if value := os.Getenv("ADMIN_TITLE"); value != "" {
		cfg.AdminConfig.Title = value
	}
	if value := os.Getenv("ADMIN_SAMPLE_INTERVAL_SEC"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			cfg.AdminConfig.SampleIntervalSec = parsed
		}
	}
	if value := os.Getenv("ADMIN_HISTORY_LIMIT"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			cfg.AdminConfig.HistoryLimit = parsed
		}
	}

	if value := os.Getenv("AUTH_TOKEN_SECRET"); value != "" || hasEnv("AUTH_TOKEN_SECRET") {
		cfg.AuthConfig.TokenSecret = value
	}
	if value := os.Getenv("AUTH_TOKEN_TTL_HOURS"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			cfg.AuthConfig.TokenTTLHours = parsed
		}
	}

	if value := os.Getenv("WECHAT_APP_ID"); value != "" {
		cfg.WeChatConfig.AppID = value
	}
	if value := os.Getenv("WECHAT_SECRET"); value != "" || hasEnv("WECHAT_SECRET") {
		cfg.WeChatConfig.Secret = value
	}
	if value := os.Getenv("WECHAT_SUBSCRIBE_NOTICE_TEMPLATE_ID"); value != "" {
		cfg.WeChatConfig.SubscribeNoticeTemplateID = value
	}
	if value := os.Getenv("WECHAT_SUBSCRIBE_SCHEDULE_TEMPLATE_ID"); value != "" {
		cfg.WeChatConfig.SubscribeScheduleTemplateID = value
	}
	if value := os.Getenv("WECHAT_SUBSCRIBE_TITLE_KEY"); value != "" {
		cfg.WeChatConfig.SubscribeTitleKey = value
	}
	if value := os.Getenv("WECHAT_SUBSCRIBE_CONTENT_KEY"); value != "" {
		cfg.WeChatConfig.SubscribeContentKey = value
	}
	if value := os.Getenv("WECHAT_SUBSCRIBE_PAGE"); value != "" {
		cfg.WeChatConfig.SubscribePage = value
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
