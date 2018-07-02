package task

import (
	"strconv"

	"context"
	"os"
	"os/signal"

	"time"

	"sync"

	"github.com/Peripli/service-broker-proxy/pkg/osb"
	"github.com/pmorie/osb-broker-lib/pkg/rest"
	"github.com/pmorie/osb-broker-lib/pkg/server"
	"github.com/sirupsen/logrus"
)

var wg sync.WaitGroup
var ctx context.Context
var cancel context.CancelFunc
var term chan os.Signal

func init() {

	term = make(chan os.Signal)
	signal.Notify(term, os.Interrupt)
}

func main() {
	// TODO router swap + cron job
	ctx, cancel = context.WithCancel(context.Background())
	ticker := time.NewTicker(5 * time.Second)

	defer func() {
		logrus.Info("Running deferred cancel func")
		cancel()
		logrus.Info("Main thread blocking for WG waiter...")
		wg.Wait()
		logrus.Info("WG waiting finished")
	}()

	go func() {
		wg.Add(1)
		defer wg.Done()
		select {
		case <-term:
			logrus.Info("Received OS interrupt, exiting gracefully...")
			cancel()
		case <-ctx.Done():
			return
		}
	}()
	go func() {
		//
		wg.Add(1)
		defer wg.Done()
		//
		for {
			select {
			case <-ticker.C:
				logrus.Println("ticker sleeping for 4 sec...")
				time.Sleep(4 * time.Second)
				logrus.Println("ticker finished sleeping for 4!")
			case <-ctx.Done():
				logrus.Println("Cancel came in ticker so stopping")
				ticker.Stop()
				return
			}
		}
	}()
	if err := runServer(ctx); err != nil && err != context.Canceled && err != context.DeadlineExceeded {
		logrus.Errorln("Server stopping: ", err)
	}

}

func runServer(ctx context.Context) error {
	addr := ":" + strconv.Itoa(8000)
	businessLogic, err := osb.NewBusinessLogic(nil, nil)
	if err != nil {
		return err
	}
	api, err := rest.NewAPISurface(businessLogic, nil)
	if err != nil {
		return err
	}
	s := server.New(api, nil)
	logrus.Info("Starting pmorie sbproxy!")
	defer logrus.Info("Pmorie Server finished running")
	return s.Run(ctx, addr)
}
