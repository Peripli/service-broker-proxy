package notifications_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	smnotifications "github.com/Peripli/service-manager/api/notifications"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/sirupsen/logrus"

	"github.com/Peripli/service-broker-proxy/pkg/sbproxy/notifications"
	"github.com/Peripli/service-broker-proxy/pkg/sm"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/gorilla/websocket"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var invalidRevisionStr = strconv.FormatInt(types.InvalidRevision, 10)
var (
	ClientCertificate = `-----BEGIN CERTIFICATE-----
MIICrjCCAZYCCQDfb7elrdbtvDANBgkqhkiG9w0BAQUFADAYMRYwFAYDVQQDDA1G
aXJzdCBNLiBMYXN0MCAXDTIwMDMyNDE5NTE0MloYDzMwMDAwNTI2MTk1MTQyWjAY
MRYwFAYDVQQDDA1GaXJzdCBNLiBMYXN0MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8A
MIIBCgKCAQEAyJ3Hr9hJits2ssZZu4ZaZrKxDWHffYPRQNDlNJg8uldXtizjxiyo
0bS+LmiejmyIPRa7BE38ctqoFKPcqJoHy63imWr5plJDbHpacdi1s9FhSsLFCJPe
QK4+GSDu1pOWiqWETomRVz+q0NbYaURT3qZ/YW3cCrRns/+CITz+J3bWHcVBARaq
A+WS/K3dtknTb4On0AeD4SVT5UI19alF3xooh7HG4cp+5j8JtX8CazIrdjFfEUW2
k17a3FiQbyRPYFa52of+59kFd08ABPHwmSeRdUzyhYgn+fYUZ9qUJ9twwgDi50mJ
LDyyp+bGg3exrSy1QQE4rqXuQuPvBBKOiQIDAQABMA0GCSqGSIb3DQEBBQUAA4IB
AQDCiBUdE4chHdUYK2FPrkk72GQiEXTDUP4OZpkJ6mLNCj25j/UC4ND8ByfnsYbx
4rF1Q4JslZZg80NXDSJMR43kfAt+7hnKoYHceCD2FsSlMxP+aOiaa7FbKUSqYnU1
+GGdK13EygiMr1RkVhG8GRngUvE5YzDwcm1jl4pOsShHaX26pititxwKCJSABA3G
1nfrfyHb9N30cN0Z6u4eYpMidO5Txs8Fl0Jh/xJOperl/3Z2ubWKPd1eelhS8GXx
ox8VI+BTty9Yl0fRl12cRsSF1ddOKBJxcdp3Ae2AZZ762vm/ESr1TzrX+wYgLUxp
1I6ycxMDnvvDWmGO0V4HDkPR
-----END CERTIFICATE-----`

	ClientKey = `-----BEGIN PRIVATE KEY-----
MIIEvAIBADANBgkqhkiG9w0BAQEFAASCBKYwggSiAgEAAoIBAQDIncev2EmK2zay
xlm7hlpmsrENYd99g9FA0OU0mDy6V1e2LOPGLKjRtL4uaJ6ObIg9FrsETfxy2qgU
o9yomgfLreKZavmmUkNselpx2LWz0WFKwsUIk95Arj4ZIO7Wk5aKpYROiZFXP6rQ
1thpRFPepn9hbdwKtGez/4IhPP4ndtYdxUEBFqoD5ZL8rd22SdNvg6fQB4PhJVPl
QjX1qUXfGiiHscbhyn7mPwm1fwJrMit2MV8RRbaTXtrcWJBvJE9gVrnah/7n2QV3
TwAE8fCZJ5F1TPKFiCf59hRn2pQn23DCAOLnSYksPLKn5saDd7GtLLVBATiupe5C
4+8EEo6JAgMBAAECggEAJtFo1xyptkWOgu8gY8mualq/KZC7luTPs5P4FcIzVfca
kLSE6k6v58vqVL6Hl5VmkzN3wnB4nZyzkzLVuoX7ZiziQL9TSRx30WCnaYn+NqoY
AkhHqc463hcZCvG1ZS2vnmpCfJPf3JsEKV65Bz1iYR2kXizMvAGGY2zYOCg+IVJk
IJg7TcbrdgA03/IqE2FoPHXuAlAhm5312o+N1h4wXnWRrzUDR9FMOiQCKAUtMuIU
iHMJ7naQjwjJKKsOeDeF1bzgSR4WmZZk5w8fCDdgHFpSn86DZBSfOu6t7kzhOvwR
pu6dxl1t7FPXWtu2jr+/WP0ZtoP2XeKdUMdSlZZSgQKBgQD6Kc5M4TIgjpAOrYMH
1GSD8ccr9KJIX+qy7iDNpzEb38g1Uvsx2OvYLkT1vDhmerRYbai3v+ZU2LBIASzy
gNvbeIRarvEmdoMB7iSTw4D0JJdm+JSf5n3BpBZ+b3IqGNQeBJAmoEv2aCvQOZNe
J+t0FfmgRZDj3paUIZFgTjfj8QKBgQDNTAlJPFUows0Hyd4FWlNhqalLyQWVtVzB
swJPdkyW1hPzZGHnuhDGv2azsraQy2vZUjo8p/zn4mytWng0ya7SzXZlbbdvIU/o
QMraQGIkQDADMwg95Y5R6h0FCxsBBmHaI4YpFtvk+ZvDELv927aXpzgyCj4v/lSc
qSw2ts4MGQKBgG6vcqUXertm+JxV70TWl8a9gleTfP4y2kBjFkaH9DWWFRpq5dPP
W8Kh7kcgCYBmSEdb9aufj8T4vz6Mrpt5ok2ADGenQfG3vA1tled/OB5N1mNsFy6M
qBW2iXFV1BiGNcw2TqWYhSO4QbJ21xpw5T/OvU1JmmsIQG24UH9g/F+xAoGAPtq1
yR9Yr1cc8PKEMD1cY/1O4O4V8KULVh6ZaXy9rDy09QLZ2tmjw0Xcis3/iUtOpMXB
IMsJ6nDvdw/I19ib1tyjECDMVZDsZx5XPQUTRygDyyb3sgOzVC8KXX3t8Z1jnibc
L35ZKrylTM61z95SBBJlaSSrr4P9oc1FxSao5RkCgYBhAh11SGmBntpW02ky8SbW
WUKaIE168cJ4xJmzwAV1OmZbMlc4AhJ9xmkbaViIpIe4Z/R0jp+ZbaFgsgtySER/
ZcJLWYGz00zKINL3Z9FKTa0Opg30AC9kZi9YbJMlsSJMoXx6mTE4kcYHOKv4f1qQ
XL31L67QikBXJnPmtkwxSQ==
-----END PRIVATE KEY-----`

	InvalidClientCertificate = `-----BEGIN CERTIFICATE-----
MIICrDCCAZQCCQCpKp1xS5g91TANBgkqhkiG9w0BAQUFADAYMRYwFAYDVQQDDA1G
aXJzdCBNLiBMYXN0MB4XDTIwMDMyNTE2MjQzNFoXDTIwMDQyNDE2MjQzNFowGDEW
MBQGA1UEAwwNRmlyc3QgTS4gTGFzdDCCASIwDQYJKoZIhvcNAQEBBQADggEPADCC
AQoCggEBAMX+zkYBETE0RtBlbE6OBesnTYDNYA345vGkpVlcJEZwFEr5mMFZ5z2s
dUQuUwLJIStDvxphrUCof99vPr925Ladcq/W9Gl4DkG7NP76AXeykJYC+7mvoZJH
fOSRM6TOA/OkdYk5ZGmQzflSvO33Xyj+MmZ/9uQtkz+ZCOyHxcwPxFFinPNl5fZv
LiN0p+zIvIFScpiAm1/YziaFX7XdJXlhBxMeqUWdDQuGCYGiRYdU7oZB7t2gA2Wa
oXjiOWpkd0ka4/EM/rPPWudwlbIMHd2e+ddwc7XFtN5wBhdo3bB8s3Dc9B8S3Xv1
e+1a/Iq7sQ//5eMh3s69/Sghi5Avu9kCAwEAATANBgkqhkiG9w0BAQUFAAOCAQEA
br2agn+j98e2KANkEM99Xk/oBsSmyp2tkwYr7p4dJUTCa8auLJbjIsdX9vsyprUd
JvoUamO8TKSGMhkjTS6MZJpILqJMWTpBXmDb0lktwJd062QLZklhttvhLv9YJuEG
/VXWgyRrQ4Jpy2+QAc8sWhtF42Vyu7sJZuikyT8FWO3fsILpdIg/SjsO2UGsP8hE
e6sqXUtHY6HXCdwRA4STY3ICZVMjw+NkfJPpC3gL7xbZHzzlwxQXqo7ySal4e6Sl
5ruwMNVt+2KVXdshqGNbRYNGs0x96iWbi8zRFx3uuIwLIDghWQqv+8gKcxttvkjP
LjRFM2n5hd6MzWG9criQfA==
-----END CERTIFICATE-----`
	InvalidClientKey = `-----BEGIN PRIVATE KEY-----
MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQDF/s5GARExNEbQ
ZWxOjgXrJ02AzWAN+ObxpKVZXCRGcBRK+ZjBWec9rHVELlMCySErQ78aYa1AqH/f
bz6/duS2nXKv1vRpeA5BuzT++gF3spCWAvu5r6GSR3zkkTOkzgPzpHWJOWRpkM35
Urzt918o/jJmf/bkLZM/mQjsh8XMD8RRYpzzZeX2by4jdKfsyLyBUnKYgJtf2M4m
hV+13SV5YQcTHqlFnQ0LhgmBokWHVO6GQe7doANlmqF44jlqZHdJGuPxDP6zz1rn
cJWyDB3dnvnXcHO1xbTecAYXaN2wfLNw3PQfEt179XvtWvyKu7EP/+XjId7Ovf0o
IYuQL7vZAgMBAAECggEBAKomTx3ZzOx8AF8Wyfy4EF4FaJVH6UQYol8HHxGsHYBq
0QWdeaivmglmK2Bsbun17os/rPr+9eSa6UkaUNI5WlOU+vohv+jjQ105hFGah6hV
y+sepTTtuev7g1jpb3gxkzPOITPMHn6Z8mhQsgvOifiwep+bWJC+mcwNt52NEG5L
m1jBbDvGDQL3oiSIzMnFd58WZm5lQpTIapcx+lvKtVSn2xb/d0BigCTonZXMUZhL
2y5kcGJSvNaYgrjc0oZDA3gZ4YdoUrMW99CFFeECbbzASp01J+ivIn/Tke1VCHoe
AI4GJFiSFA5jiALkxB1KMIgzaSlg1sGS5VDrlSLu8AECgYEA5qrEtgqTFOuM+Ot7
1QCva5rei8QN8qBczmSB1pZEWtmJNopU1soLOrDdj3sw0l1Up2hSBGiEwcbiTrr+
if3YUZkRRFv5D7BGnQLhe4QcXaANPjsExeQ1NOgaruIVRzaVv4PXhMP0VPErNehl
NKuhVdOI971rbSdG3A2xrELOtqMCgYEA2713xS0/3hepmlO0PHAWsQZjle9zBLPM
SvWqcArrVPRdam4x5yNOnonkGghJ/4G4xWXLHxTNJqshzuGHH9iq8Hk2mQfxMmFO
8l4sTz4d8D6GECGodKov3gDZ6h0BBBR9qv+b5Z3H7jZ2+MD5h9y17g/Kx1NXBnF1
NQNg3Oi0t1MCgYAJayelF0FyNTwIXfUseV6wUh6MLnEzWwDvHIOAs5oO65sCsxtL
uexDdT1Wwnz32f++5i+TJoFlOC29cT07fTX7/vgJhofg8B2yA5AZbweJeyOPSvGi
8vKJOoD8axbbVYs/yq5eKXIslbxh8x9Oy0NHMeAB3aYpStVF3vlGQ2QVaQKBgQDS
eP9ggL/9BaMxK92mSiKh+yGl+o2rwl/6qKZQ3VSdsdZMXDI2V241kpRGjwv5zRHj
GWZeZfk+gYpHc2OPEGRjI2c1WxMfE2+f3K4KVNAuTmTwzJxi6qQgu6X+hTt04f+g
q2ZyoBdhRw/bolMgXDpyRPQQyfXAOSpv1cWQsuBt+wKBgAmGjk+97Y+7eFiHQqKd
AB3SPFZoHO6s7UKDU2Uv7u3NAfQ+/FydwgtfDEcD9aISVZ9DVZJy4/kOSUIc45rG
+XjDdXNcJIN2fNiPMaAimisjxPC83UxrirAJISnHrJwqaCfUwDSArz+Lvf+WlZCL
AxcGuEEhttwyFSfydmYdBaQx
-----END PRIVATE KEY-----`
)

