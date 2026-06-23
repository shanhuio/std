package fakeweb

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"time"
)

// tlsCert contains a certificate in memory.
type tlsCert struct {
	// Marshalled PEM block for the certificate and the private key.
	pemCert, pemKey []byte
}

// x509KeyPair converts the PEM blocks into a X509 key pair
// for use in an HTTPS server.
func (c *tlsCert) x509KeyPair() (tls.Certificate, error) {
	return tls.X509KeyPair(c.pemCert, c.pemKey)
}

// tlsCertConfig is the configuration for creating a TLS certificate.
type tlsCertConfig struct {
	hosts    []string
	isCA     bool
	start    *time.Time
	duration time.Duration
}

func (c *tlsCertConfig) notValidBefore() time.Time {
	if c.start != nil {
		return *c.start
	}
	return time.Now()
}

func (c *tlsCertConfig) validDuration() time.Duration {
	if c.duration <= 0 {
		return 8 * time.Hour // default to 8 hours for testing.
	}
	return c.duration
}

func makeTLSCertTemplate(c *tlsCertConfig) (*x509.Certificate, error) {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, fmt.Errorf("generate serial number: %w", err)
	}

	const org = "Acme Co"
	const keyUsage = x509.KeyUsageKeyEncipherment |
		x509.KeyUsageDigitalSignature

	notValidBefore := c.notValidBefore()
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject:      pkix.Name{Organization: []string{org}},
		NotBefore:    notValidBefore,
		NotAfter:     notValidBefore.Add(c.validDuration()),

		KeyUsage:    keyUsage,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},

		BasicConstraintsValid: true,
	}

	for _, h := range c.hosts {
		if ip := net.ParseIP(h); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, h)
		}
	}

	if c.isCA {
		template.IsCA = true
		template.KeyUsage |= x509.KeyUsageCertSign
	}

	return &template, nil
}

// makeEC256TLSCert creates a ECDSA-256 TLS certificate.
func makeEC256TLSCert(c *tlsCertConfig) (*tlsCert, error) {
	return makeECCTLSCert(c, elliptic.P256())
}

// makeECCTLSCert creates a ECDSA-based HTTPs certificate.
func makeECCTLSCert(c *tlsCertConfig, curve elliptic.Curve) (*tlsCert, error) {
	if len(c.hosts) == 0 {
		return nil, errors.New("no host specified")
	}

	priv, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate private key: %w", err)
	}
	template, err := makeTLSCertTemplate(c)
	if err != nil {
		return nil, fmt.Errorf("make certificate template: %w", err)
	}

	derBytes, err := x509.CreateCertificate(
		rand.Reader, template, template, &priv.PublicKey, priv,
	)
	if err != nil {
		return nil, fmt.Errorf("create certificate: %w", err)
	}

	certOut := new(bytes.Buffer)
	if err := pem.Encode(
		certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes},
	); err != nil {
		return nil, fmt.Errorf("write certificate pem: %w", err)
	}

	pemBytes, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return nil, fmt.Errorf("marshal key bytes: %w", err)
	}
	keyOut := new(bytes.Buffer)
	if err := pem.Encode(
		keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: pemBytes},
	); err != nil {
		return nil, fmt.Errorf("write private key pem: %w", err)
	}

	return &tlsCert{pemCert: certOut.Bytes(), pemKey: keyOut.Bytes()}, nil
}

// tlsConfigs creates the certificate setup required for a set of domains.
type tlsConfigs struct {
	Domains []string
	Server  *tls.Config
	Client  *tls.Config
}

// newTLSConfigs creates a new setup with proper TLS config and HTTP
func newTLSConfigs(domains []string) (*tlsConfigs, error) {
	hosts := append([]string{}, domains...)
	c := &tlsCertConfig{
		hosts: hosts,
		isCA:  true,
	}
	cert, err := makeEC256TLSCert(c)
	if err != nil {
		return nil, fmt.Errorf("make TLS cert: %w", err)
	}

	tlsCert, err := cert.x509KeyPair()
	if err != nil {
		return nil, fmt.Errorf("unmarshal TLS cert: %w", err)
	}

	serverConfig := &tls.Config{
		NextProtos:   []string{"http/1.1", "h2"},
		Certificates: []tls.Certificate{tlsCert},
	}

	x509Cert, err := x509.ParseCertificate(tlsCert.Certificate[0])
	if err != nil {
		return nil, fmt.Errorf("parse x509 cert error: %w", err)
	}

	certPool := x509.NewCertPool()
	certPool.AddCert(x509Cert)

	return &tlsConfigs{
		Domains: hosts,
		Server:  serverConfig,
		Client:  &tls.Config{RootCAs: certPool},
	}, nil
}
