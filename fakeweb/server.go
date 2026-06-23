package fakeweb

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
)

// Server wraps a server that serves a self-signed TLS certificate.
type Server struct {
	listener net.Listener
	server   *http.Server
	client   *http.Client

	serveWait sync.WaitGroup

	tlsConfigs *tlsConfigs
}

// NewServer starts an HTTPS server that serves handler at the given domain.
func NewServer(domain string, handler http.Handler) (*Server, error) {
	tlsConfigs, err := newTLSConfigs([]string{domain})
	if err != nil {
		return nil, fmt.Errorf("make TLS configs: %w", err)
	}

	server := &http.Server{
		Handler:   handler,
		TLSConfig: tlsConfigs.Server,
	}

	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return nil, fmt.Errorf("listen on localhost: %w", err)
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfigs.Client,
			DialContext:     SinkDialFunc(lis.Addr().String()),
		},
	}

	s := &Server{
		listener:   lis,
		server:     server,
		client:     client,
		tlsConfigs: tlsConfigs,
	}

	s.serveWait.Add(1)
	go func() {
		defer s.serveWait.Done()
		if err := server.ServeTLS(lis, "", ""); err != nil {
			if err != http.ErrServerClosed {
				log.Printf("serve tls got error: %s", err)
			}
		}
	}()

	return s, nil
}

// Addr returns the address that the server listens at.
func (s *Server) Addr() string { return s.listener.Addr().String() }

// Client returns a client that always dials to the server. The client
// also trusts the TLS certificate of the server.
func (s *Server) Client() *http.Client { return s.client }

// ClientTLSConfig returns the TLS config that can be used for the client.
func (s *Server) ClientTLSConfig() *tls.Config {
	return s.tlsConfigs.Client.Clone()
}

// Close closes the server.
func (s *Server) Close() error {
	err := s.server.Close()
	s.serveWait.Wait()
	return err
}
