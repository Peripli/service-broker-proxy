package main

import (
	"fmt"

	"github.com/Peripli/service-broker-proxy/pkg/sbproxy"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func main() {
	viper.AddConfigPath("/.")
	viper.SetConfigName("application")
	viper.SetConfigType("yml")
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		logrus.Fatal("Error loading viper file configurations: ", err)
	}
	config := sbproxy.DefaultConfig()
	fmt.Println(config.Reg.User + " " + config.Reg.Password + " " + config.Reg.Host)
}
