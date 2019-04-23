/*
 * Copyright 2018 The Service Manager Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package notifications

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"time"

	"github.com/Peripli/service-broker-proxy/pkg/sm"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/gorilla/websocket"
)

var errLastNotificationGone = errors.New("last notification revision no longer present in SM")

// Producer produces messages on the notifications queue
type Producer struct {
	lastNotificationRevision int64
	conn                     *websocket.Conn
	pingPeriod               time.Duration
	pongTimeout              time.Duration
	url                      *url.URL
	user                     string
	password                 string
	skipSSLValidation        bool
	requestTimeout           time.Duration
	minPingPeriod            time.Duration
	reconnectDelay           time.Duration
}

// ProducerSettings are the settings for the producer
type ProducerSettings struct {
	*sm.Settings
	MinPingPeriod  time.Duration
	ReconnectDelay time.Duration
}

// DefaultProducerSettings are the default settings for the producer
func DefaultProducerSettings(settings *sm.Settings) *ProducerSettings {
	return &ProducerSettings{
		Settings:       settings,
		MinPingPeriod:  time.Second,
		ReconnectDelay: 3 * time.Second,
	}
}

// NewProducer returns a configured producer for the given settings
func NewProducer(settings *ProducerSettings) (*Producer, error) {
	notificationsURL, err := buildNotificationsURL(settings.URL, settings.NotificationsAPIPath)
	if err != nil {
		return nil, err
	}
	return &Producer{
		url:               notificationsURL,
		user:              settings.User,
		password:          settings.Password,
		skipSSLValidation: settings.SkipSSLValidation,
		requestTimeout:    settings.RequestTimeout,
		minPingPeriod:     settings.MinPingPeriod,
		reconnectDelay:    settings.ReconnectDelay,
	}, nil
}

func buildNotificationsURL(baseURL, notificationsPath string) (*url.URL, error) {
	smURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	switch smURL.Scheme {
	case "http":
		smURL.Scheme = "ws"
	case "https":
		smURL.Scheme = "wss"
	}
	smURL.Path = path.Join(smURL.Path, notificationsPath)
	return smURL, nil
}

// Start starts the producer in a new go-routine
func (h *Producer) Start(ctx context.Context, resyncChan chan struct{}, notificationsQueue Queue) {
	go h.run(ctx, resyncChan, notificationsQueue)
}

func (h *Producer) run(ctx context.Context, resyncChan chan struct{}, notificationsQueue Queue) {
	for {
		needResync := h.lastNotificationRevision == 0
		var err error
		h.conn, err = h.dial(ctx)
		if err == nil {
			done := make(chan struct{}, 1)
			h.conn.SetReadDeadline(time.Now().Add(h.pingPeriod + h.pongTimeout))
			h.conn.SetPongHandler(func(string) error {
				log.C(ctx).Debug("Received pong")
				h.conn.SetReadDeadline(time.Now().Add(h.pongTimeout))
				return nil
			})
			childContext, stopChildren := context.WithCancel(ctx)
			go h.readNotifications(childContext, notificationsQueue, done)
			go h.ping(childContext, done)

			if needResync {
				resyncChan <- struct{}{}
			}

			<-done // wait for at least one child goroutine (reader/writer) to exit
			stopChildren()
			log.C(ctx).Debug("Closing websocket")
			h.conn.WriteControl(websocket.CloseMessage, []byte{}, time.Now().Add(h.requestTimeout))
			h.conn.Close()
		} else {
			log.C(ctx).WithError(err).Error("could not connect websocket")
			if err == errLastNotificationGone { // skip reconnect delay
				h.lastNotificationRevision = 0
				continue
			}
		}
		select {
		case <-ctx.Done():
			log.C(ctx).Info("Context cancelled. Terminating notifications handler")
			return
		case <-time.After(h.reconnectDelay):
			log.C(ctx).Debug("Attempting to reestablish websocket connection")
		}
	}
}

func (h *Producer) readNotifications(ctx context.Context, notificationsQueue Queue, done chan<- struct{}) {
	defer func() {
		log.C(ctx).Debug("Exiting notification reader")
		done <- struct{}{}
	}()
	for {
		if ctx.Err() != nil {
			return
		}
		_, bytes, err := h.conn.ReadMessage()
		if err != nil {
			log.C(ctx).WithError(err).Error("Error reading from web socket")
			return
		}
		h.conn.SetReadDeadline(time.Now().Add(h.pongTimeout))
		var notification types.Notification
		if err := json.Unmarshal(bytes, &notification); err != nil {
			log.C(ctx).WithError(err).Error("Could not unmarshal WS message into a notification")
			return
		}
		log.C(ctx).Debugf("Received notification with revision %d", notification.Revision)
		notificationsQueue.Push(&notification)
		h.lastNotificationRevision = notification.Revision
	}
}

func (h *Producer) ping(ctx context.Context, done chan<- struct{}) {
	defer func() {
		log.C(ctx).Debug("Exiting pinger")
		done <- struct{}{}
	}()
	ticker := time.NewTicker(h.pingPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			log.C(ctx).Debug("Sending ping")
			if err := h.conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				log.C(ctx).WithError(err).Error("Could not write message on the websocket")
				return
			}
		}
	}
}

func (h *Producer) dial(ctx context.Context) (*websocket.Conn, error) {
	headers := http.Header{}
	auth := "Basic " + base64.StdEncoding.EncodeToString([]byte(h.user+":"+h.password))
	headers.Add("Authorization", auth)

	connectURL := *h.url
	if h.lastNotificationRevision > 0 {
		q := connectURL.Query()
		q.Set("last_notification_revision", strconv.FormatInt(h.lastNotificationRevision, 10))
		connectURL.RawQuery = q.Encode()
	}
	dialer := &websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: h.requestTimeout,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: h.skipSSLValidation,
		},
	}
	log.C(ctx).Debugf("Connecting to %s ...", &connectURL)
	conn, resp, err := dialer.DialContext(ctx, connectURL.String(), headers)
	if err != nil {
		if resp == nil {
			log.C(ctx).WithError(err).Errorf("Could not connect to %s", &connectURL)
		} else {
			log.C(ctx).WithError(err).Errorf("Could not connect to %s: status: %d", &connectURL, resp.StatusCode)
			if resp.StatusCode == http.StatusGone {
				return nil, errLastNotificationGone
			}
		}
		return nil, err
	}

	// TODO: define constants for these headers in service-manager
	if h.lastNotificationRevision == 0 {
		revision, err := strconv.ParseInt(resp.Header.Get("last_notification_revision"), 10, 64)
		if err != nil {
			return nil, err
		}
		if revision <= 0 {
			return nil, fmt.Errorf("invalid last notification revision received (%d)", revision)
		}
		h.lastNotificationRevision = revision
	}

	maxPingPeriod, err := time.ParseDuration(resp.Header.Get("max_ping_period"))
	if err != nil {
		return nil, err
	}
	if maxPingPeriod < h.minPingPeriod {
		return nil, fmt.Errorf("invalid max ping period (%s) must be greater than the minimum ping period (%s)", maxPingPeriod, h.minPingPeriod)
	}
	h.pingPeriod = (maxPingPeriod * 2) / 3
	h.pongTimeout = (h.pingPeriod * 13) / 10 // should be longer than pingPeriod
	log.C(ctx).Debugf("Ping period: %s pong timeout: %s", h.pingPeriod, h.pongTimeout)
	return conn, nil
}
