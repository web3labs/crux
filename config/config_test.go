package config

import (
	"testing"
	"reflect"
)

const configFile = "config_testdata.conf"

func TestInitFlags(t *testing.T) {
	InitFlags()
	conf := AllSettings()
	expected := map[string]interface{}{
		Port:         -1,
		Verbosity:    1,
		BerkeleyDb:   false,
		GenerateKeys: "",
		AlwaysSendTo: "",
		Storage:      "crux.db",
		WorkDir:      ".",
		Url:          "",
		PublicKeys:   "",
		OtherNodes:   "",
		PrivateKeys:  "",
		Socket:       "crux.ipc",
	}

	verifyConfig(t, conf, expected)
}

func TestLoadConfig(t *testing.T) {
	err := LoadConfig(configFile)

	if err != nil {
		t.Fatalf("Unable to load config file: %s, %s", configFile, err)
	}

	conf := AllSettings()

	expected := map[string]interface{}{
		Verbosity: 1,
		AlwaysSendTo: []interface{}{},
		TlsServerChain: []interface{}{},
		Storage: "dir:storage",
		WorkDir: "data",
		Url: "http://127.0.0.1:9001/",
		TlsServerTrust: "tofu",
		PublicKeys: []interface{}{"foo.pub"},
		OtherNodes: []interface{}{"http://127.0.0.1:9000/"},
		TlsKnownServers:"tls-known-servers",
		TlsClientCert: "tls-client-cert.pem",
		PrivateKeys: []interface{}{"foo.key"},
		TlsServerCert: "tls-server-cert.pem",
		Tls: "strict",
		TlsKnownClients: "tls-known-clients",
		TlsClientChain: []interface{}{},
		TlsClientKey: "tls-client-key.pem",
		Socket: "constellation.ipc",
		TlsClientTrust: "ca-or-tofu",
		TlsServerKey: "tls-server-key.pem",
		Port: 9001,
	}

	verifyConfig(t, conf, expected)
}

func verifyConfig(t *testing.T, conf map[string]interface{}, expected map[string]interface{}) {
	for expK, expV := range expected {
		//if conf[key] != value {
		if actV, ok := conf[expK]; ok {
			var eq bool
			switch actV.(type) {  // we cannot use == for equality with []interface{}
			case []interface{}:
				eq = reflect.DeepEqual(actV, expV)
			default:
				eq = actV == expV
			}
			if !eq {
				t.Errorf("Key: %s with value %v could not be found", expK, expV)
			}
		}
	}
}

func TestGetBoolConfig(t *testing.T) {
	if GetBool(Verbosity) != true {
		t.Errorf("Verbosity is %t", GetBool(Verbosity))
	}
}

func TestGetIntConfig(t *testing.T) {
	if GetInt(Port) != 9001 {
		t.Errorf("Port num 9001 is expected but we got %d", GetInt(Port))
	}
}