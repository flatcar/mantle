package brightbox

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"golang.org/x/oauth2"
)

//go:generate ./generate_default_functions paths.yaml

// Client represents a connection to the Brightbox API. You should use NewConnect
// to allocate and configure Clients, and pass in either a
// clientcredentials or password configuration.
type Client struct {
	UserAgent      string
	baseURL        *url.URL
	client         *http.Client
	tokensource    oauth2.TokenSource
	hardcoreDecode bool
}

// ResourceBaseURL returns the base URL within the client
func (q *Client) ResourceBaseURL() *url.URL {
	return q.baseURL
}

// HTTPClient returns the current HTTP structure within the client
func (q *Client) HTTPClient() *http.Client {
	return q.client
}

// ExtractTokenID implements the AuthResult interface for gophercloud clients
func (q *Client) ExtractTokenID() (string, error) {
	token, err := q.tokensource.Token()
	if err != nil {
		return "", err
	}
	return token.AccessToken, nil
}

// AllowUnknownFields stops the Client generating an error is an unsupported field is
// returned by the API.
func (q *Client) AllowUnknownFields() {
	q.hardcoreDecode = false
}

// DisallowUnknownFields causes the Client to generate an error if an unsupported field is
// returned by the API.
func (q *Client) DisallowUnknownFields() {
	q.hardcoreDecode = true
}

// apiGet makes a GET request to the API
// and decoding any JSON response.
//
// relURL is the relative path of the endpoint to the base URL, e.g. "servers".
func apiGet[O any](
	ctx context.Context,
	q *Client,
	relURL string,
) (*O, error) {
	return apiCommand[O](ctx, q, "GET", relURL)
}

// apiGetCollection makes a GET request to the API
// and decoding any JSON response into an appropriate slice
//
// relURL is the relative path of the endpoint to the base URL, e.g. "servers".
func apiGetCollection[S ~[]O, O any](
	ctx context.Context,
	q *Client,
	relURL string,
) (S, error) {
	collection, err := apiGet[S](ctx, q, relURL)
	if collection == nil {
		return nil, err
	}
	return *collection, err
}

// apiPost makes a POST request to the API, JSON encoding any given data
// and decoding any JSON response.
//
// relURL is the relative path of the endpoint to the base URL, e.g. "servers".
//
// if reqBody is non-nil, it will be Marshaled to JSON and set as the request
// body.
func apiPost[O any](
	ctx context.Context,
	q *Client,
	relURL string,
	reqBody interface{},
) (*O, error) {
	return apiObject[O](ctx, q, "POST", relURL, reqBody)
}

// apiPut makes a PUT request to the API, JSON encoding any given data
// and decoding any JSON response.
//
// relURL is the relative path of the endpoint to the base URL, e.g. "servers".
//
// if reqBody is non-nil, it will be Marshaled to JSON and set as the request
// body.
func apiPut[O any](
	ctx context.Context,
	q *Client,
	relURL string,
	reqBody interface{},
) (*O, error) {
	return apiObject[O](ctx, q, "PUT", relURL, reqBody)
}

// apiDelete makes a DELETE request to the API
//
// relURL is the relative path of the endpoint to the base URL, e.g. "servers".
func apiDelete[O any](
	ctx context.Context,
	q *Client,
	relURL string,
) (*O, error) {
	return apiCommand[O](ctx, q, "DELETE", relURL)
}

func apiObject[O any](
	ctx context.Context,
	q *Client,
	method string,
	relURL string,
	reqBody interface{},
) (*O, error) {
	req, err := jsonRequest(ctx, q, method, relURL, reqBody)
	if err != nil {
		return nil, err
	}
	res, err := q.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	return jsonResponse[O](res, q.hardcoreDecode)
}

func apiCommand[O any](
	ctx context.Context,
	q *Client,
	method string,
	relURL string,
) (*O, error) {
	return apiObject[O](ctx, q, method, relURL, nil)
}

func jsonResponse[O any](res *http.Response, hardcoreDecode bool) (*O, error) {
	if res.StatusCode >= 200 && res.StatusCode <= 299 {
		decode := json.NewDecoder(res.Body)
		if hardcoreDecode {
			decode.DisallowUnknownFields()
		}
		result := new(O)
		err := decode.Decode(result)
		if err != nil {
			var unmarshalError *json.UnmarshalTypeError
			if errors.As(err, &unmarshalError) {
				unmarshalError.Offset = decode.InputOffset()
			}
			return nil, &APIError{
				RequestURL: res.Request.URL,
				StatusCode: res.StatusCode,
				Status:     res.Status,
				ParseError: err,
			}
		}
		if decode.More() {
			return nil, &APIError{
				RequestURL: res.Request.URL,
				StatusCode: res.StatusCode,
				Status:     res.Status,
				ParseError: fmt.Errorf("Response body has additional unparsed data at position %d", decode.InputOffset()+1),
			}
		}
		return result, err
	}
	return nil, newAPIError(res)
}

func newAPIError(res *http.Response) *APIError {
	if res.StatusCode >= 200 && res.StatusCode <= 299 {
		return nil
	}
	apierr := APIError{
		RequestURL: res.Request.URL,
		StatusCode: res.StatusCode,
		Status:     res.Status,
	}
	var body []byte
	body, apierr.ParseError = io.ReadAll(res.Body)

	if len(body) > 0 {
		err := json.Unmarshal(body, &apierr)
		apierr.ParseError = err
	}
	apierr.ResponseBody = body
	return &apierr
}

func jsonRequest(ctx context.Context, q *Client, method string, relURL string, body interface{}) (*http.Request, error) {
	absUrl, err := q.baseURL.Parse(relURL)
	if err != nil {
		return nil, err
	}
	absUrl.RawQuery = q.baseURL.RawQuery
	buf, err := jsonReader(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, absUrl.String(), buf)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")

	if q.UserAgent != "" {
		req.Header.Add("User-Agent", q.UserAgent)
	}
	return req, nil
}

func jsonReader(from interface{}) (io.Reader, error) {
	var buf bytes.Buffer
	if from == nil {
		return &buf, nil
	}
	err := json.NewEncoder(&buf).Encode(from)
	if err != nil {
		return nil, err
	}
	return &buf, nil
}
