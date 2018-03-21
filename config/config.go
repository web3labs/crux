package config

import (
	"github.com/spf13/viper"
)

const (
	Verbosity = "verbosity"
	AlwaysSendTo = "alwayssendto"
	TlsServerChain = "tlsserverchain"
	Storage = "storage"
	WorkDir = "workdir"
	Url = "url"
	TlsServerTrust = "tlsservertrust"
	PublicKeys = "publickeys"
	OtherNodes = "othernodes"
	TlsKnownServers = "tlsknownservers"
	TlsClientCert = "tlsclientcert"
	PrivateKeys = "privatekeys"
	TlsServerCert = "tlsservercert"
	Tls = "tls"
	TlsKnownClients = "tlsknownclients"
	TlsClientChain = "tlsclientchain"
	TlsClientKey = "tlsclientkey"
	Socket = "socket"
	TlsClientTrust = "tlsclienttrust"
	TlsServerKey = "tlsserverkey"
	Port = "port"
)

func LoadConfig(configPath string) error {
	viper.SetConfigType("hcl")
	viper.SetConfigFile(configPath)
	return viper.ReadInConfig()
}

func AllSettings() map[string]interface{} {
	return viper.AllSettings()
}

func GetInt(key string) int {
	return viper.GetInt(key)
}

func GetString(key string) string {
	return viper.GetString(key)
}

func GetStringSlice(key string) []string {
	return viper.GetStringSlice(key)
}


