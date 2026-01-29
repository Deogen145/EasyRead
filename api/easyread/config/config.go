package config

import (
	"strings"
	"sync"

	"github.com/spf13/viper"
)

type (
	Config struct {
		Server *Server `mapstructure:"server"`
		DB     *DB     `mapstructure:"db"`
	}

	Server struct {
		Port string `mapstructure:"port"`
	}

	DB struct {
		Host     string `mapstructure:"host"`
		Port     int    `mapstructure:"port"`
		User     string `mapstructure:"user"`
		Password string `mapstructure:"password"`
		Name     string `mapstructure:"name"`
	}
)

var (
	once           sync.Once
	configInstance *Config
)

func GetConfig() *Config {
	once.Do(func() {
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
		viper.AutomaticEnv()
		viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

		if err := viper.ReadInConfig(); err != nil {
			panic(err)
		}

		var cfg Config
		if err := viper.Unmarshal(&cfg); err != nil {
			panic(err)
		}

		configInstance = &cfg
	})

	return configInstance
}
