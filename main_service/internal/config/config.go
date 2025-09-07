package config

import (
	"log"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env        string        `yaml:"env" env-default:"local"`
	Clients    ClientsConfig `yaml:"clients"`
	AppSecret  string        `yaml:"app_secret" env-required:"true" env:"APP_SECRET"`
	HTTPServer `yaml:"http_server"`
	Postgres   `yaml:"postgres"`
	Redis      `yaml:"redis"`
	RabbitMQ   `yaml:"rabbitmq"`
}

type HTTPServer struct {
	Address     string        `yaml:"address" env-default:"localhost:8080"`
	Timeout     time.Duration `yaml:"timeout" env-default:"4s"`
	IdleTimeout time.Duration `yaml:"idle_timeout" env-default:"60s"`
}

type Client struct {
	Address      string        `yaml:"address"`
	Timeout      time.Duration `yaml:"timeout"`
	RetriesCount int           `yaml:"retries_count"`
}

type ClientsConfig struct {
	SSO Client `yaml:"sso"`
}

type Postgres struct {
	Host     string `yaml:"host" env-default:"postgres"`
	Port     int    `yaml:"port" env-default:"5432"`
	User     string `yaml:"user" env-required:"true"`
	Password string `yaml:"password" env-required:"true"`
	DBName   string `yaml:"dbname" env-required:"true"`
	SSLMode  string `yaml:"sslmode" env-default:"disabled"`
}

type Redis struct {
	Host     string `yaml:"host" env-default:"redis"`
	Password string `yaml:"password" env-required:"true"`
	DB       int    `yaml:"db" env-default:"0"`
}

type RabbitMQ struct {
	URL       string `yaml:"url" env-required:"true"`
	QueueName string `yaml:"queue_name" env-default:"notifications_queue"`
}

func MustLoad(configPath string) *Config {
	// проверка существования файла
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Fatalf("config file does not exist: %s", configPath)
	}

	var cfg Config

	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Fatalf("cannot read config: %s", configPath)
	}

	return &cfg
}
