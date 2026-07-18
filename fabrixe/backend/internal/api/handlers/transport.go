package handlers

import (
	"crypto/tls"
	"net/http"
	"time"
)

// newInsecureTransport returns an HTTP transport that skips TLS verification.
// Used only for peer-to-peer node pinging where certs are self-signed.
func newInsecureTransport() http.RoundTripper {
	return &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, //nolint:gosec // intentional: peer nodes use self-signed certs
		},
		ResponseHeaderTimeout: 5 * time.Second,
		DisableKeepAlives:     true,
	}
}
