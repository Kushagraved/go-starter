// Package config loads the typed application configuration.
//
// Values come from config/<env>/config.yaml via Viper, and any of them can be
// overridden by an environment variable: uppercase the path, replace dots with
// underscores, prefix with APP_.
//
//	settings.sql.primary.password  ->  APP_SETTINGS_SQL_PRIMARY_PASSWORD
//	settings.redis.host            ->  APP_SETTINGS_REDIS_HOST
//
// That keeps secrets out of the YAML in real environments while the committed dev
// config stays runnable as-is.
//
// Struct fields are tagged `yaml` (not `mapstructure`) so the tags match the file
// format they describe; Load points Viper's decoder at the yaml tag name.
package config

import (
	"strings"
	"sync"
	"time"

	"github.com/go-viper/mapstructure/v2"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

const envPrefix = "APP"

var (
	once     sync.Once
	instance *Config
	loadErr  error
)

type Config struct {
	Settings Settings `yaml:"settings"`
}

type Settings struct {
	Env     string     `yaml:"env"`
	Server  ServerCfg  `yaml:"server"`
	Logger  LoggerCfg  `yaml:"logger"`
	SQL     SQLCfg     `yaml:"sql"`
	Redis   RedisCfg   `yaml:"redis"`
	AWS     AWSCfg     `yaml:"aws"`
	Storage StorageCfg `yaml:"storage"`
	SQS     SQSCfg     `yaml:"sqs"`
}

type ServerCfg struct {
	Port            string        `yaml:"port"`
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
}

type LoggerCfg struct {
	Level string `yaml:"level"`
}

type SQLCfg struct {
	Primary DBSettings `yaml:"primary"`
}

type DBSettings struct {
	Host               string        `yaml:"host"`
	Port               int           `yaml:"port"`
	DBName             string        `yaml:"db_name"`
	User               string        `yaml:"user"`
	Password           string        `yaml:"password"`
	SSLMode            string        `yaml:"ssl_mode"`
	MaxOpenConnections int           `yaml:"max_open_connections"`
	MaxIdleConnections int           `yaml:"max_idle_connections"`
	ConnMaxLifetime    time.Duration `yaml:"conn_max_lifetime"`
}

type RedisCfg struct {
	Host        string        `yaml:"host"`
	Password    string        `yaml:"password"`
	DialTimeout time.Duration `yaml:"dial_timeout"`
	ReadTimeout time.Duration `yaml:"read_timeout"`
}

type AWSCfg struct {
	Region string `yaml:"region"`
	// Endpoint overrides the endpoint for all AWS services. Point it at LocalStack
	// locally; leave empty in real environments.
	Endpoint string `yaml:"endpoint"`
	// Credentials are for LocalStack only. Leave blank elsewhere so the default
	// credential chain is used.
	AccessKeyID     string `yaml:"access_key_id"`
	SecretAccessKey string `yaml:"secret_access_key"`
}

type StorageCfg struct {
	// Bucket is the default object storage bucket.
	Bucket string `yaml:"bucket"`
}

/*
SQSCfg holds queue configuration. Each queue gets its own named field rather than
living in a map, so a queue is referenced as a compile-time checked
`settings.SQS.TestEvents` instead of a string lookup that fails silently.

Add one QueueCfg field per queue:

	type SQSCfg struct {
	    ...
	    UserEvents  QueueCfg `yaml:"user_events"`
	    OrderEvents QueueCfg `yaml:"order_events"`
	}
*/
type SQSCfg struct {
	// Enable is the master switch. False starts no consumers at all.
	Enable bool `yaml:"enable"`
	// Workers is the default worker-pool size when a queue does not set its own.
	Workers int `yaml:"workers"`
	// AutoCommit=false deletes a message only after it is processed successfully.
	AutoCommit bool `yaml:"auto_commit"`

	// TestEvents is a sample queue, created in LocalStack and ready to use.
	TestEvents QueueCfg `yaml:"test_events"`
}

// QueueCfg configures a single queue's consumer.
type QueueCfg struct {
	Enable bool `yaml:"enable"`
	// QueueName is the SQS queue name; the URL is resolved at runtime.
	QueueName string `yaml:"queue_name"`
	// Workers overrides SQSCfg.Workers for this queue.
	Workers int `yaml:"workers"`
	// MaxMessages is how many messages a single poll fetches (SQS max is 10).
	MaxMessages int32 `yaml:"max_messages"`
	// VisibilityTimeout is how long a message stays hidden while being processed.
	VisibilityTimeout int32 `yaml:"visibility_timeout"`
	// WaitTimeSeconds is the long-poll duration; 20 is the SQS max.
	WaitTimeSeconds int32 `yaml:"wait_time_seconds"`
	// RequireExpBackoff widens the visibility timeout on each retry instead of
	// letting a failing message hot-loop.
	RequireExpBackoff bool `yaml:"require_exp_backoff"`
}

// Load reads the config file at path once and caches the result.
func Load(path string) (*Config, error) {
	once.Do(func() {
		v := viper.New()
		v.SetConfigFile(path)
		v.SetConfigType("yaml")
		v.SetEnvPrefix(envPrefix)
		v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
		v.AutomaticEnv()

		setDefaults(v)

		if err := v.ReadInConfig(); err != nil {
			loadErr = errors.Wrapf(err, "failed to read config file %s", path)
			return
		}

		var cfg Config
		// Read the `yaml` struct tags instead of mapstructure's default. Viper's
		// built-in decode hooks (string -> time.Duration, comma -> slice) are set
		// before this option runs, so they still apply.
		if err := v.Unmarshal(&cfg, func(dc *mapstructure.DecoderConfig) {
			dc.TagName = "yaml"
		}); err != nil {
			loadErr = errors.Wrap(err, "failed to unmarshal config")
			return
		}

		if err := cfg.validate(); err != nil {
			loadErr = err
			return
		}
		instance = &cfg
	})
	return instance, loadErr
}

// Get returns the loaded config. Load must have been called first.
func Get() *Config {
	if instance == nil {
		panic("config not loaded; call config.Load() first")
	}
	return instance
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("settings.server.port", "8080")
	v.SetDefault("settings.server.read_timeout", "15s")
	v.SetDefault("settings.server.write_timeout", "15s")
	v.SetDefault("settings.server.shutdown_timeout", "10s")
	v.SetDefault("settings.logger.level", "info")
	v.SetDefault("settings.sql.primary.ssl_mode", "disable")
	v.SetDefault("settings.sql.primary.max_open_connections", 10)
	v.SetDefault("settings.sql.primary.max_idle_connections", 5)
	v.SetDefault("settings.sql.primary.conn_max_lifetime", "30m")
	v.SetDefault("settings.redis.dial_timeout", "5s")
	v.SetDefault("settings.redis.read_timeout", "3s")
	v.SetDefault("settings.sqs.workers", 2)
}

// validate fails fast at boot instead of surfacing a confusing connection error
// on the first request.
func (c *Config) validate() error {
	s := c.Settings
	if s.SQL.Primary.Host == "" || s.SQL.Primary.DBName == "" {
		return errors.New("settings.sql.primary.host and .db_name are required")
	}
	if s.Redis.Host == "" {
		return errors.New("settings.redis.host is required")
	}
	if s.AWS.Region == "" {
		return errors.New("settings.aws.region is required")
	}
	return nil
}
