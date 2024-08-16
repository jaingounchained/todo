package util

import (
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Environment             string        `mapstructure:"ENVIRONMENT"`
	DBDriver                string        `mapstructure:"DB_DRIVER"`
	DBSource                string        `mapstructure:"DB_SOURCE"`
	HTTPServerAddress       string        `mapstructure:"HTTP_SERVER_ADDRESS"`
	GRPCServerAddress       string        `mapstructure:"GRPC_SERVER_ADDRESS"`
	StorageType             string        `mapstructure:"STORAGE_TYPE"`
	LocalStorageDirectory   string        `mapstructure:"LOCAL_STORAGE_DIRECTORY"`
	TokenSymmetricKey       string        `mapstructure:"TOKEN_SYMMETRIC_KEY"`
	AccessTokenDuration     time.Duration `mapstructure:"ACCESS_TOKEN_DURATION"`
	RefreshTokenDuration    time.Duration `mapstructure:"REFRESH_TOKEN_DURATION"`
	MigrationURL            string        `mapstructure:"MIGRATION_URL"`
	RedisAddress            string        `mapstructure:"REDIS_ADDRESS"`
	EmailSenderName         string        `mapstructure:"EMAIL_SENDER_NAME"`
	EmailSenderAddress      string        `mapstructure:"EMAIL_SENDER_ADDRESS"`
	EmailSenderPassword     string        `mapstructure:"EMAIL_SENDER_PASSWORD"`
	MockEmailSenderLocalDir string        `mapstructure:"MOCK_EMAIL_SENDER_LOCAL_DIR"`
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
