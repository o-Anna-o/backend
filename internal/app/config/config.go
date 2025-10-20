package config

import (
	"os"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Config struct {
	ServiceHost   string
	ServicePort   int
	RedisEndpoint string
	RedisPassword string
	JwtKey        string
}

func NewConfig() (*Config, error) {
	var err error
	configName := "config"
	if os.Getenv("CONFIG_NAME") != "" {
		configName = os.Getenv("CONFIG_NAME")
	}

	viper.SetConfigName(configName)
	viper.SetConfigType("toml")
	viper.AddConfigPath("config")
	viper.AddConfigPath(".")
	viper.WatchConfig()

	err = viper.ReadInConfig()
	if err != nil {
		return nil, err
	}

	// Чтение .env
	err = godotenv.Load()
	if err != nil {
		logrus.Warn("Error loading .env file, using defaults")
	}

	viper.BindEnv("RedisEndpoint", "REDIS_ENDPOINT")
	viper.BindEnv("RedisPassword", "REDIS_PASSWORD")
	viper.BindEnv("JwtKey", "JWT_KEY")

	cfg := &Config{}
	err = viper.Unmarshal(cfg)
	if err != nil {
		return nil, err
	}

	logrus.Info("config parsed")
	return cfg, nil
}
