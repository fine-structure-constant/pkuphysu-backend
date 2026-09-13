package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

var (
	Conf *Config
)

const DefaultFileBaseDir = "./data/static/files"

type Database struct {
	Host        string `mapstructure:"HOST"`
	Port        int    `mapstructure:"PORT"`
	User        string `mapstructure:"USER"`
	Password    string `mapstructure:"PASSWORD"`
	DBName      string `mapstructure:"NAME"`
	SSLMode     string `mapstructure:"SSL_MODE"`
	TablePrefix string `mapstructure:"TABLE_PREFIX"`
}

type Cors struct {
	AllowOrigins []string `json:"allow_origins" mapstructure:"ALLOW_ORIGINS"`
	AllowMethods []string `json:"allow_methods" mapstructure:"ALLOW_METHODS"`
	AllowHeaders []string `json:"allow_headers" mapstructure:"ALLOW_HEADERS"`
}

type LogConfig struct {
	Enable     bool   `mapstructure:"ENABLE"`
	Level      string `mapstructure:"LEVEL"`
	MaxSize    int    `mapstructure:"MAX_SIZE"`
	MaxAge     int    `mapstructure:"MAX_AGE"`
	MaxBackups int    `mapstructure:"MAX_BACKUPS"`
	Compress   bool   `mapstructure:"COMPRESS"`
	FilePath   string `mapstructure:"FILE_PATH"`
}

type EmailConfig struct {
	Host     string `mapstructure:"HOST"`
	Port     int    `mapstructure:"PORT"`
	Username string `mapstructure:"USERNAME"`
	Password string `mapstructure:"PASSWORD"`
	From     string `mapstructure:"FROM"`
}

type RateLimitConfig struct {
	RequestsPerSecond int           `mapstructure:"REQUESTS_PER_SECOND"`
	Burst             int           `mapstructure:"BURST"`
	BlockDuration     time.Duration `mapstructure:"BLOCK_DURATION"`
}

type StorageConfig struct {
	BaseDir string `mapstructure:"BASE_DIR"`
}

type Config struct {
	Port        int             `mapstructure:"PORT"`
	JwtSecret   string          `mapstructure:"JWT_SECRET"`
	TokenExpire int             `mapstructure:"TOKEN_EXPIRE"`
	Database    Database        `mapstructure:"DATABASE"`
	LogConfig   LogConfig       `mapstructure:"log_config"`
	Cors        Cors            `mapstructure:"cors"`
	Email       EmailConfig     `mapstructure:"email"`
	RateLimit   RateLimitConfig `mapstructure:"rate_limit"`
	Storage     StorageConfig   `mapstructure:"storage"`
}

func LoadConfig() (*Config, error) {
	configName := "config"
	if environment := os.Getenv("PKUPHYSU_ENV"); environment != "" {
		configName = "config." + environment
	} else if _, err := os.Stat(filepath.Join("data", "config", "config.dev.toml")); err == nil {
		configName = "config.dev"
	}
	viper.SetConfigName(configName)
	viper.SetConfigType("toml")

	viper.AddConfigPath("./data/config")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return nil, fmt.Errorf("config file not found")
		}
		return nil, fmt.Errorf("config file read error: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unable to decode config: %w", err)
	}
	if strings.TrimSpace(cfg.Storage.BaseDir) == "" {
		cfg.Storage.BaseDir = DefaultFileBaseDir
	}

	return &cfg, nil
}

func FileBaseDir() string {
	if Conf == nil || strings.TrimSpace(Conf.Storage.BaseDir) == "" {
		return DefaultFileBaseDir
	}
	return Conf.Storage.BaseDir
}

func InitConfig() {
	cfg, err := LoadConfig()
	if err != nil {
		panic(err)
	}
	Conf = cfg
}
