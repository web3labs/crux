package config

import (
	"testing"
	"reflect"
)

const configFile = "config_testdata.conf"

func TestLoadConfig(t *testing.T) {
	conf, err := LoadConfig(configFile)

	if err != nil {
		t.Fatalf("Unable to load config file: %s, %s", configFile, err)
	}

	expected := map[string]interface{}{
		"verbosity": 1,
		"alwayssendto": []interface{}{},
		"tlsserverchain": []interface{}{},
		"storage": "dir:storage",
		"workdir": "data",
		"url": "http://127.0.0.1:9001/",
		"tlsservertrust": "tofu",
		"publickeys": [1]string{"foo.pub"},
		"othernodes": [1]string{"http://127.0.0.1:9000/"},
		"tlsknownservers":"tls-known-servers",
		"tlsclientcert": "tls-client-cert.pem",
		"privatekeys": [1]string{"foo.key"},
		"tlsservercert": "tls-server-cert.pem",
		"tls": "strict",
		"tlsknownclients": "tls-known-clients",
		"tlsclientchain": []interface{}{},
		"tlsclientkey": "tls-client-key.pem",
		"socket": "constellation.ipc",
		"tlsclienttrust": "ca-or-tofu",
		"tlsserverkey": "tls-server-key.pem",
		"port": 9001,
	}

	verifyConfig(t, conf, expected)
}

func verifyConfig(t *testing.T, conf map[string]interface{}, expected map[string]interface{}) {
	for expK, expV := range expected {
		//if conf[key] != value {
		if actV, ok := conf[expK]; !ok {
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
