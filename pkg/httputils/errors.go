package httputils

import "fmt"

// HTTPErrorResponse is an error type that provides error details compliant with the
// Open Service Broker API conventions
type HTTPErrorResponse struct {
	StatusCode   int    `json:"-"`
	ErrorMessage string `json:"description,omitempty"`
	ErrorKey     string `json:"error,omitempty"`
}

// Error returns a string representation of the HTTPErrorResponse
func (e HTTPErrorResponse) Error() string {
	return fmt.Sprintf("Status: %v; ErrorKey: %v; ErrorMessage: %v", e.StatusCode, e.ErrorKey, e.ErrorMessage)
}
