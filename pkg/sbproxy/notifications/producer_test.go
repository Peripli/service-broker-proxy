package notifications_test

import (
	"context"
	"encoding/json"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/sirupsen/logrus"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Peripli/service-broker-proxy/pkg/sbproxy/notifications"
	"github.com/Peripli/service-broker-proxy/pkg/sm"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/gorilla/websocket"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestNotifications(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Notifications Suite")
}

var _ = Describe("Notifications", func() {
	var ctx context.Context
	var cancelFunc func()
	var resyncChan chan struct{}
	var notificationsQueue notifications.Queue
	var logInterceptor *logWriter

	var settings *notifications.ProducerSettings
	var producer *notifications.Producer
	var producerCtx context.Context

	var wsServer *wsServer

	BeforeEach(func() {
		ctx, cancelFunc = context.WithCancel(context.Background())
		logInterceptor = &logWriter{}
		logInterceptor.Reset()
		log.AddHook(logInterceptor)
		producerCtx = log.Configure(ctx, &log.Settings{
			Level:  "debug",
			Format: "text",
			Output: GinkgoWriter,
		})
		wsServer = newWSServer()
		wsServer.Start()
		settings = notifications.DefaultProducerSettings(&sm.Settings{
			URL:                  wsServer.url,
			User:                 "admin",
			Password:             "admin",
			NotificationsAPIPath: "/v1/notifications",
			RequestTimeout:       2 * time.Second,
		})

		settings.MinPingPeriod = 100 * time.Millisecond
		settings.ReconnectDelay = 100 * time.Millisecond

		var err error
		producer, err = notifications.NewProducer(settings)
		Expect(err).ToNot(HaveOccurred())
	})

	startProducer := func(resyncChanSize, notificationsQueueSize int) {
		resyncChan = make(chan struct{}, resyncChanSize)
		notificationsQueue = make(notifications.ChannelQueue, notificationsQueueSize)
		producer.Start(producerCtx, resyncChan, notificationsQueue)
	}

	notification := &types.Notification{
		Revision: 123,
		Payload:  json.RawMessage("{}"),
	}

	AfterEach(func() {
		cancelFunc()
		wsServer.Close()
	})

	Context("When last notification revision is not found", func() {
		BeforeEach(func() {
			requestCnt := 0
			wsServer.onRequest = func(r *http.Request) {
				requestCnt++
				if requestCnt == 1 {
					wsServer.statusCode = http.StatusGone
				} else {
					wsServer.statusCode = 0
				}
			}
			startProducer(10, 10)
		})

		It("Sends on resync channel", func() {
			Eventually(resyncChan).Should(Receive())
		})
	})

	Context("When notifications is sent by the server", func() {
		BeforeEach(func() {
			wsServer.onClientConnected = func(conn *websocket.Conn) {
				defer GinkgoRecover()
				err := conn.WriteJSON(notification)
				Expect(err).ToNot(HaveOccurred())
			}
			startProducer(10, 10)
		})

		It("forwards the notification on the channel", func() {
			Eventually(notificationsQueue.Listen()).Should(Receive(Equal(notification)))
		})
	})

	Context("When connection is closed", func() {
		It("reconnects with last known notification revision", func(done Done) {
			requestCount := 0
			wsServer.onRequest = func(r *http.Request) {
				defer GinkgoRecover()
				requestCount++
				if requestCount > 1 {
					rev := r.URL.Query().Get("last_notification_revision")
					Expect(rev).To(Equal(strconv.FormatInt(notification.Revision, 10)))
					close(done)
				}
			}
			once := &sync.Once{}
			wsServer.onClientConnected = func(conn *websocket.Conn) {
				once.Do(func() {
					defer GinkgoRecover()
					err := conn.WriteJSON(notification)
					Expect(err).ToNot(HaveOccurred())
					Eventually(notificationsQueue.Listen()).Should(Receive(Equal(notification)))
					conn.Close()
				})
			}
			startProducer(10, 10)
		})
	})

	Context("When websocket is connected", func() {
		It("Pings the servers within max_ping_period", func(done Done) {
			times := make(chan time.Time, 10)
			wsServer.onClientConnected = func(conn *websocket.Conn) {
				times <- time.Now()
			}
			wsServer.pingHandler = func(string) error {
				defer GinkgoRecover()
				now := time.Now()
				times <- now
				if len(times) == 3 {
					start := <-times
					for i := 0; i < 2; i++ {
						t := <-times
						pingPeriod, err := time.ParseDuration(wsServer.maxPingPeriod)
						Expect(err).ToNot(HaveOccurred())
						Expect(t.Sub(start)).To(BeNumerically("<", pingPeriod))
						start = t
					}
					close(done)
				}
				wsServer.conn.WriteControl(websocket.PongMessage, []byte{}, now.Add(1*time.Second))
				return nil
			}
			startProducer(10, 10)
		})
	})

	Context("When invalid last notification revision is sent", func() {
		BeforeEach(func() {
			wsServer.lastNotificationRevision = "-1"

		})
		It("returns error", func() {
			startProducer(10, 10)
			Eventually(logInterceptor.String).Should(ContainSubstring("invalid last notification revision"))
		})
	})

	Context("When server does not return pong within the timeout", func() {
		It("Reconnects", func(done Done) {
			var initialPingTime time.Time
			var timeBetweenReconnection time.Duration
			wsServer.pingHandler = func(s string) error {
				if initialPingTime.IsZero() {
					initialPingTime = time.Now()
				}
				return nil
			}
			wsServer.onClientConnected = func(conn *websocket.Conn) {
				if !initialPingTime.IsZero() {
					defer GinkgoRecover()
					timeBetweenReconnection = time.Now().Sub(initialPingTime)
					pingPeriod, err := time.ParseDuration(wsServer.maxPingPeriod)
					Expect(err).ToNot(HaveOccurred())
					Expect(timeBetweenReconnection).To(BeNumerically("<", settings.ReconnectDelay+pingPeriod))
					close(done)
				}
			}
			startProducer(10, 10)

		})
	})

	Context("When notification is not a valid JSON", func() {
		It("Logs error and reconnects", func(done Done) {
			connectionAttempts := 0
			wsServer.onClientConnected = func(conn *websocket.Conn) {
				defer GinkgoRecover()
				err := conn.WriteMessage(websocket.TextMessage, []byte("not-json"))
				Expect(err).ToNot(HaveOccurred())
				connectionAttempts++
				if connectionAttempts == 2 {
					Eventually(logInterceptor.String).Should(ContainSubstring("unmarshal"))
					close(done)
				}
			}
			startProducer(10, 10)
		})
	})

	Context("When last_notification_revision is not a number", func() {
		BeforeEach(func() {
			wsServer.lastNotificationRevision = "not a number"
		})
		It("Logs error and reconnects", func(done Done) {
			connectionAttempts := 0
			wsServer.onClientConnected = func(conn *websocket.Conn) {
				defer GinkgoRecover()
				connectionAttempts++
				if connectionAttempts == 2 {
					Eventually(logInterceptor.String).Should(ContainSubstring(wsServer.lastNotificationRevision))
					close(done)
				}
			}
			startProducer(10, 10)
		})
	})

	Context("When max_ping_period is not a number", func() {
		BeforeEach(func() {
			wsServer.maxPingPeriod = "not a number"
		})
		It("Logs error and reconnects", func(done Done) {
			connectionAttempts := 0
			wsServer.onClientConnected = func(conn *websocket.Conn) {
				defer GinkgoRecover()
				connectionAttempts++
				if connectionAttempts == 2 {
					Eventually(logInterceptor.String).Should(ContainSubstring(wsServer.maxPingPeriod))
					close(done)
				}
			}
			startProducer(10, 10)
		})
	})

	Context("When max_ping_period is less than the configured min ping period", func() {
		BeforeEach(func() {
			wsServer.maxPingPeriod = (settings.MinPingPeriod - 20*time.Millisecond).String()
		})
		It("Logs error and reconnects", func(done Done) {
			connectionAttempts := 0
			wsServer.onClientConnected = func(conn *websocket.Conn) {
				defer GinkgoRecover()
				connectionAttempts++
				if connectionAttempts == 2 {
					Eventually(logInterceptor.String).Should(ContainSubstring(wsServer.maxPingPeriod))
					close(done)
				}
			}
			startProducer(10, 10)
		})
	})

	Context("When SM URL is not valid", func() {
		It("Returns error", func() {
			settings.URL = "::invalid-url"
			newProducer, err := notifications.NewProducer(settings)
			Expect(newProducer).To(BeNil())
			Expect(err).To(HaveOccurred())
		})
	})

	Context("When SM returns error status", func() {
		BeforeEach(func() {
			wsServer.statusCode = http.StatusInternalServerError
		})
		It("Logs and reconnects", func(done Done) {
			connectionAttempts := 0
			wsServer.onRequest = func(r *http.Request) {
				connectionAttempts++
				if connectionAttempts == 2 {
					wsServer.statusCode = 0
				}
			}
			wsServer.onClientConnected = func(conn *websocket.Conn) {
				Eventually(logInterceptor.String).Should(ContainSubstring("bad handshake"))
				close(done)
			}
			startProducer(10, 10)
		})
	})

	Context("When cannot connect to given address", func() {
		BeforeEach(func() {
			settings.URL = "http://bad-host"
			var err error
			producer, err = notifications.NewProducer(settings)
			Expect(err).ToNot(HaveOccurred())
		})
		It("Logs the error and tries to reconnect", func() {
			startProducer(10, 10)
			Eventually(logInterceptor.String).Should(ContainSubstring("no such host"))
			Eventually(logInterceptor.String).Should(ContainSubstring("Attempting to reestablish websocket connection"))
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
		lastNotificationRevision: "1",
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/notifications", s.handler)
	s.mux = mux
	return s
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
	header.Set("last_notification_revision", s.lastNotificationRevision)
	header.Set("max_ping_period", s.maxPingPeriod)
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
}

func (w *logWriter) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (w *logWriter) Fire(entry *logrus.Entry) error {
	str, _ := entry.String()
	_, err := w.WriteString(str)
	return err
}
