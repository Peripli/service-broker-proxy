package main

import (
	"os"

	"fmt"
	"strings"

	"os/exec"

	"log"

	"github.com/Peripli/service-broker-proxy/pkg/sm"
	"github.com/spf13/viper"
)

func ExampleSimpleEnvVar() {
	os.Setenv("FOO", "bar")
	viper.AutomaticEnv()
	fmt.Printf("foo=%v\n", viper.Get("foo"))
	// Output: foo=bar
}

func ExampleNestedValue() {
	os.Setenv("FOO_BAR", "blabla")
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)
	viper.SetDefault("foo.bar", "default")
	viper.AutomaticEnv()
	fmt.Printf("foo.bar=%v\n", viper.Get("foo.bar"))
	// Output: foo.bar=blabla
}

func ExampleNestedValue2() {
	os.Setenv("FOO_BAR", "bblabla")
	os.Setenv("FOO_CAR", "cblabla")
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)
	//viper.SetDefault("foo.bar", "default")
	viper.AutomaticEnv()
	viper.Set("foo.rar", "rasd")
	sub := viper.Sub("foo")
	if sub == nil {

	} else {
		sub.AutomaticEnv()
		sub.SetEnvPrefix("foo")
	}
	fmt.Printf("foo.bar=%v\n", sub.Get("bar"))
	// Output: foo.bar=blabla
}

func main() {
	os.Setenv("PROXY_SM_HOST", "envURI")
	viper.Set("sm.user", "overrideUser")
	//ExampleSimpleEnvVar()
	//ExampleNestedValue()
	//ExampleNestedValue2()
	//ExampleNestedValue3()
	sm.DefaultConfig()

}

func ExampleNestedValue3() {
	os.Setenv("FOO_BAR", "bblabla")
	os.Setenv("WAR", "WAARR")
	//viper.Set("foo.rar", "rasd")
	viper.SetConfigName("app")
	viper.SetConfigType("yml")

	out, err := exec.Command("pwd").Output()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("The date is %s\n", out)

	viper.AddConfigPath(".")
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)
	viper.AutomaticEnv()
	//viper.SetDefault("foo.bar", "default")
	//viper.SetDefault("war", "default")
	//viper.SetDefault("foo.bar", "fdefault")
	if err := viper.ReadInConfig(); err != nil {
		panic(err)
	}
	//viper.SetEnvPrefix("SM")
	//sub := "foo"
	//if sub == nil {
	//
	//} else {
	//	sub.AutomaticEnv()
	//	sub.SetEnvPrefix("foo")
	//}
	fmt.Printf("foo.bar=%v\n", viper.Get("foo"+".bar"))

	var h = &Holder{}
	viper.Unmarshal(h)
	fmt.Println(h.Foo.Bar)
	fmt.Println(h.War)
	// Output: foo.bar=blabla
}

type Holder struct {
	Foo Foo
	War string
}

type Foo struct {
	Bar string
}
