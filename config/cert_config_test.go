package config

import (
	"testing"
	)

const certConfigFile = "cert_config_testdata.conf"

func TestCertInitFlags(t *testing.T) {
	CertInitFlags()
	conf := CertAllSettings()
	expected := map[string]interface{}{
		OrgName:		[]string{},
		CountryCode:	[]string{},
		Province:		[]string{},
		Locality:		[]string{},
		Address:		[]string{},
		PostalCode:		[]string{},
		ValidityYears:	10,
		ValidityMonths:	0,
		ValidityDays:	0,
		IsCA:			true,
		KeyBits:		2048,
		FileName:		"ca",
	}
	verifyConfig(t, conf, expected)
}

func TestCertLoadConfig(t *testing.T) {
	err := CertLoadConfig(certConfigFile)

	if err != nil {
		t.Fatalf("Unable to load config file: %s, %v", certConfigFile, err)
	}

	conf := CertAllSettings()

	expected := map[string]interface{}{
		OrgName:		"blk-io",
		CountryCode:	"UK",
		Province:		"",
		Locality:		"London",
		Address:		"",
		PostalCode:		"",
		ValidityYears:	10,
		ValidityMonths:	0,
		ValidityDays:	0,
		IsCA:			true,
		KeyBits:		2048,
		FileName:		"tm",
	}

	verifyConfig(t, conf, expected)
}