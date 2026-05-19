package certdelay

import (
	"crypto/tls"
	"crypto/x509"
	"math/big"
	"testing"
	"time"
)

func TestGetter(t *testing.T) {
	now := time.Now()
	sleep := func(d time.Duration) {
		now = now.Add(d)
	}
	readNow := func() time.Time {
		return now
	}

	config := &getterConfig{
		now:   readNow,
		sleep: sleep,
	}

	f := func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
		// TODO(h8liu): use a better certificate
		cert := new(tls.Certificate)
		cert.Leaf = &x509.Certificate{
			SerialNumber: big.NewInt(12345),
			NotBefore:    now,
			NotAfter:     now.Add(time.Hour),
		}
		return cert, nil
	}

	start := now
	g := newGetter(f, config)
	hello := &tls.ClientHelloInfo{ServerName: "example.com"}
	if _, err := g.get(hello); err != nil {
		t.Fatal("get certificate: ", err)
	}

	atLeast := start.Add(defaultNewCertDelay)
	if now.Before(atLeast) {
		t.Errorf(
			"should have delayed to +%s, but at +%s",
			defaultNewCertDelay, now.Sub(start),
		)
	}

	// get again
	atLeast = now.Add(defaultNewCertDelay)
	before := now
	if _, err := g.get(hello); err != nil {
		t.Fatal("get certificate: ", err)
	}
	if now.Before(atLeast) {
		t.Errorf(
			"should have delayed to +%s, but at +%s",
			defaultNewCertDelay, now.Sub(before),
		)
	}

	mature := start.Add(defaultNewCertMature)
	if now.Before(mature) {
		now = mature
	}

	// get again, this time after it is matured.
	before = now
	if _, err := g.get(hello); err != nil {
		t.Fatal("get certificate: ", err)
	}
	if now.After(before) {
		t.Errorf("time moved by %s on second get", now.Sub(before))
	}
}
