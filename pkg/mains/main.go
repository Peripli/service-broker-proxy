package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
)

type Message struct {
	Id   int    `json:"id"`
	Name string `json:"name"` // struct to json omit empty fields , make it *string to differenciate null and 0
}

func (m Message) String() string {
	return strconv.Itoa(m.Id) + " " + m.Name
}

// curl localhost:8000 -d '{"name":"Hello"}'
//unmarshal from fully loaded in memory string (byte array)
func WithUnmarshal(w http.ResponseWriter, r *http.Request) {

	// Read body
	b, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Unmarshal
	var msg Message
	err = json.Unmarshal(b, &msg)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	fmt.Println("After unmrashaling " + msg.String())

	output, err := json.Marshal(msg)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	fmt.Println("After marshaling " + string(output))
	w.Header().Set("content-type", "application/json")
	w.Write(output)
}

// Decode - from stream to json + unmarshal to struct and encode - from go value + marshal to json to stream
func WithDecode(w http.ResponseWriter, r *http.Request) {
	var msg Message
	if r.Body == nil {
		http.Error(w, "Please send a request body", 400)
		return
	}

	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	fmt.Println("After decoding " + msg.String())
	if err := json.NewEncoder(w).Encode(msg); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
}

func main123() {
	http.HandleFunc("/unmarshal", WithUnmarshal)
	http.HandleFunc("/decode", WithDecode)
	address := ":8080"
	log.Println("Starting sbproxy on address", address)
	err := http.ListenAndServe(address, nil)
	if err != nil {
		panic(err)
	}
}
