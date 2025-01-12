package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	Environment   string `mapstructure:"ENVIRONMENT"`
	DBSource      string `mapstructure:"DB_SOURCE"`
	Port          string `mapstructure:"PORT"`
	SmtpHost      string `mapstructure:"SMTP_HOST"`
	SmtpPort      int    `mapstructure:"SMTP_PORT"`
	SmtpUsername  string `mapstructure:"SMTP_USERNAME"`
	SmtpPassword  string `mapstructure:"SMTP_PASSWORD"`
	NatsServerUrl string `mapstructure:"NATS_SERVER_URL"`
}

func LoadConfig(path string) (config Config, err error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("app")
	viper.SetConfigType("env")

	viper.AutomaticEnv()
	err = viper.ReadInConfig()
	if err != nil {
		return
	}
	err = viper.Unmarshal(&config)
	return
}
