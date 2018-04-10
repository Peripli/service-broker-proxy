package middleware_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"net/http"
	"net/http/httptest"

	"encoding/json"

	. "github.com/Peripli/service-broker-proxy/pkg/sbproxy/middleware"
)

var _ = Describe("Basic Authentication wrapper", func() {
	const (
		validUsername = "validUsername"
		validPassword = "validPassword"
	)
	var (
		httpRecorder   *httptest.ResponseRecorder
		wrappedHandler http.Handler
	)

	newRequest := func(user, pass string) *http.Request {
		request, err := http.NewRequest("GET", "", nil)
		Expect(err).NotTo(HaveOccurred())
		request.SetBasicAuth(user, pass)
		return request
	}

	BeforeEach(func() {
		httpRecorder = httptest.NewRecorder()
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("{}"))
		})
		wrappedHandler = BasicAuth(validUsername, validPassword)(handler)
	})

	DescribeTable("when given a request with basic authorization",
		func(expectedStatus int, expectedError, username, password string) {
			request := newRequest(username, password)
			wrappedHandler.ServeHTTP(httpRecorder, request)

			Expect(httpRecorder.Code).To(Equal(expectedStatus))
			if expectedError != "" {
				var body map[string]string
				err := json.Unmarshal(httpRecorder.Body.Bytes(), &body)
				Expect(err).ToNot(HaveOccurred())
				Expect(body["error"]).To(Equal(expectedError))
				Expect(body["description"]).To(Not(BeEmpty()))
			}
		},
		Entry("returns 401 for empty username", http.StatusUnauthorized, "Not Authorized", "", validPassword),
		Entry("returns 401 for empty password", http.StatusUnauthorized, "Not Authorized", validUsername, ""),
		Entry("returns 401 for invalid credentials", http.StatusUnauthorized, "Not Authorized", "test", "test"),
		Entry("returns 200 for valid credentials", http.StatusOK, "", validUsername, validPassword),
	)
})
