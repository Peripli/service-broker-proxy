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

package sm

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"io"
	"net/http"
	"net/http/httptest"
	"path"
	"strconv"
	"time"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/web"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/pkg/errors"

	"github.com/Peripli/service-manager/test/common"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

const smURL = "http://example.com"

type MockTransport struct {
	f func(req *http.Request) (*http.Response, error)
}

//Test certificates
var (
	InvalidClientCertificate = `-----BEGIN CERTIFICATE-----
MIICrDCCAZQCCQCpKp1xS5g91TANBgkqhkiG9w0BAQUFADAYMRYwFAYDVQQDDA1G
aXJzdCBNLiBMYXN0MB4XDTIwMDMyNTE2MjQzNFoXDTIwMDQyNDE2MjQzNFowGDEW
MBQGA1UEAwwNRmlyc3QgTS4gTGFzdDCCASIwDQYJKoZIhvcNAQEBBQADggEPADCC
AQoCggEBAMX+zkYBETE0RtBlbE6OBesnTYDNYA345vGkpVlcJEZwFEr5mMFZ5z2s
dUQuUwLJIStDvxphrUCof99vPr925Ladcq/W9Gl4DkG7NP76AXeykJYC+7mvoZJH
fOSRM6TOA/OkdYk5ZGmQzflSvO33Xyj+MmZ/9uQtkz+ZCOyHxcwPxFFinPNl5fZv
LiN0p+zIvIFScpiAm1/YziaFX7XdJXlhBxMeqUWdDQuGCYGiRYdU7oZB7t2gA2Wa
oXjiOWpkd0ka4/EM/rPPWudwlbIMHd2e+ddwc7XFtN5wBhdo3bB8s3Dc9B8S3Xv1
e+1a/Iq7sQ//5eMh3s69/Sghi5Avu9kCAwEAATANBgkqhkiG9w0BAQUFAAOCAQEA
br2agn+j98e2KANkEM99Xk/oBsSmyp2tkwYr7p4dJUTCa8auLJbjIsdX9vsyprUd
JvoUamO8TKSGMhkjTS6MZJpILqJMWTpBXmDb0lktwJd062QLZklhttvhLv9YJuEG
/VXWgyRrQ4Jpy2+QAc8sWhtF42Vyu7sJZuikyT8FWO3fsILpdIg/SjsO2UGsP8hE
e6sqXUtHY6HXCdwRA4STY3ICZVMjw+NkfJPpC3gL7xbZHzzlwxQXqo7ySal4e6Sl
5ruwMNVt+2KVXdshqGNbRYNGs0x96iWbi8zRFx3uuIwLIDghWQqv+8gKcxttvkjP
LjRFM2n5hd6MzWG9criQfA==
-----END CERTIFICATE-----`
	InvalidClientKey = `-----BEGIN PRIVATE KEY-----
MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQDF/s5GARExNEbQ
ZWxOjgXrJ02AzWAN+ObxpKVZXCRGcBRK+ZjBWec9rHVELlMCySErQ78aYa1AqH/f
bz6/duS2nXKv1vRpeA5BuzT++gF3spCWAvu5r6GSR3zkkTOkzgPzpHWJOWRpkM35
Urzt918o/jJmf/bkLZM/mQjsh8XMD8RRYpzzZeX2by4jdKfsyLyBUnKYgJtf2M4m
hV+13SV5YQcTHqlFnQ0LhgmBokWHVO6GQe7doANlmqF44jlqZHdJGuPxDP6zz1rn
cJWyDB3dnvnXcHO1xbTecAYXaN2wfLNw3PQfEt179XvtWvyKu7EP/+XjId7Ovf0o
IYuQL7vZAgMBAAECggEBAKomTx3ZzOx8AF8Wyfy4EF4FaJVH6UQYol8HHxGsHYBq
0QWdeaivmglmK2Bsbun17os/rPr+9eSa6UkaUNI5WlOU+vohv+jjQ105hFGah6hV
y+sepTTtuev7g1jpb3gxkzPOITPMHn6Z8mhQsgvOifiwep+bWJC+mcwNt52NEG5L
m1jBbDvGDQL3oiSIzMnFd58WZm5lQpTIapcx+lvKtVSn2xb/d0BigCTonZXMUZhL
2y5kcGJSvNaYgrjc0oZDA3gZ4YdoUrMW99CFFeECbbzASp01J+ivIn/Tke1VCHoe
AI4GJFiSFA5jiALkxB1KMIgzaSlg1sGS5VDrlSLu8AECgYEA5qrEtgqTFOuM+Ot7
1QCva5rei8QN8qBczmSB1pZEWtmJNopU1soLOrDdj3sw0l1Up2hSBGiEwcbiTrr+
if3YUZkRRFv5D7BGnQLhe4QcXaANPjsExeQ1NOgaruIVRzaVv4PXhMP0VPErNehl
NKuhVdOI971rbSdG3A2xrELOtqMCgYEA2713xS0/3hepmlO0PHAWsQZjle9zBLPM
SvWqcArrVPRdam4x5yNOnonkGghJ/4G4xWXLHxTNJqshzuGHH9iq8Hk2mQfxMmFO
8l4sTz4d8D6GECGodKov3gDZ6h0BBBR9qv+b5Z3H7jZ2+MD5h9y17g/Kx1NXBnF1
NQNg3Oi0t1MCgYAJayelF0FyNTwIXfUseV6wUh6MLnEzWwDvHIOAs5oO65sCsxtL
uexDdT1Wwnz32f++5i+TJoFlOC29cT07fTX7/vgJhofg8B2yA5AZbweJeyOPSvGi
8vKJOoD8axbbVYs/yq5eKXIslbxh8x9Oy0NHMeAB3aYpStVF3vlGQ2QVaQKBgQDS
eP9ggL/9BaMxK92mSiKh+yGl+o2rwl/6qKZQ3VSdsdZMXDI2V241kpRGjwv5zRHj
GWZeZfk+gYpHc2OPEGRjI2c1WxMfE2+f3K4KVNAuTmTwzJxi6qQgu6X+hTt04f+g
q2ZyoBdhRw/bolMgXDpyRPQQyfXAOSpv1cWQsuBt+wKBgAmGjk+97Y+7eFiHQqKd
AB3SPFZoHO6s7UKDU2Uv7u3NAfQ+/FydwgtfDEcD9aISVZ9DVZJy4/kOSUIc45rG
+XjDdXNcJIN2fNiPMaAimisjxPC83UxrirAJISnHrJwqaCfUwDSArz+Lvf+WlZCL
AxcGuEEhttwyFSfydmYdBaQx
-----END PRIVATE KEY-----`

	ClientCertificate = `-----BEGIN CERTIFICATE-----
MIICrjCCAZYCCQDfb7elrdbtvDANBgkqhkiG9w0BAQUFADAYMRYwFAYDVQQDDA1G
aXJzdCBNLiBMYXN0MCAXDTIwMDMyNDE5NTE0MloYDzMwMDAwNTI2MTk1MTQyWjAY
MRYwFAYDVQQDDA1GaXJzdCBNLiBMYXN0MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8A
MIIBCgKCAQEAyJ3Hr9hJits2ssZZu4ZaZrKxDWHffYPRQNDlNJg8uldXtizjxiyo
0bS+LmiejmyIPRa7BE38ctqoFKPcqJoHy63imWr5plJDbHpacdi1s9FhSsLFCJPe
QK4+GSDu1pOWiqWETomRVz+q0NbYaURT3qZ/YW3cCrRns/+CITz+J3bWHcVBARaq
A+WS/K3dtknTb4On0AeD4SVT5UI19alF3xooh7HG4cp+5j8JtX8CazIrdjFfEUW2
k17a3FiQbyRPYFa52of+59kFd08ABPHwmSeRdUzyhYgn+fYUZ9qUJ9twwgDi50mJ
LDyyp+bGg3exrSy1QQE4rqXuQuPvBBKOiQIDAQABMA0GCSqGSIb3DQEBBQUAA4IB
AQDCiBUdE4chHdUYK2FPrkk72GQiEXTDUP4OZpkJ6mLNCj25j/UC4ND8ByfnsYbx
4rF1Q4JslZZg80NXDSJMR43kfAt+7hnKoYHceCD2FsSlMxP+aOiaa7FbKUSqYnU1
+GGdK13EygiMr1RkVhG8GRngUvE5YzDwcm1jl4pOsShHaX26pititxwKCJSABA3G
1nfrfyHb9N30cN0Z6u4eYpMidO5Txs8Fl0Jh/xJOperl/3Z2ubWKPd1eelhS8GXx
ox8VI+BTty9Yl0fRl12cRsSF1ddOKBJxcdp3Ae2AZZ762vm/ESr1TzrX+wYgLUxp
1I6ycxMDnvvDWmGO0V4HDkPR
-----END CERTIFICATE-----`

	ClientKey = `-----BEGIN PRIVATE KEY-----
MIIEvAIBADANBgkqhkiG9w0BAQEFAASCBKYwggSiAgEAAoIBAQDIncev2EmK2zay
xlm7hlpmsrENYd99g9FA0OU0mDy6V1e2LOPGLKjRtL4uaJ6ObIg9FrsETfxy2qgU
o9yomgfLreKZavmmUkNselpx2LWz0WFKwsUIk95Arj4ZIO7Wk5aKpYROiZFXP6rQ
1thpRFPepn9hbdwKtGez/4IhPP4ndtYdxUEBFqoD5ZL8rd22SdNvg6fQB4PhJVPl
QjX1qUXfGiiHscbhyn7mPwm1fwJrMit2MV8RRbaTXtrcWJBvJE9gVrnah/7n2QV3
TwAE8fCZJ5F1TPKFiCf59hRn2pQn23DCAOLnSYksPLKn5saDd7GtLLVBATiupe5C
4+8EEo6JAgMBAAECggEAJtFo1xyptkWOgu8gY8mualq/KZC7luTPs5P4FcIzVfca
kLSE6k6v58vqVL6Hl5VmkzN3wnB4nZyzkzLVuoX7ZiziQL9TSRx30WCnaYn+NqoY
AkhHqc463hcZCvG1ZS2vnmpCfJPf3JsEKV65Bz1iYR2kXizMvAGGY2zYOCg+IVJk
IJg7TcbrdgA03/IqE2FoPHXuAlAhm5312o+N1h4wXnWRrzUDR9FMOiQCKAUtMuIU
iHMJ7naQjwjJKKsOeDeF1bzgSR4WmZZk5w8fCDdgHFpSn86DZBSfOu6t7kzhOvwR
pu6dxl1t7FPXWtu2jr+/WP0ZtoP2XeKdUMdSlZZSgQKBgQD6Kc5M4TIgjpAOrYMH
1GSD8ccr9KJIX+qy7iDNpzEb38g1Uvsx2OvYLkT1vDhmerRYbai3v+ZU2LBIASzy
gNvbeIRarvEmdoMB7iSTw4D0JJdm+JSf5n3BpBZ+b3IqGNQeBJAmoEv2aCvQOZNe
J+t0FfmgRZDj3paUIZFgTjfj8QKBgQDNTAlJPFUows0Hyd4FWlNhqalLyQWVtVzB
swJPdkyW1hPzZGHnuhDGv2azsraQy2vZUjo8p/zn4mytWng0ya7SzXZlbbdvIU/o
QMraQGIkQDADMwg95Y5R6h0FCxsBBmHaI4YpFtvk+ZvDELv927aXpzgyCj4v/lSc
qSw2ts4MGQKBgG6vcqUXertm+JxV70TWl8a9gleTfP4y2kBjFkaH9DWWFRpq5dPP
W8Kh7kcgCYBmSEdb9aufj8T4vz6Mrpt5ok2ADGenQfG3vA1tled/OB5N1mNsFy6M
qBW2iXFV1BiGNcw2TqWYhSO4QbJ21xpw5T/OvU1JmmsIQG24UH9g/F+xAoGAPtq1
yR9Yr1cc8PKEMD1cY/1O4O4V8KULVh6ZaXy9rDy09QLZ2tmjw0Xcis3/iUtOpMXB
IMsJ6nDvdw/I19ib1tyjECDMVZDsZx5XPQUTRygDyyb3sgOzVC8KXX3t8Z1jnibc
L35ZKrylTM61z95SBBJlaSSrr4P9oc1FxSao5RkCgYBhAh11SGmBntpW02ky8SbW
WUKaIE168cJ4xJmzwAV1OmZbMlc4AhJ9xmkbaViIpIe4Z/R0jp+ZbaFgsgtySER/
ZcJLWYGz00zKINL3Z9FKTa0Opg30AC9kZi9YbJMlsSJMoXx6mTE4kcYHOKv4f1qQ
XL31L67QikBXJnPmtkwxSQ==
-----END PRIVATE KEY-----`
)

