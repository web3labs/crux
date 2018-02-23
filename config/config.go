package config

import (
	"github.com/spf13/viper"
)

func LoadConfig(configPath string) (map[string]interface{}, error) {
	viper.SetConfigType("hcl")
	viper.SetConfigFile(configPath)
	err := viper.ReadInConfig()
	if err != nil {
		return nil, err
	}
	return viper.AllSettings(), err
}