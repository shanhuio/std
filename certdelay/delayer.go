package certdelay

import (
	"crypto/tls"
	"time"
)

// Delayer configures the delay behavior applied to newly issued certificates.
type Delayer struct {
	// NewCertDelay is how long to stall the response the first time a newly
	// issued certificate is requested. The delay gives logs of certificate
	// transparency time to catch up so strict browsers don't reject the cert
	// on SCT timestamp checks. Zero falls back to DefaultNewCertDelay.
	NewCertDelay time.Duration

	// NewCertMature is how long after a cert is first seen before it stops
	// being treated as new; once the cert is "mature", subsequent requests
	// skip the delay. Zero falls back to DefaultNewCertMature.
	NewCertMature time.Duration

	// NewCertWindow is the window after a cert's NotBefore during which it
	// is considered recent enough to warrant the delay treatment. Certs
	// whose NotBefore is older than this never get delayed. Zero falls back
	// to DefaultNewCertWindow.
	NewCertWindow time.Duration

	// CertForDomain, when non-nil, is consulted first for each request;
	// when it returns nil, the request falls through to the wrapped
	// GetCertificateFunc.
	CertForDomain func(domain string) *tls.Certificate
}

func (d *Delayer) toGetterConfig() *getterConfig {
	return &getterConfig{
		newCertDelay:  d.NewCertDelay,
		newCertMature: d.NewCertMature,
		newCertWindow: d.NewCertWindow,
		certForDomain: d.CertForDomain,
	}
}

// Wrap returns f wrapped to apply d's delay behavior to newly issued
// certificates.
func (d *Delayer) Wrap(f GetCertificateFunc) GetCertificateFunc {
	g := newGetter(f, d.toGetterConfig())
	return g.get
}

// Wrap returns f wrapped to apply the default delay behavior to newly issued
// certificates. It is a shorthand for (&Delayer{}).Wrap(f).
func Wrap(f GetCertificateFunc) GetCertificateFunc {
	return (&Delayer{}).Wrap(f)
}
