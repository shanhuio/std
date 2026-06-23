package fakeweb

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"reflect"
	"sync"
	"testing"
)

func TestTLSConfigs(t *testing.T) {
	var wg sync.WaitGroup
	defer wg.Wait()

	domains := []string{"shanhu.io", "www.shanhu.io"}
	tlsConfigs, err := newTLSConfigs(domains)
	if err != nil {
		t.Fatal("make config:", err)
	}

	if !reflect.DeepEqual(tlsConfigs.Domains, domains) {
		t.Errorf("domains got %v, want %v", tlsConfigs.Domains, domains)
	}

	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal("listen: ", err)
	}
	defer lis.Close()

	tlsLis := tls.NewListener(lis, tlsConfigs.Server)

	serve := func() error {
		conn, err := tlsLis.Accept()
		if err != nil {
			return fmt.Errorf("accept: %w", err)
		}
		defer conn.Close()

		buf := make([]byte, 16)

		if _, err := io.ReadFull(conn, buf); err != nil {
			return fmt.Errorf("read: %w", err)
		}

		// echo back
		if _, err := conn.Write(buf); err != nil {
			return fmt.Errorf("write: %w", err)
		}

		if err := conn.Close(); err != nil {
			return fmt.Errorf("close: %w", err)
		}
		return nil
	}

	serveErrChan := make(chan error, 1)

	wg.Go(func() {
		serveErrChan <- serve()
	})

	tlsConfigs.Client.ServerName = "www.shanhu.io"
	conn, err := tls.Dial("tcp", lis.Addr().String(), tlsConfigs.Client)
	if err != nil {
		t.Fatal("dial server: ", err)
	}

	req := []byte("0123456789abcdef")
	if _, err := conn.Write(req); err != nil {
		t.Fatal("client write: ", err)
	}

	resp := make([]byte, 16)
	if _, err := io.ReadFull(conn, resp); err != nil {
		t.Fatal("client read: ", err)
	}

	if !bytes.Equal(resp, req) {
		t.Errorf("got response %q, want %q", resp, req)
	}

	if err := conn.Close(); err != nil {
		t.Fatal("client close: ", err)
	}

	if err := <-serveErrChan; err != nil {
		t.Error("serve error: ", err)
	}
}
