package sm

import (
	"fmt"
	"github.com/Peripli/service-manager/pkg/env/envfakes"
	"github.com/fatih/structs"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"time"
)

var _ = Describe("Config", func() {
	var (
		err    error
		config *Settings
	)

	Describe("New", func() {
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
					User:              "admin",
					Password:          "admin",
					Host:              "https://example.com",
					OsbAPI:            "/osb",
					RequestTimeout:    5 * time.Second,
					ResyncPeriod:      5 * time.Minute,
					SkipSSLValidation: true,
					Transport:         nil,
				}
				fakeEnv.UnmarshalReturns(nil)

				fakeEnv.UnmarshalStub = func(value interface{}) error {
					field := structs.New(value).Field("Sm")
					val, ok := field.Value().(*Settings)
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
			config = DefaultSettings()
			config.User = "admin"
			config.Password = "admin"
			config.Host = "https://example.com"
			config.OsbAPI = "/osb"
		})

		Context("when config is valid", func() {
			It("returns no error", func() {
				err = config.Validate()
				Expect(err).To(Not(HaveOccurred()))
			})
		})

		Context("when request timeout is missing", func() {
			It("returns an error", func() {
				config.RequestTimeout = 0
				assertErrorDuringValidate()
			})
		})

		Context("when OSB API is missing", func() {
			It("returns an error", func() {
				config.OsbAPI = ""
				assertErrorDuringValidate()
			})
		})

		Context("when resync period is missing", func() {
			It("returns an error", func() {
				config.ResyncPeriod = 0
				assertErrorDuringValidate()
			})
		})

		Context("when host is missing", func() {
			It("returns an error", func() {
				config.Host = ""
				assertErrorDuringValidate()
			})
		})

		Context("when user is missing", func() {
			It("returns an error", func() {
				config.User = ""
				assertErrorDuringValidate()
			})
		})

		Context("when password is missing", func() {
			It("returns an error", func() {
				config.Password = ""
				assertErrorDuringValidate()
			})
		})
	})
})
