package main

import (
	"fmt"

	"github.com/Peripli/service-manager/pkg/env"
	"github.com/spf13/pflag"
)

type SM struct {
	Host int
}

type Props struct {
	Sm SM
}

func main() {
	props := &Props{}
	//os.Setenv("SM_HOST", "122a3")
	env := env.Default("")
	pflag.String("sm.host", "a1", "help message for flagname")
	//FLAG gotta be set before env loading
	env.Load()
	//env.Viper.BindPFlag("sm.host", pflag.Lookup("sm.host"))
	pflag.Parse()
	err := env.Unmarshal(props)
	if err != nil {
		panic(err)
	}
	fmt.Println(props.Sm.Host)

}
