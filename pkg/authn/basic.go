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
	"fmt"
	httpsec "github.com/Peripli/service-manager/pkg/security/http"

	"github.com/Peripli/service-manager/pkg/web"
)

// NewInMemoryAuthenticator builds new basic authenticator with in memory defined username and password
func NewInMemoryAuthenticator(user, password string) httpsec.Authenticator {
	return &inmemoryBasicAuthenticator{
		expectedUsername: user,
		expectedPassword: password,
	}
}

type inmemoryBasicAuthenticator struct {
	expectedUsername string
	expectedPassword string
}

func (a *inmemoryBasicAuthenticator) Authenticate(request *web.Request) (*web.UserContext, httpsec.Decision, error) {
	username, password, ok := request.BasicAuth()
	if !ok {
		return nil, httpsec.Abstain, nil
	}

	if username != a.expectedUsername || password != a.expectedPassword {
		return nil, httpsec.Deny, fmt.Errorf("provided credentials are invalid")
	}

	return &web.UserContext{
		Name: username,
	}, httpsec.Allow, nil
}
