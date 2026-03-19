package config

import (
	"fmt"
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
	v.AutomaticEnv()

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
	v.SetDefault("mysql.conn_max_lifetime", 5*time.Second)
	v.SetDefault("redis.addr", ":6379")
	v.SetDefault("redis.db", 0)
	v.SetDefault("auth.access_token_ttl", "24h")
	v.SetDefault("auth.password_hash_cost", 12)
	v.SetDefault("cache.tasks_ttl", "5m")
	v.SetDefault("rate_limit.per_user_per_minute", 100)
	v.SetDefault("email.timeout", "2s")
}
