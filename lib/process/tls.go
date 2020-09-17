package process

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"time"

	"github.com/gravitational/teleport/lib/utils"

	"github.com/gravitational/trace"
)

// GenerateSelfSignedCert generates a self signed certificate that
// is valid for given domain names and ips, returns PEM-encoded bytes with key and cert
func GenerateSelfSignedCert(hostNames []string) (*utils.TLSCredentials, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(time.Hour * 24 * 824) // 825 days

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	entity := pkix.Name{
		CommonName:   "localhost",
		Country:      []string{"US"},
		Organization: []string{"localhost"},
	}

	template := x509.Certificate{
		SerialNumber:          serialNumber,
		Issuer:                entity,
		Subject:               entity,
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	// collect IP addresses localhost resolves to and add them to the cert. template:
	template.DNSNames = append(hostNames, "localhost.local")
	ips, _ := net.LookupIP("localhost")
	if ips != nil {
		template.IPAddresses = ips
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(priv.Public())
	if err != nil {
		return nil, trace.Wrap(err)
	}

	return &utils.TLSCredentials{
		PublicKey:  pem.EncodeToMemory(&pem.Block{Type: "RSA PUBLIC KEY", Bytes: publicKeyBytes}),
		PrivateKey: pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)}),
		Cert:       pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes}),
	}, nil
}
