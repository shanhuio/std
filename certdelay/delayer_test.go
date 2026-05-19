package certdelay

import (
	"crypto/tls"
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
