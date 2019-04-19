package notifications_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/Peripli/service-broker-proxy/pkg/sbproxy/notifications"
	"github.com/Peripli/service-broker-proxy/pkg/sm"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/types"
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

	var handler *notifications.Handler
	var wsServer *wsServer

	BeforeEach(func() {
		ctx, cancelFunc = context.WithCancel(context.Background())
		handlerCtx := log.Configure(ctx, &log.Settings{
			Level:  "debug",
			Format: "text",
			Output: GinkgoWriter,
		})
		wsServer = newWSServer()
		wsServer.Start()
		handlerSettings := &sm.Settings{
			URL:                  wsServer.url,
			User:                 "admin",
			Password:             "admin",
			NotificationsAPIPath: "/v1/notifications",
			RequestTimeout:       2 * time.Second,
		}
		handler = notifications.NewHandler(handlerCtx, handlerSettings)
	})

	startHandler := func(resyncChanSize, notificationsQueueSize int) {
		resyncChan = make(chan struct{}, resyncChanSize)
		notificationsQueue = make(notifications.ChannelQueue, notificationsQueueSize)
		handler.Start(resyncChan, notificationsQueue)
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
			wsServer.statusCode = http.StatusGone
			startHandler(10, 10)
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
			startHandler(10, 10)
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
			startHandler(10, 10)
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
						Expect(t.Sub(start)).To(BeNumerically("<", wsServer.maxPingPeriod))
						start = t
					}
					close(done)
				}
				wsServer.conn.WriteControl(websocket.PongMessage, []byte{}, now.Add(1*time.Second))
				return nil
			}
			startHandler(10, 10)
		})
	})
})

type wsServer struct {
	url                              string
	server                           *httptest.Server
	mux                              *http.ServeMux
	conn                             *websocket.Conn
	connMutex                        sync.Mutex
	lastNotificationRevision         int
	maxPingPeriod                    time.Duration
	statusCode                       int
	onClientConnected                func(*websocket.Conn)
	lastReceivedNotificationRevision int64
	onRequest                        func(r *http.Request)
	pingHandler                      func(string) error
}

func newWSServer() *wsServer {
	s := &wsServer{
		maxPingPeriod:            100 * time.Millisecond,
		lastNotificationRevision: 1,
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
	header.Set("last_notification_revision", strconv.Itoa(s.lastNotificationRevision))
	header.Set("max_ping_period", s.maxPingPeriod.String())
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
