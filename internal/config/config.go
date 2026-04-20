package config

import (
	"log"
	"sync"

	"gopkg.in/yaml.v3"
	"os"
)

var (
	once     sync.Once
	instance *Config
)

type Config struct {
	Settings Settings `yaml:"settings"`
}

type Settings struct {
	Env    string      `yaml:"env"`
	Server ServerCfg   `yaml:"server"`
	Logger LoggerCfg   `yaml:"logger"`
	DBDriver string    `yaml:"db_driver"` // "memory" | "postgres"
	SQL    SQLCfg      `yaml:"sql"`
	Redis  RedisCfg    `yaml:"redis"`
	Queue  QueueCfg    `yaml:"queue"`
}

type ServerCfg struct {
	Port string `yaml:"port"`
}

type LoggerCfg struct {
	Level string `yaml:"level"`
}

type SQLCfg struct {
	Primary DBSettings `yaml:"primary"`
}

type DBSettings struct {
	Host               string `yaml:"host"`
	Port               int    `yaml:"port"`
	DBName             string `yaml:"db_name"`
	User               string `yaml:"user"`
	Password           string `yaml:"password"`
	MaxOpenConnections int    `yaml:"max_open_connections"`
	MaxIdleConnections int    `yaml:"max_idle_connections"`
}

type RedisCfg struct {
	Host        string `yaml:"host"`
	Password    string `yaml:"password"`
	DB          int    `yaml:"db"`
	DialTimeout int    `yaml:"dial_timeout"`
	ReadTimeout int    `yaml:"read_timeout"`
}

type QueueCfg struct {
	Driver  string                 `yaml:"driver"` // "memory" | "sqs"
	Workers int                    `yaml:"workers"`
	Queues  map[string]QueueEntry  `yaml:"queues"`
}

type QueueEntry struct {
	Name    string `yaml:"name"`
	Enabled bool   `yaml:"enabled"`
}

func Load(path string) *Config {
	once.Do(func() {
		data, err := os.ReadFile(path)
		if err != nil {
			log.Fatalf("failed to read config file %s: %v", path, err)
		}
		var cfg Config
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			log.Fatalf("failed to parse config: %v", err)
		}
		instance = &cfg
	})
	return instance
}

func Get() *Config {
	if instance == nil {
		log.Fatal("config not loaded; call config.Load() first")
	}
	return instance
}
