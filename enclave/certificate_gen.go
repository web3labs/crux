package enclave

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	log "github.com/sirupsen/logrus"
	"github.com/blk-io/crux/config"
	"math/big"
	"os"
	"time"
	"fmt"
)

func CertGen(configFile string) error {
	config.CertInitFlags()
	err := config.CertLoadConfig(configFile)
	if err != nil {
		log.Errorf("Loading from config failed with %v", err)
		return err
	}

	serialNo, err := rand.Int(rand.Reader, big.NewInt(32))
	if err != nil {
		log.Errorf("Generating random number failed with %v", err)
		return err
	}
	certificateParam := &x509.Certificate{
		SerialNumber: serialNo,
		Subject: pkix.Name{
			Organization:  config.CertGetStringSlice(config.OrgName),
			Country:       config.CertGetStringSlice(config.CountryCode),
			Province:      config.CertGetStringSlice(config.Province),
			Locality:      config.CertGetStringSlice(config.Locality),
			StreetAddress: config.CertGetStringSlice(config.Address),
			PostalCode:    config.CertGetStringSlice(config.PostalCode),
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(
			config.CertGetInt(config.ValidityYears),
			config.CertGetInt(config.ValidityMonths),
			config.CertGetInt(config.ValidityDays)),
		IsCA:                  config.CertGetBool(config.IsCA),
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Errorf("Generating private Key failed with %v", err)
		return err
	}

	publicKey := &privateKey.PublicKey
	certificate, err := x509.CreateCertificate(rand.Reader, certificateParam, certificateParam, publicKey, privateKey)
	if err != nil {
		log.Errorf("Create certificate failed %v", err)
		return err
	}

	// Public key
	certOut, err := os.Create(fmt.Sprintf("%s.crt", config.CertGetString(config.FileName)))
	if err != nil {
		log.Errorf("Writing public key to file failed with %v", err)
		return err
	}
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certificate})
	certOut.Close()

	// Private key
	keyOut, err := os.OpenFile (
		fmt.Sprintf("%s.key", config.CertGetString(config.FileName)),
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Errorf("Writing private key to file failed with %v", err)
		return err
	}
	pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})
	keyOut.Close()

	return nil
}