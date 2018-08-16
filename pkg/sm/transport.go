package sm

import (
	"net/http"
	"crypto/tls"
)

// BasicAuthTransport implements http.RoundTripper interface and intercepts that request that is being sent,
// adding basic authorization and delegates back to the original transport.
type Transport struct {
	Username string
	Password string

	Rt http.RoundTripper
}

// RoundTrip implements http.RoundTrip and adds basic authorization header before delegating to the
// underlying RoundTripper
func (b Transport) RoundTrip(request *http.Request) (*http.Response, error) {
	if b.Username != "" && b.Password != "" {
		request.SetBasicAuth(b.Username, b.Password)
	}

	return b.Rt.RoundTrip(request)
}

type SkipSSLTransport struct {
	SkipSslValidation bool
}

// RoundTrip implements http.RoundTrip and adds basic authorization header before delegating to the
// underlying RoundTripper
func (b SkipSSLTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	defaultTransport := http.DefaultTransport.(*http.Transport)
	t := &http.Transport{
		Proxy:                 defaultTransport.Proxy,
		TLSHandshakeTimeout:   defaultTransport.TLSHandshakeTimeout,
		ExpectContinueTimeout: defaultTransport.ExpectContinueTimeout,
	}

	t.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: b.SkipSslValidation,
	}
	return t.RoundTrip(request)
}