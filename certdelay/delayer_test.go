package certdelay

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"math/big"
	"testing"
	"time"
)

func TestDelayerToGetterConfig(t *testing.T) {
	sentinel := new(tls.Certificate)
	withCert := func(string) *tls.Certificate { return sentinel }

	for _, c := range []struct {
		name        string
		d           Delayer
		wantCertHit bool
	}{
		{name: "zero", d: Delayer{}},
		{
			name: "durations",
			d: Delayer{
				NewCertDelay:  5 * time.Second,
				NewCertMature: 7 * time.Second,
				NewCertWindow: 4 * time.Hour,
			},
		},
		{
			name:        "with-certForDomain",
			d:           Delayer{CertForDomain: withCert},
			wantCertHit: true,
		},
		{
			name: "all-fields",
			d: Delayer{
				NewCertDelay:  5 * time.Second,
				NewCertMature: 7 * time.Second,
				NewCertWindow: 4 * time.Hour,
				CertForDomain: withCert,
			},
			wantCertHit: true,
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := c.d.toGetterConfig()
			if got.newCertDelay != c.d.NewCertDelay {
				t.Errorf(
					"newCertDelay: got %v, want %v",
					got.newCertDelay, c.d.NewCertDelay,
				)
			}
			if got.newCertMature != c.d.NewCertMature {
				t.Errorf(
					"newCertMature: got %v, want %v",
					got.newCertMature, c.d.NewCertMature,
				)
			}
			if got.newCertWindow != c.d.NewCertWindow {
				t.Errorf(
					"newCertWindow: got %v, want %v",
					got.newCertWindow, c.d.NewCertWindow,
				)
			}
			if c.wantCertHit {
				if got.certForDomain == nil {
					t.Errorf("certForDomain: got nil, want non-nil")
				} else if got.certForDomain("anything") != sentinel {
					t.Errorf("certForDomain: did not return the configured cert")
				}
			} else if got.certForDomain != nil {
				t.Errorf("certForDomain: got non-nil, want nil")
			}
			if got.now != nil {
				t.Errorf("now: got non-nil, want nil")
			}
			if got.sleep != nil {
				t.Errorf("sleep: got non-nil, want nil")
			}
		})
	}
}

func TestWrap(t *testing.T) {
	sentinel := new(tls.Certificate)
	// A cert whose NotBefore is well before now sits outside the default
	// window, so Wrap's default Delayer passes it straight through without
	// the (real, 2s) delay -- keeping the test fast.
	sentinel.Leaf = &x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now().Add(-24 * time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}

	called := false
	f := func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
		called = true
		return sentinel, nil
	}

	wrapped := Wrap(f)
	hello := &tls.ClientHelloInfo{ServerName: "example.com"}
	cert, err := wrapped(hello)
	if err != nil {
		t.Fatal("wrapped get: ", err)
	}
	if !called {
		t.Error("wrapped function was not called")
	}
	if cert != sentinel {
		t.Errorf("got cert %p, want %p", cert, sentinel)
	}
}

func TestWrapPropagatesError(t *testing.T) {
	wantErr := errors.New("boom")
	f := func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
		return nil, wantErr
	}

	wrapped := Wrap(f)
	hello := &tls.ClientHelloInfo{ServerName: "example.com"}
	if _, err := wrapped(hello); !errors.Is(err, wantErr) {
		t.Errorf("got error %v, want %v", err, wantErr)
	}
}