func TLSMockServer(mockResponse string) *httptest.Server {

	router := mux.NewRouter()
	router.HandleFunc("/v1/testTLS", func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusOK)
		io.WriteString(rw, mockResponse)
	}).Methods(http.MethodGet)

	uServer := httptest.NewUnstartedServer(router)
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM([]byte(ClientCertificate))
	uServer.TLS = &tls.Config{}
	uServer.TLS.ClientCAs = caCertPool
	uServer.TLS.ClientAuth = tls.RequireAndVerifyClientCert
	uServer.StartTLS()
	return uServer
}

func (t *MockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.f(req)
}

func httpClient(reaction *common.HTTPReaction, checks *common.HTTPExpectations) *MockTransport {
	return &MockTransport{
		f: common.DoHTTP(reaction, checks),
	}
}

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

var _ = Describe("Client", func() {
	Describe("NewClient", func() {
		var settings *Settings

		BeforeEach(func() {
			settings = &Settings{
				User:                 "admin",
				Password:             "admin",
				URL:                  smURL,
				OSBAPIPath:           "/osb",
				NotificationsAPIPath: "/v1/notifications",
				RequestTimeout:       5,
				SkipSSLValidation:    false,
				Transport:            nil,
			}
		})

		When("client is connecting to an mTLS server", func() {
			var mockServer *httptest.Server
			BeforeEach(func() {
				mockServer = TLSMockServer(`{
					"num_items": 1,
					"items" : [{"name" : "hello world"}]
				}`)

				settings = &Settings{
					User:                 "admin",
					Password:             "admin",
					URL:                  "http://example.com",
					OSBAPIPath:           "/osb",
					NotificationsAPIPath: "/v1/notifications",
					RequestTimeout:       5 * time.Second,
					SkipSSLValidation:    true,
					TLSClientKey:         InvalidClientKey,
					TLSClientCertificate: ClientCertificate,
				}
			})

			AfterEach(func() {
				mockServer.Close()
			})

			When("mTLS valid settings are present in config", func() {

				BeforeEach(func() {
					settings.TLSClientKey = ClientKey
					settings.TLSClientCertificate = ClientCertificate
				})

				It("should establish a valid connection to the mTLS server", func() {
					client, err := NewClient(settings)
					Expect(err).ShouldNot(HaveOccurred())
					transport := client.httpClient.Transport.(*BasicAuthTransport)
					_, ok := transport.Rt.(*SSLTransport)
					Expect(ok).To(BeTrue())
					var params map[string]string
					result := make([]*types.ServiceOffering, 0)
					err = client.listAll(context.TODO(), mockServer.URL+"/v1/testTLS", params, &result)
					Expect(result[0].Name).To(Equal("hello world"))
					Expect(err).ShouldNot(HaveOccurred())
				})
			})

			When("mTLS private key does not match pubic key", func() {

				BeforeEach(func() {
					settings.TLSClientKey = ClientKey
					settings.TLSClientCertificate = InvalidClientCertificate
				})

				It("should fail due to not matching keys", func() {
					_, err := NewClient(settings)
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("private key does not match public key"))
				})
			})
			When("bad certificates is used by the client", func() {
				BeforeEach(func() {
					settings.TLSClientCertificate = InvalidClientCertificate
					settings.TLSClientKey = InvalidClientKey
				})

				It("should fail to establish a connection", func() {
					client, err := NewClient(settings)
					Expect(err).ShouldNot(HaveOccurred())
					transport := client.httpClient.Transport.(*BasicAuthTransport)
					_, ok := transport.Rt.(*SSLTransport)
					Expect(ok).To(BeTrue())
					var params map[string]string
					result := make([]*types.ServiceOffering, 0)
					err = client.listAll(context.TODO(), mockServer.URL+"/v1/testTLS", params, &result)
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("bad certificate"))
				})
			})
		})

		Context("when config is invalid", func() {
			It("returns an error", func() {
				settings.User = ""
				_, err := NewClient(settings)

				Expect(err).Should(HaveOccurred())
			})
		})

		Context("when config is valid", func() {
			Context("when transport is present in config", func() {
				It("it uses it as base transport", func() {
					settings.Transport = http.DefaultTransport

					client, err := NewClient(settings)
					Expect(err).ShouldNot(HaveOccurred())
					Expect(client.httpClient.Transport.(*BasicAuthTransport).Rt).To(Equal(http.DefaultTransport))

				})
			})

			Context("when transport is not present in config", func() {
				It("uses a skip ssl transport as base transport", func() {
					client, err := NewClient(settings)
					Expect(err).ShouldNot(HaveOccurred())
					transport := client.httpClient.Transport.(*BasicAuthTransport)
					_, ok := transport.Rt.(*SSLTransport)
					Expect(ok).To(BeTrue())
				})
			})
		})
	})

	const CorrelationIDValue = "corelation-id-value"
	const VisibilitiesPageSize = 100

	type testCase struct {
		expectations *common.HTTPExpectations
		reaction     *common.HTTPReaction

		expectedErr      error
		expectedResponse interface{}
	}

	newClient := func(t *testCase) *ServiceManagerClient {
		client, err := NewClient(&Settings{
			User:                 "admin",
			Password:             "admin",
			URL:                  smURL,
			OSBAPIPath:           "/osb",
			NotificationsAPIPath: "/v1/notifications",
			RequestTimeout:       2 * time.Second,
			SkipSSLValidation:    false,
			VisibilitiesPageSize: VisibilitiesPageSize,
			Transport:            httpClient(t.reaction, t.expectations),
		})
		Expect(err).ShouldNot(HaveOccurred())
		return client
	}

	testContextWithCorrelationID := func(correlationID string) context.Context {
		ctx := context.Background()

		entry := log.C(ctx).WithField(log.FieldCorrelationID, correlationID)
		return log.ContextWithLogger(ctx, entry)
	}

	assertResponse := func(t *testCase, resp interface{}, err error) {
		if t.expectedErr != nil {
			Expect(errors.Cause(err).Error()).To(ContainSubstring(t.expectedErr.Error()))
		} else {
			Expect(err).To(BeNil())
		}

		if t.expectedResponse != nil {
			Expect(resp).To(Equal(t.expectedResponse))
		} else {
			Expect(resp).To(BeNil())
		}
	}

	const okBrokerResponse = `{
		"items": [
		{
			"id": "brokerID",
			"name": "brokerName",
			"description": "Service broker providing some valuable services",
			"broker_url": "https://service-broker-url"
		}
		]
	}`

	clientBrokersResponse := []*types.ServiceBroker{
		{
			Base: types.Base{
				ID: "brokerID",
			},
			Description: "Service broker providing some valuable services",
			Name:        "brokerName",
			BrokerURL:   "https://service-broker-url",
		},
	}

	brokerEntries := []TableEntry{
		Entry("Successfully obtain brokers", testCase{
			expectations: &common.HTTPExpectations{
				URL: path.Join(smURL, web.ServiceBrokersURL),
				Params: map[string]string{
					"fieldQuery": "ready eq true",
				},
				Headers: map[string]string{
					"Authorization":             "Basic " + basicAuth("admin", "admin"),
					log.CorrelationIDHeaders[0]: CorrelationIDValue,
				},
			},
			reaction: &common.HTTPReaction{
				Status: http.StatusOK,
				Body:   okBrokerResponse,
				Err:    nil,
			},
			expectedResponse: clientBrokersResponse,
			expectedErr:      nil,
		}),

		Entry("Returns error when API returns error", testCase{
			expectations: &common.HTTPExpectations{
				URL: path.Join(smURL, web.ServiceBrokersURL),
				Headers: map[string]string{
					"Authorization":             "Basic " + basicAuth("admin", "admin"),
					log.CorrelationIDHeaders[0]: CorrelationIDValue,
				},
			},
			reaction: &common.HTTPReaction{
				Status: http.StatusInternalServerError,
				Err:    fmt.Errorf("error"),
			},
			expectedResponse: nil,
			expectedErr:      fmt.Errorf("error"),
		}),

		Entry("Returns error when API response body is invalid", testCase{
			expectations: &common.HTTPExpectations{
				URL: path.Join(smURL, web.ServiceBrokersURL),
				Headers: map[string]string{
					"Authorization":             "Basic " + basicAuth("admin", "admin"),
					log.CorrelationIDHeaders[0]: CorrelationIDValue,
				},
			},
			reaction: &common.HTTPReaction{
				Status: http.StatusOK,
				Body:   `invalid`,
				Err:    nil,
			},
			expectedResponse: nil,
			expectedErr:      fmt.Errorf("error parsing response body"),
		}),

		Entry("Returns error when API returns error", testCase{
			expectations: &common.HTTPExpectations{
				URL: path.Join(smURL, web.ServiceBrokersURL),
				Headers: map[string]string{
					"Authorization":             "Basic " + basicAuth("admin", "admin"),
					log.CorrelationIDHeaders[0]: CorrelationIDValue,
				},
			},
			reaction: &common.HTTPReaction{
				Status: http.StatusInternalServerError,
				Body:   `{"error":"error"}`,
				Err:    nil,
			},
			expectedResponse: nil,
			expectedErr:      fmt.Errorf("StatusCode: 500 Body: {\"error\":\"error\"}"),
		}),
	}

	DescribeTable("GETBrokers", func(t testCase) {
		client := newClient(&t)
		resp, err := client.GetBrokers(testContextWithCorrelationID(CorrelationIDValue))
		assertResponse(&t, resp, err)
	}, brokerEntries...)

	const okPlanResponse = `{
		"items": [
			 {
				  "created_at": "2018-12-27T09:14:54Z",
				  "updated_at": "2018-12-27T09:14:54Z",
				  "id": "180dd7fb-1c6e-41fe-95ee-aefb51513032",
				  "name": "dummy1plan1",
				  "description": "dummy 1 example plan 1",
				  "catalog_id": "1f400825-1434-5278-9913-dfcf63fcd647",
				  "catalog_name": "dummy1plan1",
				  "free": false,
				  "bindable": true,
				  "plan_updateable": false,
				  "service_offering_id": "47c7790a-3cd1-4520-a030-471f91dc616e"
			 }
		]
	}`

	servicePlans := func(servicePlans string) []*types.ServicePlan {
		c := struct {
			Plans []*types.ServicePlan `json:"items"`
		}{}
		err := json.Unmarshal([]byte(servicePlans), &c)
		if err != nil {
			panic(err)
		}
		return c.Plans
	}

	planEntries := []TableEntry{
		Entry("Successfully obtain plans", testCase{
			expectations: &common.HTTPExpectations{
				URL: path.Join(smURL, web.ServicePlansURL),
				Params: map[string]string{
					"fieldQuery": "ready eq true",
				},
				Headers: map[string]string{
					"Authorization":             "Basic " + basicAuth("admin", "admin"),
					log.CorrelationIDHeaders[0]: CorrelationIDValue,
				},
			},
			reaction: &common.HTTPReaction{
				Status: http.StatusOK,
				Body:   okPlanResponse,
				Err:    nil,
			},
			expectedResponse: servicePlans(okPlanResponse),
			expectedErr:      nil,
		}),

		Entry("Returns error when API returns error", testCase{
			expectations: &common.HTTPExpectations{
				URL: path.Join(smURL, web.ServicePlansURL),
				Headers: map[string]string{
					"Authorization":             "Basic " + basicAuth("admin", "admin"),
					log.CorrelationIDHeaders[0]: CorrelationIDValue,
				},
			},
			reaction: &common.HTTPReaction{
				Status: http.StatusInternalServerError,
				Body:   okPlanResponse,
				Err:    fmt.Errorf("expected error"),
			},
			expectedResponse: nil,
			expectedErr:      fmt.Errorf("expected error"),
		}),
	}

	DescribeTable("GETPlans", func(t testCase) {
		client := newClient(&t)
		resp, err := client.GetPlans(testContextWithCorrelationID(CorrelationIDValue))
		assertResponse(&t, resp, err)
	}, planEntries...)

	const okVisibilityResponse = `{
		"items": [
			 {
				  "id": "127b5b3a-c0bc-45be-bcaf-f1083566214f",
				  "platform_id": "bf092091-76ba-4398-a301-40472b794aea",
				  "service_plan_id": "180dd7fb-1c6e-41fe-95ee-aefb51513032",
				  "labels": {
						"organization_guid": [
							"d0761213-012d-4bc5-8a7b-7780875d8913",
							"15317fc3-693c-423a-90ba-6f86d6559abe"
						],
						"something": ["generic"]
				  },
				  "created_at": "2018-12-27T14:35:23Z",
				  "updated_at": "2018-12-27T14:35:23Z"
			 }
		]
   }`

	serviceVisibilities := func(serviceVisibilities string) []*types.Visibility {
		c := struct {
			Visibilities []*types.Visibility `json:"items"`
		}{}
		err := json.Unmarshal([]byte(serviceVisibilities), &c)
		if err != nil {
			panic(err)
		}
		return c.Visibilities
	}

	visibilitiesEntries := []TableEntry{
		Entry("Successfully obtain visibilities", testCase{
			expectations: &common.HTTPExpectations{
				URL: path.Join(smURL, web.VisibilitiesURL),
				Params: map[string]string{
					"fieldQuery": "ready eq true and service_plan_id in ('180dd7fb-1c6e-41fe-95ee-aefb51513032')",
					"max_items":  strconv.Itoa(VisibilitiesPageSize),
				},
				Headers: map[string]string{
					"Authorization":             "Basic " + basicAuth("admin", "admin"),
					log.CorrelationIDHeaders[0]: CorrelationIDValue,
				},
			},
			reaction: &common.HTTPReaction{
				Status: http.StatusOK,
				Body:   okVisibilityResponse,
				Err:    nil,
			},
			expectedResponse: serviceVisibilities(okVisibilityResponse),
			expectedErr:      nil,
		}),

		Entry("Returns error when API returns error", testCase{
			expectations: &common.HTTPExpectations{
				URL: path.Join(smURL, web.VisibilitiesURL),
				Headers: map[string]string{
					"Authorization":             "Basic " + basicAuth("admin", "admin"),
					log.CorrelationIDHeaders[0]: CorrelationIDValue,
				},
			},
			reaction: &common.HTTPReaction{
				Status: http.StatusInternalServerError,
				Body:   okPlanResponse,
				Err:    fmt.Errorf("expected error"),
			},
			expectedResponse: nil,
			expectedErr:      fmt.Errorf("expected error"),
		}),
	}

	DescribeTable("GETVisibilities", func(t testCase) {
		client := newClient(&t)
		resp, err := client.GetVisibilities(testContextWithCorrelationID(CorrelationIDValue), []string{"180dd7fb-1c6e-41fe-95ee-aefb51513032"})
		assertResponse(&t, resp, err)
	}, visibilitiesEntries...)

	activateBrokerCredentialsEntries := []TableEntry{
		Entry("Successfully activate credentials", testCase{
			expectations: &common.HTTPExpectations{
				Headers: map[string]string{
					"Authorization": "Basic " + basicAuth("admin", "admin"),
				},
				Body: "{}",
			},
			reaction: &common.HTTPReaction{
				Status: http.StatusOK,
				Body:   "{}",
				Err:    nil,
			},
			expectedResponse: "{}",
			expectedErr:      nil,
		}),

		Entry("Returns not found error when given credentials id not exists", testCase{
			expectations: &common.HTTPExpectations{
				Headers: map[string]string{
					"Authorization": "Basic " + basicAuth("admin", "admin"),
				},
				Body: "{}",
			},
			reaction: &common.HTTPReaction{
				Status: http.StatusNotFound,
				Err:    ErrBrokerPlatformCredentialsNotFound,
			},
			expectedResponse: nil,
			expectedErr:      ErrBrokerPlatformCredentialsNotFound,
		}),

		Entry("Returns error when API returns error", testCase{
			expectations: &common.HTTPExpectations{
				Headers: map[string]string{
					"Authorization": "Basic " + basicAuth("admin", "admin"),
				},
				Body: "{}",
			},
			reaction: &common.HTTPReaction{
				Status: http.StatusInternalServerError,
				Err:    fmt.Errorf("expected error"),
			},
			expectedResponse: nil,
			expectedErr:      fmt.Errorf("expected error"),
		}),
	}

	DescribeTable("ActivateCredentials", func(t testCase) {
		client := newClient(&t)
		err := client.ActivateCredentials(testContextWithCorrelationID(CorrelationIDValue), "12345")
		if t.expectedErr != nil {
			Expect(errors.Cause(err).Error()).To(ContainSubstring(t.expectedErr.Error()))
		} else {
			Expect(err).To(BeNil())
		}
		fmt.Print(err)
	}, activateBrokerCredentialsEntries...)
})
