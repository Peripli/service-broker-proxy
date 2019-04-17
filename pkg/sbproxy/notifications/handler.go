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

var lastNotificationGone = errors.New("Last notification revision no longer present in SM")

type Handler struct {
	ctx                      context.Context
	smSettings               *sm.Settings
	lastNotificationRevision int64
	conn                     *websocket.Conn
	pingPeriod               time.Duration
	pongTimeout              time.Duration
}

func NewHandler(ctx context.Context, options *sm.Settings) (*Handler, error) {
	return &Handler{
		ctx:        ctx,
		smSettings: options,
	}, nil
}

func (h *Handler) Start(fullResyncChan chan struct{}, notificationsChan chan *types.Notification) {
	for {
		var err error
		h.conn, err = h.dial()
		if err != nil {
			if err == lastNotificationGone {
				fullResyncChan <- struct{}{}
				continue
			}
			select {
			case <-h.ctx.Done():
				log.C(h.ctx).Info("notifications: stopping connect attempts")
				return
			case <-time.After(time.Second * 5):
				continue
			}
		}
		h.conn.SetReadDeadline(time.Now().Add(h.pongTimeout))
		h.conn.SetPongHandler(func(string) error {
			h.conn.SetReadDeadline(time.Now().Add(h.pongTimeout))
			return nil
		})
		childContext, stopChildren := context.WithCancel(h.ctx)
		done := make(chan bool, 1)
		go wsReader(childContext, h.conn, h.pongTimeout, notificationsChan, done)
		go wsWriter(childContext, h.conn, h.pingPeriod, done)

		<-done // wait for at least one child goroutine (reader/writer) to exit
		stopChildren()
		h.conn.WriteMessage(websocket.CloseMessage, []byte{})
		h.conn.Close()
		if h.ctx.Err() != nil {
			log.C(h.ctx).Info("Context cancelled. Terminating notifications handler")
			return
		}
	}
}

func wsReader(ctx context.Context, conn *websocket.Conn, pongTimeout time.Duration, notificationsChan chan *types.Notification, done chan<- bool) {
	defer close(done)
	for {
		if ctx.Err() != nil {
			return
		}
		_, bytes, err := conn.ReadMessage()
		if err != nil {
			log.C(ctx).WithError(err).Info("Error reading from web socket")
			return
		}
		conn.SetReadDeadline(time.Now().Add(pongTimeout))
		var notification types.Notification
		if err := json.Unmarshal(bytes, &notification); err != nil {
			log.C(ctx).WithError(err).Error("Could not unmarshal WS message into a notification")
			return
		}
		notificationsChan <- &notification
	}
}

func wsWriter(ctx context.Context, conn *websocket.Conn, pingPeriod time.Duration, done chan<- bool) {
	defer close(done)
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err := conn.WriteMessage(websocket.PingMessage, []byte{})
			if err != nil {
				return
			}
		}
	}
}

func (h *Handler) dial() (*websocket.Conn, error) {
	headers := http.Header{}
	auth := "Basic " + base64.StdEncoding.EncodeToString([]byte(h.smSettings.User+":"+h.smSettings.Password))
	headers.Add("Authorization", auth)
	url, err := url.Parse(h.smSettings.URL)
	if err != nil {
		return nil, err
	}
	url.Path = path.Join(url.Path, h.smSettings.NotificationsAPIPath)
	if h.lastNotificationRevision > 0 {
		url.Query().Set("last_notification_revision", strconv.FormatInt(h.lastNotificationRevision, 10))
	}
	dialer := &websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: h.smSettings.RequestTimeout,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: h.smSettings.SkipSSLValidation,
		},
	}
	conn, resp, err := dialer.DialContext(h.ctx, url.String(), headers)
	if err != nil {
		log.C(h.ctx).Errorf("notifications: could not connect: %v %s", err, resp.Status)
		if resp != nil && resp.StatusCode == http.StatusGone {
			h.lastNotificationRevision = 0
			return nil, lastNotificationGone
		}
		return nil, err
	}

	revision, err := strconv.ParseInt(resp.Header.Get("last_notification_revision"), 10, 64)
	if err != nil {
		return nil, err
	}
	h.lastNotificationRevision = revision

	maxPingPeriod, err := time.ParseDuration(resp.Header.Get("max_ping_period"))
	if err != nil {
		return nil, err
	}
	h.pingPeriod = (maxPingPeriod * 2) / 3
	h.pongTimeout = (h.pingPeriod * 11) / 10 // should be longer than pingPeriod
	return conn, nil
}
