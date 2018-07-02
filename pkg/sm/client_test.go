package sm_test

import (
	. "github.com/onsi/ginkgo"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("Client", func() {
	var (
		ccServer *ghttp.Server
	)

	BeforeEach(func() {
		ccServer = ghttp.NewServer()

	})

	AfterEach(func() {
		ccServer.Close()
	})

	Describe("GetBrokers", func() {
		Context("when an error status code is returned by the Service Manager", func() {
			It("returns an error", func() {

			})

		})

		Context("when an invalid response is returned by the Service Manager", func() {
			It("returns an error", func() {

			})
		})

		Context("when the request is successful", func() {
			Context("when no brokers are found in the Service Manager", func() {
				It("returns an empty slice", func() {

				})

				It("returns no error", func() {

				})

			})

			Context("when brokers are found in the Service Manager", func() {
				It("returns all of the brokers", func() {

				})

				It("returns no error", func() {

				})
			})
		})
	})
})
