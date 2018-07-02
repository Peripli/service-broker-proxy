package main

import (
	"encoding/json"
	"fmt"
)

// ResponseError error send as response json
type ResponseError struct {
	ErrorType   string `json:"error,omitempty"`
	Description string `json:"description,omitempty"`
	statusCode  int
}

// GetStatusCode - return the status code
func (responseError ResponseError) GetStatusCode() int {
	return responseError.statusCode
}

// ToJSON return error as JSON
func (responseError ResponseError) ToJSON() []byte {
	jsonResponse, err := json.Marshal(&responseError)
	if err != nil {
		panic(err)
	}
	return jsonResponse
}

// if the receiver was *ResponseError, then we could only assign err error := &ResponseError{} and not ResponseError{}
// since it is ResponseError, then we can assign both
func (responseError ResponseError) Error() string {
	return fmt.Sprintf("type here always matches the receiver: %T", responseError)
}

func errorWrapper(err error) {

	//var respError *ResponseError
	//	fmt.Printf("Before switch casting: %v %T", err, err)
	//	fmt.Println()

	switch t := err.(type) {
	case *ResponseError:
		fmt.Printf(" %v After switch casting: %T", t, t)
		fmt.Println()
		//respError = t
	case ResponseError:
		fmt.Printf(" %v After switch casting: %T", t, t)
		fmt.Println()
		//respError = &t
	}
}

func custHandler1() error {
	return &ResponseError{
		ErrorType:   "errortype",
		Description: "POINTER",
		statusCode:  123}
}

func custHandler2() error {
	return ResponseError{
		ErrorType:   "errortype",
		Description: "VALUE",
		statusCode:  123}
}

func main() {
	errorWrapper(custHandler1())
	errorWrapper(custHandler2())
}
