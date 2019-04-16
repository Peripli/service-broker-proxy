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
	"errors"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"time"

	"github.com/Peripli/service-broker-proxy/pkg/sm"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/types"
	n "github.com/dotchev/go-json-bench"
	"github.com/gorilla/websocket"
)

var lastNotificationGone = errors.New("Last notification revision no longer present in SM")

type Handler struct {
	ctx                      context.Context
	cancelFunc               func()
	smSettings               *sm.Settings
	lastNotificationRevision int64
	conn *websocket.Conn
}

func NewHandler(ctx context.Context, options *sm.Settings) (*Handler, error) {
	ctx, cancelFunc := context.WithCancel(ctx)
	return &Handler{
		ctx:        ctx,
		cancelFunc: cancelFunc,
		smSettings: options,
	}, nil
}

func (h *Handler) Stop() {
	h.cancelFunc()
	if h.conn != nil {
		h.conn.WriteMessage(websocket.CloseMessage, []byte{})
		h.conn.Close()
	}
}

func (h *Handler) Start(fullResyncChan chan bool, notificationsChan chan *types.Notification) {
	for {
		h.conn, err := h.dial()
		if err != nil {
			if err == lastNotificationGone {
				fullResyncChan <- true
				continue
			}
			select {
			case <-h.ctx.Done():
				log.C(n.ctx).Info("notifications: stopping connect attempts")
				return
			case <-time.After(time.Second * 5):
				continue
			}
		}
		for {
			select {
			case <-h.ctx.Done():
				log.C(h.ctx).Info("Context cancelled. Terminating notifications handler")
				conn.Close()
				return
			default:
				// TODO move to separate goroutine

				messageType, bytes, err := conn.ReadMessage()
				if err != nil {
					if h.ctx.Err() != nil {
						log.C(h.ctx).WithError(err).Info("notifications: not reconnecting")
						return
					} else {
						conn.Close()
						break
					}
				}
				if messageType == websocket.PongMessage {
					// todo: reset ping timer and set smAlive = true
				}
				notificationsChan <- notification
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
	revisionHeaderValue := resp.Header.Get("last_notification_revision")
	revision, err := strconv.Atoi(revisionHeaderValue)
	if err != nil {
		return nil, err
	}
	h.lastNotificationRevision = revision
	return conn, nil
}
