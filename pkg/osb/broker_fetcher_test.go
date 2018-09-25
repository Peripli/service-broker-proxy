package osb_test

import (
	"context"
	"github.com/Peripli/service-broker-proxy/pkg/osb"
	smosb "github.com/Peripli/service-manager/api/osb"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("BrokerFetcher", func() {
	Describe("FetcherBroker", func() {
		const (
			user     = "admin"
			password = "admin"
			url      = "https://example.com"
			brokerID = "brokerID"
		)
		var fetcher smosb.BrokerFetcher

		BeforeEach(func() {
			fetcher = &osb.BrokerDetails{
				Username: user,
				Password: password,
				URL:      url,
			}
		})

		It("returns a broker type with correct broker details", func() {
			broker, err := fetcher.FetchBroker(context.TODO(), brokerID)

			Expect(err).ShouldNot(HaveOccurred())
			Expect(broker.Credentials.Basic.Username).To(Equal(user))
			Expect(broker.Credentials.Basic.Password).To(Equal(password))
			Expect(broker.BrokerURL).To(Equal(url + "/" + brokerID))
		})
	})
})
