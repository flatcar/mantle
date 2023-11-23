package brightbox

import (
	"fmt"
	"net/url"
	"strings"
)

// APIError can be returned when an API request fails. It provides any error
// messages provided by the API, along with other details about the response.
type APIError struct {
	// StatusCode will hold the HTTP status code from the request that errored
	StatusCode int
	// Status will hold the HTTP status line from the request that errored
	Status string
	// AuthError will hold any available OAuth "error" field contents. See
	// https://api.gb1.brightbox.com/1.0/#errors
	AuthError string `json:"error"`
	// AuthErrorDescription will hold any available OAuth "error_description"
	// field contents. See https://api.gb1.brightbox.com/1.0/#errors
	AuthErrorDescription string `json:"error_description"`
	// ErrorName will hold any available Brightbox API "error_name" field
	// contents. See https://api.gb1.brightbox.com/1.0/#request_errors
	ErrorName string `json:"error_name"`
	// Errors will hold any available Brightbox API "errors" field contents. See
	// https://api.gb1.brightbox.com/1.0/#request_errors
	Errors []string `json:"errors"`
	// ParseError will hold any errors from the JSON parser whilst parsing an
	// API response
	ParseError error
	// RequestURL will hold the full URL used to make the request that errored,
	// if available
	RequestURL *url.URL
	// ResponseBody will hold the raw respose body of the request that errored,
	// if available
	ResponseBody []byte
}

// Error implements the error interface
func (e *APIError) Error() string {
	if e.AuthError != "" {
		return fmt.Sprintf("%s, %s", e.AuthError, e.AuthErrorDescription)
	}
	if e.ErrorName != "" {
		if len(e.Errors) == 0 {
			return e.ErrorName
		}
		return fmt.Sprintf("%s: %s", e.ErrorName, strings.Join(e.Errors, ": "))
	}
	if e.ParseError != nil {
		return fmt.Sprintf("ParseError at %s: %s", e.RequestURL, e.ParseError.Error())
	}
	return fmt.Sprintf("HttpError at %s: %s", e.RequestURL, e.Status)
}

// Unwrap implements the error wrapping interface
// Returns the parse errors from the JSON parser and Unmarshal interface
func (e *APIError) Unwrap() error {
	return e.ParseError
}
