// Package config provides the configuration settings to be used by the application at runtime
package config

import (
	"flag"
	"fmt"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"os"
)

const (
	Verbosity          = "verbosity"
	VerbosityShorthand = "v"
	AlwaysSendTo       = "alwayssendto"
	Storage            = "storage"
	WorkDir            = "workdir"
	Url                = "url"
	OtherNodes         = "othernodes"
	PublicKeys         = "publickeys"
	PrivateKeys        = "privatekeys"
	Port               = "port"
	Socket             = "socket"

	GenerateKeys = "generate-keys"

	BerkeleyDb   = "berkeleydb"
	UseGRPC      = "grpc"
	GrpcJsonPort = "grpcport"

	Tls             = "tls"
	TlsServerChain  = "tlsserverchain"
	TlsServerTrust  = "tlsservertrust"
	TlsKnownServers = "tlsknownservers"
	TlsClientCert   = "tlsclientcert"
	TlsServerCert   = "tlsservercert"
	TlsKnownClients = "tlsknownclients"
	TlsClientChain  = "tlsclientchain"
	TlsClientKey    = "tlsclientkey"
	TlsClientTrust  = "tlsclienttrust"
	TlsServerKey    = "tlsserverkey"
)

// InitFlags initializes all supported command line flags.
func InitFlags() {
	flag.String(GenerateKeys, "", "Generate a new keypair")
	flag.String(Url, "", "The URL to advertise to other nodes (reachable by them)")
	flag.Int(Port, -1, "The local port to listen on")
	flag.String(WorkDir, ".", "The folder to put stuff in ")
	flag.String(Socket, "crux.ipc", "IPC socket to create for access to the Private API")
	flag.String(OtherNodes, "", "\"Boot nodes\" to connect to to discover the network")
	flag.String(PublicKeys, "", "Public keys hosted by this node")
	flag.String(PrivateKeys, "", "Private keys hosted by this node")
	flag.String(Storage, "crux.db", "Database storage file name")
	flag.Bool(BerkeleyDb, false,
		"Use Berkeley DB for working with an existing Constellation data store [experimental]")

	flag.Int(Verbosity, 1, "Verbosity level of logs")
	flag.Int(VerbosityShorthand, 1, "Verbosity level of logs (shorthand)")
	flag.String(AlwaysSendTo, "", "List of public keys for nodes to send all transactions too")
	flag.Bool(UseGRPC, true, "Use gRPC server")
	flag.Bool(Tls, false, "Use TLS to secure HTTP communications")
	flag.String(TlsServerCert, "", "The server certificate to be used")
	flag.String(TlsServerKey, "", "The server private key")
	flag.Int(GrpcJsonPort, -1, "The local port to listen on for JSON extensions of gRPC")

	// storage not currently supported as we use LevelDB

	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	viper.BindPFlags(pflag.CommandLine) // Binding the flags to test the initial configuration
}

// Usage prints usage instructions to the console.
func Usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "      %-25s%s\n", "crux.config", "Optional config file")
	pflag.PrintDefaults()
}

// ParseCommandLine parses all provided command line arguments.
func ParseCommandLine() {
	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)
}

// LoadConfig loads all configuration settings in the provided configPath location.
func LoadConfig(configPath string) error {
	viper.SetConfigType("hcl")
	viper.SetConfigFile(configPath)
	return viper.ReadInConfig()
}

func AllSettings() map[string]interface{} {
	return viper.AllSettings()
}

func GetBool(key string) bool {
	return viper.GetBool(key)
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
