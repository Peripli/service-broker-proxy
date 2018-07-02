package main

import (
	"encoding/json"
	"fmt"
)

type User struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type WrappedUser struct {
	*User
	Name string `json:"name,omitempty"`
}

func main() {
	u := &WrappedUser{}
	err := json.Unmarshal([]byte(`{
	  "email": "test@example.com",
	  "password": "secret"
	}`), u)
	if err != nil {
		panic(err)
	}

	bytes, err := json.Marshal(u)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(bytes))
}
