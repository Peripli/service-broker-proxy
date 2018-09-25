package sbproxy

import (
	"fmt"
	"github.com/Peripli/service-broker-proxy/pkg/sm"
	"github.com/Peripli/service-manager/pkg/env/envfakes"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/server"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"time"
	"github.com/Peripli/service-broker-proxy/pkg/sbproxy/reconcile"
)

var _ = Describe("Config", func() {

	var (
		config *Settings
		err    error
	)

	Describe("NewSettings", func() {
		var (
			fakeEnv       *envfakes.FakeEnvironment
			creationError = fmt.Errorf("creation error")
		)

		assertErrorDuringNewConfiguration := func() {
			_, err := NewSettings(fakeEnv)
			Expect(err).To(HaveOccurred())
		}

		BeforeEach(func() {
			fakeEnv = &envfakes.FakeEnvironment{}
		})

		Context("when unmarshaling from environment fails", func() {
			It("returns an error", func() {
				fakeEnv.UnmarshalReturns(creationError)

				assertErrorDuringNewConfiguration()
			})
		})

		Context("when unmarshalling is successful", func() {
			var (
				validSettings *Settings
				emptySettings *Settings
			)

			BeforeEach(func() {
				emptySettings = &Settings{}
				validSettings = &Settings{
					Server: &server.Settings{
						Port:            8080,
						RequestTimeout:  5 * time.Second,
						ShutdownTimeout: 5 * time.Second,
					},
					Log: &log.Settings{
						Level:  "debug",
						Format: "text",
					},
					Sm: &sm.Settings{
						User:              "admin",
						Password:          "admin",
						Host:              "https://sm.com",
						OsbAPI:            "/osb",
						RequestTimeout:    5 * time.Second,
						ResyncPeriod:      5 * time.Minute,
						SkipSSLValidation: true,
					},
					Reconcile: &reconcile.Settings{
						Host: "https://selfhost.com",
					},
				}
				fakeEnv.UnmarshalReturns(nil)

				fakeEnv.UnmarshalStub = func(value interface{}) error {
					val, ok := value.(*Settings)
					if ok {
						*val = *config
					}
					return nil
				}
			})
			Context("when loaded from environment", func() {
				JustBeforeEach(func() {
					config = validSettings
				})

				It("uses the config values from env", func() {
					c, err := NewSettings(fakeEnv)

					Expect(err).To(Not(HaveOccurred()))
					Expect(fakeEnv.UnmarshalCallCount()).To(Equal(1))

					Expect(err).To(Not(HaveOccurred()))

					Expect(c).Should(Equal(validSettings))
				})
			})

			Context("when missing from environment", func() {
				JustBeforeEach(func() {
					config = emptySettings
				})

				It("returns an empty config", func() {
					c, err := NewSettings(fakeEnv)

					Expect(err).To(Not(HaveOccurred()))
					Expect(fakeEnv.UnmarshalCallCount()).To(Equal(1))

					Expect(c).Should(Equal(emptySettings))
				})
			})
		})
	})

	Describe("Validate", func() {
		assertErrorDuringValidate := func() {
			err = config.Validate()
			Expect(err).To(HaveOccurred())
		}

		BeforeEach(func() {
			config = &Settings{
				Server: &server.Settings{
					Port:            8080,
					RequestTimeout:  5 * time.Second,
					ShutdownTimeout: 5 * time.Second,
				},
				Log: &log.Settings{
					Level:  "debug",
					Format: "text",
				},
				Sm: &sm.Settings{
					User:              "admin",
					Password:          "admin",
					Host:              "https://sm.com",
					OsbAPI:            "/osb",
					RequestTimeout:    5 * time.Second,
					ResyncPeriod:      5 * time.Minute,
					SkipSSLValidation: true,
				},
				Reconcile: &reconcile.Settings{
					Host: "https://selfhost.com",
				},
			}
		})
		Context("when selfhost is missing", func() {
			It("returns an error", func() {
				config.Reconcile.Host = ""
				assertErrorDuringValidate()
			})
		})

		Context("when server config is invalid", func() {
			It("returns an error", func() {
				config.Server.RequestTimeout = 0
				assertErrorDuringValidate()
			})
		})

		Context("when log config is invalid", func() {
			It("returns an error", func() {
				config.Log.Format = ""
				assertErrorDuringValidate()
			})
		})

		Context("when sm config is invalid", func() {
			It("returns an error", func() {
				config.Sm.Host = ""
				assertErrorDuringValidate()
			})
		})
	})
})
