package notifications_test

import (
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/Peripli/service-broker-proxy/pkg/sbproxy/notifications"
	"github.com/Peripli/service-broker-proxy/pkg/sm"
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
		wsServer = newWSServer()
		wsServer.Start()
		handlerSettings := &sm.Settings{
			URL:                  wsServer.url,
			User:                 "admin",
			Password:             "admin",
			NotificationsAPIPath: "/v1/notifications",
			RequestTimeout:       2 * time.Second,
		}
		handler = notifications.NewHandler(ctx, handlerSettings)
	})

	startHandler := func(resyncChanSize, notificationsQueueSize int) {
		resyncChan = make(chan struct{}, resyncChanSize)
		notificationsQueue = make(notifications.ChannelQueue, notificationsQueueSize)
		go handler.Start(resyncChan, notificationsQueue)
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

	Context("When last notification revision is found", func() {
		BeforeEach(func() {
			wsServer.lastNotificationRevision = 1
			startHandler(10, 10)
		})

		It("connects the websocket", func() {
			Eventually(resyncChan).Should(Receive())
		})
	})

})

type wsServer struct {
	url                      string
	server                   *httptest.Server
	mux                      *http.ServeMux
	conn                     *websocket.Conn
	lastNotificationRevision int
	maxPingPeriod            time.Duration
	statusCode               int
}

func newWSServer() *wsServer {
	s := &wsServer{}
	s.maxPingPeriod = time.Second * 5
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
	if s.conn != nil {
		s.conn.Close()
	}
}

func (s *wsServer) handler(w http.ResponseWriter, r *http.Request) {
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
	s.conn, err = upgrader.Upgrade(w, r, header)
	if err != nil {
		log.Println(err)
		return
	}
}
