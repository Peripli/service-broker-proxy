package handlers_test

import (
	"fmt"
	"github.com/Peripli/service-broker-proxy/pkg/platform"
	"strings"
	"testing"

	"github.com/Peripli/service-manager/test/testutil"
	"github.com/sirupsen/logrus"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const brokerNameForNameProvider = "Broker_Name"

type PlatformBrokerClientMock struct {
	platform.BrokerClient
	platform.BrokerPlatformNameProvider
}

type PlatformVisibilityClientMock struct {
	platform.VisibilityClient
	platform.BrokerPlatformNameProvider
}

func TestHandlers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Handlers Suite")
}

func VerifyErrorLogged(f func()) {
	hook := &testutil.LogInterceptor{}
	logrus.AddHook(hook)
	Expect(hook).ToNot(ContainSubstring("error"))
	f()
	Expect(hook).To(ContainSubstring("error"))
}

func mockGetBrokerPlatformNameFunc(name string) string {
	return strings.Replace(strings.ToLower(name), "_", "-", -1)
}

func brokerProxyName(prefix, brokerName, brokerID string) string {
	return fmt.Sprintf("%s%s-%s", prefix, brokerName, brokerID)
}
