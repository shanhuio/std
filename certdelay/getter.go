package certdelay

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"strings"
	"sync"
	"time"
)

func nowFunc(f func() time.Time) func() time.Time {
	if f != nil {
		return f
	}
	return time.Now
}

type timeEntry struct {
	mature time.Time // After this time, no delay will be applied.
	expire time.Time // After this time, will be cleaned up.
}

// HelloCertFunc is the function that gets TLS certificate based on the
// HelloInfo.
type HelloCertFunc func(hello *tls.ClientHelloInfo) (*tls.Certificate, error)

type getter struct {
	getFunc HelloCertFunc
	now     func() time.Time
	sleep   func(d time.Duration)

	newCertDelay  time.Duration
	newCertMature time.Duration
	newCertWindow time.Duration

	mu            sync.Mutex
	certs         map[string]*timeEntry
	certForDomain func(domain string) *tls.Certificate

	cleanUpTimer *gapper
}

type getterConfig struct {
	certForDomain func(domain string) *tls.Certificate
	now           func() time.Time
	sleep         func(d time.Duration)
	newCertDelay  time.Duration
	newCertMature time.Duration
	newCertWindow time.Duration
}

func newGetter(f HelloCertFunc, config *getterConfig) *getter {
	now := nowFunc(config.now)
	sleep := config.sleep
	if sleep == nil {
		sleep = time.Sleep
	}

	delay := config.newCertDelay
	if delay == 0 {
		delay = DefaultNewCertDelay
	}
	mature := config.newCertMature
	if mature == 0 {
		mature = DefaultNewCertMature
	}
	window := config.newCertWindow
	if window == 0 {
		window = DefaultNewCertWindow
	}

	const cleanUpPeriod = time.Hour

	return &getter{
		getFunc: f,
		now:     now,
		sleep:   sleep,

		newCertDelay:  delay,
		newCertMature: mature,
		newCertWindow: window,

		certs:         make(map[string]*timeEntry),
		certForDomain: config.certForDomain,

		cleanUpTimer: newGapperNow(cleanUpPeriod, now()),
	}
}

func (g *getter) checkCleanUp() {
	now := g.now()
	if g.cleanUpTimer.check(now) {
		go g.cleanUp()
	}
}

func (g *getter) cleanUp() {
	now := g.now()

	g.mu.Lock()
	defer g.mu.Unlock()

	var toDelete []string
	for k, v := range g.certs {
		if now.After(v.expire) {
			toDelete = append(toDelete, k)
		}
	}
	for _, k := range toDelete {
		delete(g.certs, k)
	}
}

// DefaultNewCertDelay is the default delay applied to the return of a newly
// issued certificate.
const DefaultNewCertDelay = 2 * time.Second

// DefaultNewCertMature is the default age after which no more delaying is
// applied to a newly issued certificate.
const DefaultNewCertMature = 3 * time.Second

// DefaultNewCertWindow is the default window within which a certificate's
// NotBefore is considered recent enough to warrant the delay treatment.
const DefaultNewCertWindow = 2 * time.Hour

// trackCert records the cert under its serial-number key if not already
// known, and reports whether the request should be delayed.
func (g *getter) trackCert(cert *x509.Certificate, now time.Time) bool {
	// We use the SerialNumber as the key here. This assumes that all the
	// certificates are issued by the same issuer, and the issuer uses
	// unique serial numbers for certificates.
	k := fmt.Sprintf("%x", cert.SerialNumber)

	g.mu.Lock()
	defer g.mu.Unlock()

	entry, ok := g.certs[k]
	if !ok {
		g.certs[k] = &timeEntry{
			mature: now.Add(g.newCertMature),
			expire: cert.NotAfter,
		}
		return true
	}
	return now.Before(entry.mature)
}

func (g *getter) delayUnlessMature(cert *x509.Certificate, now time.Time) {
	needDelay := g.trackCert(cert, now)
	if needDelay {
		g.sleep(g.newCertDelay)
	}
}

func (g *getter) maybeDelay(cert *x509.Certificate) {
	now := g.now()
	if cert.NotBefore.Before(now.Add(-g.newCertWindow)) {
		// If the cert's start time is more than newCertWindow ago, then this
		// is not likely a new certificate.
		return
	}

	// Now, we will check the if the certificate is "mature", and delay for
	// some time.
	g.delayUnlessMature(cert, now)
	g.checkCleanUp()
}

func (g *getter) get(hello *tls.ClientHelloInfo) (
	*tls.Certificate, error,
) {
	if g.certForDomain != nil {
		name := strings.TrimSuffix(hello.ServerName, ".")
		if cert := g.certForDomain(name); cert != nil {
			return cert, nil
		}
	}

	cert, err := g.getFunc(hello)
	if err != nil {
		return cert, err
	}
	if cert.Leaf != nil {
		g.maybeDelay(cert.Leaf)
	}
	return cert, nil
}

