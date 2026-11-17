package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type AuthConfig struct {
	Username string `yaml:"-" env:"API_USERNAME" env-required:"true"`
	Password string `yaml:"-" env:"API_PASSWORD" env-required:"true"`
}

type HTTPConfig struct {
	Address           string        `yaml:"address" env:"HTTP_ADDRESS"`
	ReadHeaderTimeout time.Duration `yaml:"read_header_timeout" env:"HTTP_READ_HEADER_TIMEOUT"`
}

type AppSettingsConfig struct {
	MaxAliasLength int `yaml:"maxAliasLength" env:"MAX_ALIAS_LENGTH"`
}

type PostgresConfig struct {
	Host     string `yaml:"host" env:"POSTGRES_HOST"`
	Port     int    `yaml:"port" env:"POSTGRES_PORT"`
	Database string `yaml:"-" env:"POSTGRES_DB" env-required:"true"`
	User     string `yaml:"-" env:"POSTGRES_USER" env-required:"true"`
	Password string `yaml:"-" env:"POSTGRES_PASSWORD" env-required:"true"`
}

type AppConfig struct {
	Env         string            `yaml:"env" env:"APP_ENV"`
	HTTP        HTTPConfig        `yaml:"http"`
	Postgres    PostgresConfig    `yaml:"postgres"`
	AppSettings AppSettingsConfig `yaml:"settings"`
	Auth        AuthConfig        `yaml:"-"`
}

func Load() (*AppConfig, error) {
	var envOnly struct{}

	// load .env vars in process env
	if err := cleanenv.ReadConfig(".env", &envOnly); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("load .env: %w", err)
		}
	}

	path := os.Getenv("CONFIG_PATH")
	if path == "" {
		return nil, fmt.Errorf("CONFIG_PATH is empty")
	}

	var cfg AppConfig

	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		return nil, fmt.Errorf("load config %q: %w", path, err)
	}

	if cfg.Auth.Username == "" || cfg.Auth.Password == "" {
		return nil, fmt.Errorf("API_USERNAME and API_PASSWORD must not be empty")
	}

	return &cfg, nil
}
