package sm

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Peripli/service-manager/pkg/log"
)

type HttpClient interface {
	Do(*http.Request) (*http.Response, error)
}

type RetryableClient struct {
	Client             HttpClient
	MaxRetryCount      int
	TimeBetweenRetries time.Duration
}

func (rc *RetryableClient) Do(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	currentTry := 1
	for {
		if rc.MaxRetryCount < currentTry {
			return nil, fmt.Errorf("Max %d retry count reached", rc.MaxRetryCount)
		}
		log.C(ctx).Infof("Try request to %s for %d time", req.URL.String(), currentTry)
		res, err := rc.Client.Do(req)
		if err != nil {
			log.C(ctx).Errorf("Request to %s failed with: %s", req.URL.String(), err)
		}
		if res.StatusCode < 200 || res.StatusCode > 299 {
			log.C(ctx).Errorf("Request to %s failed with status: %d", req.URL.String(), res.StatusCode)
		} else {
			return res, err
		}
		currentTry++
		log.C(ctx).Infof("Will retry request in %s", rc.TimeBetweenRetries)
		wait := time.After(rc.TimeBetweenRetries)
		select {
		case <-wait:
			continue
		case <-ctx.Done():
			return nil, fmt.Errorf("request cancelled: %s", ctx.Err().Error())
		}
	}
}

type RetryableTransport struct {
	Transport          http.RoundTripper
	MaxRetryCount      int
	TimeBetweenRetries time.Duration
}

func (rt *RetryableTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	currentTry := 1
	for {
		if rt.MaxRetryCount < currentTry {
			return nil, fmt.Errorf("Max %d retry count reached", rt.MaxRetryCount)
		}
		log.C(ctx).Infof("Try request to %s for %d time", req.URL.String(), currentTry)
		res, err := rt.Transport.RoundTrip(req)
		if err != nil {
			log.C(ctx).Errorf("Request to %s failed with: %s", req.URL.String(), err)
		}
		if res.StatusCode < 200 || res.StatusCode > 299 {
			log.C(ctx).Errorf("Request to %s failed with status: %d", req.URL.String(), res.StatusCode)
		} else {
			return res, err
		}
		currentTry++
		log.C(ctx).Infof("Will retry request in %s", rt.TimeBetweenRetries)
		wait := time.After(rt.TimeBetweenRetries)
		select {
		case <-wait:
			continue
		case <-ctx.Done():
			return nil, fmt.Errorf("request cancelled: %s", ctx.Err().Error())
		}
	}
}
