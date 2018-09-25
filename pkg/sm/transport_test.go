package sm

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/onsi/gomega/ghttp"
	"net/http"
)

var _ = Describe("Transport", func() {
	var testServer *ghttp.Server
	var req *http.Request
	var err error

	BeforeEach(func() {
		testServer  = ghttp.NewServer()
		req, err = http.NewRequest("GET", testServer.URL() + "/v1/test", nil)
		Expect(err).ShouldNot(HaveOccurred())
	})

	Describe("Basic auth transport", func() {
		var basicTransport *BasicAuthTransport

		BeforeEach(func() {
			basicTransport = &BasicAuthTransport{
				Username: "admin",
				Password: "admin",
				Rt:       &SkipSSLTransport{
					SkipSslValidation: true,
				},
			}
		})

		It("adds Authorization header when basic auth credentials are present", func() {
			testServer.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/v1/test"),
					ghttp.VerifyBasicAuth("admin", "admin"),
					ghttp.RespondWith(http.StatusOK, "{}"),
				),
			)

			response, err := basicTransport.RoundTrip(req)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(response.StatusCode).To(Equal(http.StatusOK))
		})

		It("does not add Authorization header when basic auth credentials are missing", func() {
			testServer.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/v1/test"),
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						Expect(r.Header.Get("Authorization")).To(BeEmpty())
					}),
					ghttp.RespondWith(http.StatusOK, "{}"),
				),
			)

			basicTransport.Username = ""
			basicTransport.Password = ""

			response, err := basicTransport.RoundTrip(req)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(response.StatusCode).To(Equal(http.StatusOK))
		})
	})

})