func TestNotifications(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Notifications Suite")
}

var _ = Describe("Notifications", func() {
	Describe("Producer Settings", func() {
		var settings *notifications.ProducerSettings
		BeforeEach(func() {
			settings = notifications.DefaultProducerSettings()
		})
		It("Default settings are valid", func() {
			err := settings.Validate()
			Expect(err).ToNot(HaveOccurred())
		})

		Context("When MinPingPeriod is invalid", func() {
			It("Validate returns error", func() {
				settings.MinPingPeriod = 0
				err := settings.Validate()
				Expect(err).To(HaveOccurred())
			})
		})

		Context("When ReconnectDelay is invalid", func() {
			It("Validate returns error", func() {
				settings.ReconnectDelay = -1 * time.Second
				err := settings.Validate()
				Expect(err).To(HaveOccurred())
			})
		})
		Context("When PongTimeout is invalid", func() {
			It("Validate returns error", func() {
				settings.PongTimeout = 0
				err := settings.Validate()
				Expect(err).To(HaveOccurred())
			})
		})
		Context("When PingPeriodPercentage is invalid", func() {
			It("Validate returns error", func() {
				settings.PingPeriodPercentage = 0
				err := settings.Validate()
				Expect(err).To(HaveOccurred())

				settings.PingPeriodPercentage = 100
				err = settings.Validate()
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when resync period is missing", func() {
			It("returns an error", func() {
				settings.ResyncPeriod = 0
				err := settings.Validate()
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("Producer", func() {
		var ctx context.Context
		var cancelFunc func()
		var logInterceptor *logWriter

		var producerSettings *notifications.ProducerSettings
		var smSettings *sm.Settings
		var producer *notifications.Producer
		var producerCtx context.Context

		var server *wsServer
		var group *sync.WaitGroup

		BeforeEach(func() {
			group = &sync.WaitGroup{}
			ctx, cancelFunc = context.WithCancel(context.Background())
			logInterceptor = &logWriter{}
			logInterceptor.Reset()
			var err error
			producerCtx, err = log.Configure(ctx, &log.Settings{
				Level:  "debug",
				Format: "text",
				Output: "ginkgowriter",
			})
			Expect(err).ToNot(HaveOccurred())
			log.AddHook(logInterceptor)
			server = newWSServer()
			server.Start()
			smSettings = &sm.Settings{
				URL:                  server.url,
				User:                 "admin",
				Password:             "admin",
				RequestTimeout:       2 * time.Second,
				NotificationsAPIPath: "/v1/notifications",
			}
			producerSettings = &notifications.ProducerSettings{
				MinPingPeriod:        100 * time.Millisecond,
				ReconnectDelay:       100 * time.Millisecond,
				PongTimeout:          20 * time.Millisecond,
				ResyncPeriod:         300 * time.Millisecond,
				PingPeriodPercentage: 60,
				MessagesQueueSize:    10,
			}
		})

		JustBeforeEach(func() {
			var err error
			producer, err = notifications.NewProducer(producerSettings, smSettings)
			Expect(err).ToNot(HaveOccurred())
		})

		notification := &types.Notification{
			Revision: 123,
			Payload:  json.RawMessage("{}"),
		}
		message := &notifications.Message{Notification: notification}

		AfterEach(func() {
			cancelFunc()
			server.Close()
		})

		Context("During websocket connect", func() {
			It("Sends correct basic credentials", func(done Done) {
				server.onRequest = func(r *http.Request) {
					defer GinkgoRecover()
					username, password, ok := r.BasicAuth()
					Expect(ok).To(BeTrue())
					Expect(username).To(Equal(smSettings.User))
					Expect(password).To(Equal(smSettings.Password))
					close(done)
				}
				producer.Start(producerCtx, group)
			})
		})

		Context("When last notification revision is not found", func() {
			BeforeEach(func() {
				requestCnt := 0
				server.onRequest = func(r *http.Request) {
					requestCnt++
					if requestCnt == 1 {
						server.statusCode = http.StatusGone
					} else {
						server.statusCode = 0
					}
				}
			})

			It("Sends restart message", func() {
				Eventually(producer.Start(producerCtx, group)).Should(Receive(Equal(&notifications.Message{Resync: true})))
			})
		})

		Context("When notifications is sent by the server", func() {
			BeforeEach(func() {
				server.onClientConnected = func(conn *websocket.Conn) {
					defer GinkgoRecover()
					err := conn.WriteJSON(notification)
					Expect(err).ToNot(HaveOccurred())
				}
			})

			It("forwards the notification on the channel", func() {
				Eventually(producer.Start(producerCtx, group)).Should(Receive(Equal(message)))
			})
		})

		Context("When connection is closed", func() {
			It("reconnects with last known notification revision", func(done Done) {
				requestCount := 0
				server.onRequest = func(r *http.Request) {
					defer GinkgoRecover()
					requestCount++
					if requestCount > 1 {
						rev := r.URL.Query().Get(smnotifications.LastKnownRevisionQueryParam)
						Expect(rev).To(Equal(strconv.FormatInt(notification.Revision, 10)))
						close(done)
					}
				}
				once := &sync.Once{}
				server.onClientConnected = func(conn *websocket.Conn) {
					once.Do(func() {
						defer GinkgoRecover()
						err := conn.WriteJSON(notification)
						Expect(err).ToNot(HaveOccurred())
						conn.Close()
					})
				}
				messages := producer.Start(producerCtx, group)
				Eventually(messages).Should(Receive(Equal(message)))
			})
		})

		Context("When a server that accept mTLS exists", func() {
			var tlsServer *httptest.Server
			var tlsSocketServer *wsServer

			BeforeEach(func() {
				tlsServer, tlsSocketServer = newServerWithTLS()
				producerSettings = &notifications.ProducerSettings{
					MinPingPeriod:        100 * time.Millisecond,
					ReconnectDelay:       100 * time.Millisecond,
					PongTimeout:          20 * time.Millisecond,
					ResyncPeriod:         300 * time.Millisecond,
					PingPeriodPercentage: 60,
					MessagesQueueSize:    10,
				}

				smSettings = &sm.Settings{
					URL:                  tlsServer.URL,
					User:                 "admin",
					Password:             "admin",
					RequestTimeout:       11 * time.Second,
					TLSClientCertificate: ClientCertificate,
					TLSClientKey:         ClientKey,
					SkipSSLValidation:    true,
					NotificationsAPIPath: "/v1/notifications",
				}
			})

			AfterEach(func() {
				tlsServer.Close()
			})

			When("valid mTLS certificate and private keys are used", func() {
				BeforeEach(func() {
					smSettings.TLSClientCertificate = ClientCertificate
					smSettings.TLSClientKey = ClientKey
				})

				It("should successfully establish a socket connection", func() {
					times := make(chan time.Time, 1)
					tlsSocketServer.onClientConnected = func(conn *websocket.Conn) {
						times <- time.Now()
					}
					newProducer, _ := notifications.NewProducer(producerSettings, smSettings)
					newProducer.Start(producerCtx, group)
					Eventually(times, time.Millisecond*100).Should(Receive())
				})
			})

			When("an invalid certificate is used with a valid client key", func() {
				BeforeEach(func() {
					smSettings.TLSClientCertificate = InvalidClientKey
					smSettings.TLSClientKey = ClientKey
				})

				It("should not establish a socket connection", func() {
					times := make(chan time.Time, 1)
					tlsSocketServer.onClientConnected = func(conn *websocket.Conn) {
						times <- time.Now()
					}
					newProducer, _ := notifications.NewProducer(producerSettings, smSettings)
					newProducer.Start(producerCtx, group)
					Consistently(times, time.Millisecond*100).ShouldNot(Receive())
				})
			})

			When("when both certificate and client key are invalid", func() {
				BeforeEach(func() {
					smSettings.TLSClientCertificate = InvalidClientCertificate
					smSettings.TLSClientKey = InvalidClientKey
				})

				It("should not establish a socket connection", func() {
					times := make(chan time.Time, 1)
					tlsSocketServer.onClientConnected = func(conn *websocket.Conn) {
						times <- time.Now()
					}
					newProducer, _ := notifications.NewProducer(producerSettings, smSettings)
					newProducer.Start(producerCtx, group)
					Consistently(times, time.Millisecond*100).ShouldNot(Receive())
				})
			})
		})

		Context("When a websocket is connected", func() {
			It("Pings the servers within max_ping_period", func(done Done) {
				times := make(chan time.Time, 10)
				server.onClientConnected = func(conn *websocket.Conn) {
					times <- time.Now()
				}
				server.pingHandler = func(string) error {
					defer GinkgoRecover()
					now := time.Now()
					times <- now
					if len(times) == 3 {
						start := <-times
						for i := 0; i < 2; i++ {
							t := <-times
							pingPeriod, err := time.ParseDuration(server.maxPingPeriod)
							Expect(err).ToNot(HaveOccurred())
							Expect(t.Sub(start)).To(BeNumerically("<", pingPeriod))
							start = t
						}
						close(done)
					}
					server.conn.WriteControl(websocket.PongMessage, []byte{}, now.Add(1*time.Second))
					return nil
				}
				producer.Start(producerCtx, group)
			})
		})

		Context("When invalid last notification revision is sent", func() {
			BeforeEach(func() {
				server.lastNotificationRevision = "-5"
			})

			It("returns error", func() {
				producer.Start(producerCtx, group)
				Eventually(logInterceptor.String).Should(ContainSubstring("invalid last notification revision"))
			})
		})

		Context("When server does not return pong within the timeout", func() {
			It("Reconnects", func(done Done) {
				var initialPingTime time.Time
				var timeBetweenReconnection time.Duration
				server.pingHandler = func(s string) error {
					if initialPingTime.IsZero() {
						initialPingTime = time.Now()
					}
					return nil
				}
				server.onClientConnected = func(conn *websocket.Conn) {
					if !initialPingTime.IsZero() {
						defer GinkgoRecover()
						timeBetweenReconnection = time.Now().Sub(initialPingTime)
						pingPeriod, err := time.ParseDuration(server.maxPingPeriod)
						Expect(err).ToNot(HaveOccurred())
						Expect(timeBetweenReconnection).To(BeNumerically("<", producerSettings.ReconnectDelay+pingPeriod))
						close(done)
					}
				}
				producer.Start(producerCtx, group)
			})
		})

		Context("When notification is not a valid JSON", func() {
			It("Logs error and reconnects", func(done Done) {
				connectionAttempts := 0
				server.onClientConnected = func(conn *websocket.Conn) {
					defer GinkgoRecover()
					err := conn.WriteMessage(websocket.TextMessage, []byte("not-json"))
					Expect(err).ToNot(HaveOccurred())
					connectionAttempts++
					if connectionAttempts == 2 {
						Eventually(logInterceptor.String).Should(ContainSubstring("unmarshal"))
						close(done)
					}
				}
				producer.Start(producerCtx, group)
			})
		})

		assertLogContainsMessageOnReconnect := func(message string, done Done) {
			connectionAttempts := 0
			server.onClientConnected = func(conn *websocket.Conn) {
				defer GinkgoRecover()
				connectionAttempts++
				if connectionAttempts == 2 {
					Eventually(logInterceptor.String).Should(ContainSubstring(message))
					close(done)
				}
			}
			producer.Start(producerCtx, group)
		}

		Context("When last_notification_revision is not a number", func() {
			BeforeEach(func() {
				server.lastNotificationRevision = "not a number"
			})
			It("Logs error and reconnects", func(done Done) {
				assertLogContainsMessageOnReconnect(server.lastNotificationRevision, done)
			})
		})

		Context("When max_ping_period is not a number", func() {
			BeforeEach(func() {
				server.maxPingPeriod = "not a number"
			})
			It("Logs error and reconnects", func(done Done) {
				assertLogContainsMessageOnReconnect(server.maxPingPeriod, done)
			})
		})

		Context("When max_ping_period is less than the configured min ping period", func() {
			BeforeEach(func() {
				server.maxPingPeriod = (producerSettings.MinPingPeriod - 20*time.Millisecond).String()
			})
			It("Logs error and reconnects", func(done Done) {
				assertLogContainsMessageOnReconnect(server.maxPingPeriod, done)
			})
		})

		Context("When SM URL is not valid", func() {
			It("Returns error", func() {
				smSettings.URL = "::invalid-url"
				newProducer, err := notifications.NewProducer(producerSettings, smSettings)
				Expect(newProducer).To(BeNil())
				Expect(err).To(HaveOccurred())
			})
		})

		Context("When SM returns error status", func() {
			BeforeEach(func() {
				server.statusCode = http.StatusInternalServerError
			})
			It("Logs and reconnects", func(done Done) {
				connectionAttempts := 0
				server.onRequest = func(r *http.Request) {
					connectionAttempts++
					if connectionAttempts == 2 {
						server.statusCode = 0
					}
				}
				server.onClientConnected = func(conn *websocket.Conn) {
					Eventually(logInterceptor.String).Should(ContainSubstring("bad handshake"))
					close(done)
				}
				producer.Start(producerCtx, group)
			})
		})

		Context("When cannot connect to given address", func() {
			BeforeEach(func() {
				smSettings.URL = "http://bad-host"
			})
			It("Logs the error and tries to reconnect", func() {
				var err error
				producer, err = notifications.NewProducer(producerSettings, smSettings)
				Expect(err).ToNot(HaveOccurred())
				producer.Start(producerCtx, group)
				Eventually(logInterceptor.String, 4*time.Second).Should(ContainSubstring("could not connect websocket"))
				Eventually(logInterceptor.String, 4*time.Second).Should(ContainSubstring("Reattempting to establish websocket connection"))
			})
		})

		Context("When context is canceled", func() {
			It("Releases the group", func() {
				testCtx, cancel := context.WithCancel(context.Background())
				waitGroup := &sync.WaitGroup{}
				producer.Start(testCtx, waitGroup)
				cancel()
				waitGroup.Wait()
			})
		})

		Context("When messages queue is full", func() {
			BeforeEach(func() {
				producerSettings.MessagesQueueSize = 2
				server.onClientConnected = func(conn *websocket.Conn) {
					defer GinkgoRecover()
					err := conn.WriteJSON(notification)
					Expect(err).ToNot(HaveOccurred())
					err = conn.WriteJSON(notification)
					Expect(err).ToNot(HaveOccurred())
				}
			})
			It("Canceling context stops the goroutines", func() {
				producer.Start(producerCtx, group)
				Eventually(logInterceptor.String).Should(ContainSubstring("Received notification "))
				cancelFunc()
				Eventually(logInterceptor.String).Should(ContainSubstring("Exiting notification reader"))
			})
		})

		Context("When resync time elapses", func() {
			BeforeEach(func() {
				producerSettings.ResyncPeriod = 10 * time.Millisecond
			})
			It("Sends a resync message", func(done Done) {
				messages := producer.Start(producerCtx, group)
				Expect(<-messages).To(Equal(&notifications.Message{Resync: true})) // on initial connect
				Expect(<-messages).To(Equal(&notifications.Message{Resync: true})) // first time the timer ticks
				Expect(<-messages).To(Equal(&notifications.Message{Resync: true})) // second time the timer ticks
				close(done)
			}, (500 * time.Millisecond).Seconds())
		})

		Context("When a force resync (410 GONE) is triggered", func() {
			BeforeEach(func() {
				producerSettings.ResyncPeriod = 200 * time.Millisecond
				producerSettings.ReconnectDelay = 150 * time.Millisecond

				requestCnt := 0
				server.onRequest = func(r *http.Request) {
					requestCnt++
					if requestCnt == 1 {
						server.statusCode = 500
					} else {
						server.statusCode = 0
					}
				}
			})
			It("Resets the resync timer period", func() {
				messages := producer.Start(producerCtx, group)
				start := time.Now()
				m := <-messages
				t := time.Now().Sub(start)
				Expect(m).To(Equal(&notifications.Message{Resync: true})) // on initial connect
				Expect(t).To(BeNumerically(">=", producerSettings.ReconnectDelay))

				m = <-messages
				t = time.Now().Sub(start)
				Expect(m).To(Equal(&notifications.Message{Resync: true})) // first time the timer ticks
				Expect(t).To(BeNumerically(">=", producerSettings.ResyncPeriod+producerSettings.ReconnectDelay))
			})
		})
	})
})

type wsServer struct {
	url                      string
	server                   *httptest.Server
	mux                      *http.ServeMux
	conn                     *websocket.Conn
	connMutex                sync.Mutex
	lastNotificationRevision string
	maxPingPeriod            string
	statusCode               int
	onClientConnected        func(*websocket.Conn)
	onRequest                func(r *http.Request)
	pingHandler              func(string) error
}

func newWSServer() *wsServer {
	s := &wsServer{
		maxPingPeriod:            (100 * time.Millisecond).String(),
		lastNotificationRevision: strconv.FormatInt(types.InvalidRevision, 10),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/notifications", s.handler)
	s.mux = mux
	return s
}

func newServerWithTLS() (*httptest.Server, *wsServer) {
	s := &wsServer{
		maxPingPeriod:            (100 * time.Millisecond).String(),
		lastNotificationRevision: strconv.FormatInt(types.InvalidRevision, 10),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/notifications", s.handler)

	uServer := httptest.NewUnstartedServer(mux)
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM([]byte(ClientCertificate))
	uServer.TLS = &tls.Config{}
	uServer.TLS.ClientCAs = caCertPool
	uServer.TLS.ClientAuth = tls.RequireAndVerifyClientCert
	uServer.StartTLS()
	return uServer, s
}

func (s *wsServer) Start() {
	s.server = httptest.NewServer(s.mux)
	s.url = s.server.URL
}

func (s *wsServer) Close() {
	if s == nil {
		return
	}
	if s.server != nil {
		s.server.Close()
	}

	s.connMutex.Lock()
	defer s.connMutex.Unlock()
	if s.conn != nil {
		s.conn.Close()
	}
}

func (s *wsServer) handler(w http.ResponseWriter, r *http.Request) {
	if s.onRequest != nil {
		s.onRequest(r)
	}

	var err error
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	header := http.Header{}
	if s.lastNotificationRevision != invalidRevisionStr {
		header.Set(smnotifications.LastKnownRevisionHeader, s.lastNotificationRevision)
	}
	header.Set(smnotifications.MaxPingPeriodHeader, s.maxPingPeriod)
	if s.statusCode != 0 {
		w.WriteHeader(s.statusCode)
		w.Write([]byte{})
		return
	}

	s.connMutex.Lock()
	defer s.connMutex.Unlock()
	s.conn, err = upgrader.Upgrade(w, r, header)
	if err != nil {
		log.C(r.Context()).WithError(err).Error("Could not upgrade websocket connection")
		return
	}
	if s.pingHandler != nil {
		s.conn.SetPingHandler(s.pingHandler)
	}
	if s.onClientConnected != nil {
		s.onClientConnected(s.conn)
	}
	go reader(s.conn)
}

func reader(conn *websocket.Conn) {
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			return
		}
	}
}

type logWriter struct {
	strings.Builder
	bufferMutex sync.Mutex
}

func (w *logWriter) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (w *logWriter) Fire(entry *logrus.Entry) error {
	str, err := entry.String()
	if err != nil {
		return err
	}
	w.bufferMutex.Lock()
	defer w.bufferMutex.Unlock()
	_, err = w.WriteString(str)
	return err
}

func (w *logWriter) String() string {
	w.bufferMutex.Lock()
	defer w.bufferMutex.Unlock()
	return w.Builder.String()
}
