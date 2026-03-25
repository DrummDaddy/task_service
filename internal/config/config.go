package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Env  string `mapstructure:"env"`
	HTTP struct {
		Addr string `mapstructure:"addr"`
	} `mapstructure:"http"`
	MySQL struct {
		DSN             string        `mapstructure:"dsn"`
		MaxOpenConns    int           `mapstructure:"max_open_conns"`
		MaxIdleConns    int           `mapstructure:"max_idle_conns"`
		ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	} `mapstructure:"mysql"`
	Redis struct {
		Addr string `mapstructure:"addr"`
		DB   int    `mapstructure:"db"`
	} `mapstructure:"redis"`
	Auth struct {
		JWTSecret        string        `mapstructure:"jwt_secret"`
		AccessTokenTTL   time.Duration `mapstructure:"access_token_ttl"`
		PasswordPepper   string        `mapstructure:"password_pepper"`
		PasswordHashCost int           `mapstructure:"password_hash_cost"`
	} `mapstructure:"auth"`
	Cache struct {
		TasksTTL time.Duration `mapstructure:"tasks_ttl"`
	} `mapstructure:"cache"`
	RateLimit struct {
		PerUserPerMinute int `mapstructure:"per_user_per_minute"`
	} `mapstructure:"rate_limit"`
	Email struct {
		BaseUrl string        `mapstructure:"base_url"`
		Timeout time.Duration `mapstructure:"timeout"`
	} `mapstructure:"email"`
}

func Load() (Config, error) {
	v := viper.New()

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./configs")

	v.SetEnvPrefix("TASK")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	v.BindEnv("http.addr", "TASK_HTTP_ADDR")
	v.BindEnv("mysql.dsn", "TASK_MYSQL_DSN")
	v.BindEnv("mysql.max_open_conns", "TASK_MYSQL_MAX_OPEN_CONNS")
	v.BindEnv("mysql.max_idle_conns", "TASK_MYSQL_MAX_IDLE_CONNS")
	v.BindEnv("mysql.conn_max_lifetime", "TASK_MYSQL_CONN_MAX_LIFETIME")
	v.BindEnv("redis.addr", "TASK_REDIS_ADDR")
	v.BindEnv("redis.db", "TASK_REDIS_DB")
	v.BindEnv("auth.jwt_secret", "TASK_AUTH_JWTSECRET")
	v.BindEnv("auth.access_token_ttl", "TASK_AUTH_ACCESS_TOKEN_TTL")
	v.BindEnv("auth.password_pepper", "TASK_AUTH_PASSWORD_PEPPER")
	v.BindEnv("auth.password_hash_cost", "TASK_AUTH_PASSWORD_HASH_COST")
	v.BindEnv("cache.tasks_ttl", "TASK_CACHE_TASKS_TTL")
	v.BindEnv("rate_limit.per_user_per_minute", "TASK_RATE_LIMIT_PER_USER_PER_MINUTE")
	v.BindEnv("email.base_url", "TASK_EMAIL_BASE_URL")
	v.BindEnv("email.timeout", "TASK_EMAIL_TIMEOUT")

	setDefault(v)
	_ = v.ReadInConfig()

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("config unmarshal: %w", err)
	}

	if cfg.Auth.JWTSecret == "" {
		return Config{}, fmt.Errorf("jwt secret required")
	}

	return cfg, nil
}

func setDefault(v *viper.Viper) {
	v.SetDefault("env", "dev")
	v.SetDefault("http.addr", ":8080")
	v.SetDefault("mysql.max_open_conns", 25)
	v.SetDefault("mysql.max_idle_conns", 25)
	v.SetDefault("mysql.conn_max_lifetime", 5*time.Minute)
	v.SetDefault("redis.addr", ":6379")
	v.SetDefault("redis.db", 0)
	v.SetDefault("auth.access_token_ttl", "24h")
	v.SetDefault("auth.password_hash_cost", 12)
	v.SetDefault("cache.tasks_ttl", "5m")
	v.SetDefault("rate_limit.per_user_per_minute", 100)
	v.SetDefault("email.timeout", "2s")
}
