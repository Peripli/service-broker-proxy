/*
 * Copyright 2018 The Service Manager Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package authn

import (
	"github.com/Peripli/service-manager/pkg/web"
	"net/http"

	httpsec "github.com/Peripli/service-manager/pkg/security/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Basic Authentication wrapper", func() {
	const (
		validUsername   = "validUsername"
		validPassword   = "validPassword"
		invalidUser     = "invalidUser"
		invalidPassword = "invalidPassword"
	)
	var (
		basic httpsec.Authenticator
	)

	newRequest := func(user, pass string) *http.Request {
		request, err := http.NewRequest("GET", "", nil)
		Expect(err).NotTo(HaveOccurred())
		request.SetBasicAuth(user, pass)
		return request
	}

	BeforeEach(func() {
		basic = NewInMemoryAuthenticator(validUsername, validPassword)
	})

	DescribeTable("when given a request with basic authorization",
		func(expectedDecision httpsec.Decision, expectsError bool, username, password string) {

			request := newRequest(username, password)
			_, decision, err := basic.Authenticate(&web.Request{Request: request})
			if expectsError {
				Expect(err).To(HaveOccurred())
				Expect(decision).To(Equal(expectedDecision))
			} else {
				Expect(decision).To(Equal(expectedDecision))
			}
		},
		Entry("returns 401 for empty username", httpsec.Deny, true, "", validPassword),
		Entry("returns 401 for empty password", httpsec.Deny, true, validUsername, ""),
		Entry("returns 401 for invalid credentials", httpsec.Deny, true, invalidUser, invalidPassword),
		Entry("returns 200 for valid credentials", httpsec.Allow, false, validUsername, validPassword),
	)
})
