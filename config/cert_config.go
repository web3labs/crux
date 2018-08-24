package config

import (
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"fmt"
	"os"
	log "github.com/sirupsen/logrus"
)

const (
	OrgName = "org-name"
	CountryCode = "country-code"
	Province = "province"
	Locality = "locality"
	Address = "address"
	PostalCode = "postal-code"

	ValidityYears = "validity-years"
	ValidityMonths = "validity-months"
	ValidityDays = "validity-days"

	IsCA = "is-ca"
	KeyBits = "key-bits"
	FileName = "filename"
)

var v *viper.Viper
var flags *pflag.FlagSet

// CertInitFlags initializes all Certificate generation supported flags.
func CertInitFlags() {
	v = viper.New()
	flags = pflag.NewFlagSet("flags", pflag.ExitOnError)
	flags.StringSlice(OrgName, []string{}, "Organisation name")
	flags.StringSlice(CountryCode, []string{}, "Country coder")
	flags.StringSlice(Province, []string{}, "Province")
	flags.StringSlice(Locality, []string{}, "Locality")
	flags.StringSlice(Address, []string{}, "Address")
	flags.StringSlice(PostalCode, []string{}, "Postal Code")

	flags.String(FileName, "ca", "File name of the Certificates")

	flags.Int(ValidityDays, 0, "Validity of the certificate - days")
	flags.Int(ValidityMonths, 0, "Validity of the certificate - months")
	flags.Int(ValidityYears, 10, "Validity of the certificate - years")
	flags.Int(KeyBits, 2048, "The number of bits in the Key")

	flags.Bool(IsCA, true, "Is Certificate Authority")

	v.BindPFlags(flags)  // Binding the flags to test the initial configuration
}

// CertUsage prints usage instructions to the console.
func CertUsage() {
	if flags == nil {
		log.Fatalf("Certificate flags not initialised ")
	}
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "      %-25s%s\n", "certificate.conf", "Certificate config file")
	flags.PrintDefaults()
}

// CertLoadConfig loads all configuration settings in the provided configPath location.
func CertLoadConfig(configPath string) error {
	if v == nil {
		log.Fatalf("Certificate configuration not initialised ")
	}
	v.SetConfigType("hcl")
	v.SetConfigFile(configPath)
	return v.ReadInConfig()
}

func CertAllSettings() map[string]interface{} {
	return v.AllSettings()
}

func CertGetBool(key string) bool {
	return v.GetBool(key)
}

func CertGetInt(key string) int {
	return v.GetInt(key)
}

func CertGetString(key string) string {
	return v.GetString(key)
}

func CertGetStringSlice(key string) []string {
	return v.GetStringSlice(key)
}


