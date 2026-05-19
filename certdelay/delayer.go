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
	// on SCT timestamp checks. Zero means the package default.
	NewCertDelay time.Duration

	// NewCertMature is how long after a cert is first seen before it stops
	// being treated as new; once the cert is "mature", subsequent requests
	// skip the delay. Zero means the package default.
	NewCertMature time.Duration

	// CertForDomain, when non-nil, is consulted first for each request;
	// when it returns nil, the request falls through to the wrapped
	// HelloCertFunc.
	CertForDomain func(domain string) *tls.Certificate
}

func (d *Delayer) toGetterConfig() *getterConfig {
	return &getterConfig{
		newCertDelay:  d.NewCertDelay,
		newCertMature: d.NewCertMature,
		certForDomain: d.CertForDomain,
	}
}

// Wrap wraps f with a getter configured from d. A zero NewCertDelay or
// NewCertMature falls back to the package default.
func (d *Delayer) Wrap(f HelloCertFunc) HelloCertFunc {
	g := newGetter(f, d.toGetterConfig())
	return g.get
}
