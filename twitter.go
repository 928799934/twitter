package twitter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/creachadair/twitter/types"
)

// BaseURL is the default base URL for production Twitter API v2.
const BaseURL = "https://api.twitter.com/2"

// A Client serves as a client for the Twitter API v2.
type Client struct {
	// The HTTP client used to issue requests to the API.
	// If nil, use http.DefaultClient.
	HTTPClient *http.Client

	// If set, this is called prior to issuing the request to the API.  If it
	// reports an error, the request is aborted and the error is returned to the
	// caller.
	Authorize func(*http.Request) error

	// If set, override the base URL for the API v2 endpoint.
	// This is mainly useful for testing.
	BaseURL string
}

func (c *Client) baseURL() (*url.URL, error) {
	if c.BaseURL != "" {
		return url.Parse(c.BaseURL)
	}
	return url.Parse(BaseURL)
}

func (c *Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}

// Start issues the specified API request and returns its HTTP response.  The
// caller is responsible for interpreting any errors or unexpected status codes
// from the request.
//
// Most callers should use c.Call, which handles the complete cycle.
func (c *Client) Start(ctx context.Context, req *Request) (*http.Response, error) {
	u, err := c.baseURL()
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %v", err)
	}
	u.Path = path.Join(u.Path, req.Method)
	req.addQueryTerms(u)

	hreq, err := http.NewRequestWithContext(ctx, req.HTTPMethod, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("invalid request: %v", err)
	}

	if auth := c.Authorize; auth != nil {
		if err := auth(hreq); err != nil {
			return nil, fmt.Errorf("attaching authorization: %v", err)
		}
	}

	return c.httpClient().Do(hreq)
}

// Finish cleans up and decodes a successful (non-nil) HTTP response returned
// by a call to Start.
//
// Most callers should use c.Call, which handles the complete cycle.
func (c *Client) Finish(rsp *http.Response) (*Reply, error) {
	if rsp == nil { // safety check
		panic("cannot Finish a nil *http.Response")
	}

	// The body must be fully read and closed to avoid orphaning resources.
	// See: https://godoc.org/net/http#Do
	var body bytes.Buffer
	io.Copy(&body, rsp.Body)
	rsp.Body.Close()
	if rsp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed: %s", rsp.Status)
	}
	var reply Reply
	if err := json.Unmarshal(body.Bytes(), &reply); err != nil {
		return nil, fmt.Errorf("decoding response body: %v", err)
	}
	return &reply, nil
}

// Call issues the specified API request and returns the decoded reply.
//
// This method is a convenience wrapper for c.Start followed by c.Finish. If
// you need control over the intermediate HTTP response or error handling, you
// can call those methods directly.
func (c *Client) Call(ctx context.Context, req *Request) (*Reply, error) {
	hrsp, err := c.Start(ctx, req)
	if err != nil {
		return nil, err
	}
	return c.Finish(hrsp)
}

// An Authorizer attaches authorization metadata to an outbound request after
// it has been populated with the caller's query but before it is sent to the
// API.  The function modifies the request in-place as needed.
type Authorizer func(*http.Request) error

// BearerTokenAuthorizer returns an authorizer that injects the specified
// bearer token into the Authorization header of each request.
func BearerTokenAuthorizer(token string) Authorizer {
	authValue := "Bearer " + token
	return func(req *http.Request) error {
		req.Header.Add("Authorization", authValue)
		return nil
	}
}

// A Request is the generic format for a Twitter API v2 request.
type Request struct {
	// The fully-expanded method path for the API to call, including parameters.
	// For example: "tweets/12345678".
	Method string

	// Additional request parameters, including optional fields and expansions.
	Params Params

	// The HTTP method to use for the request; if unset the default is "GET".
	HTTPMethod string
}

// Params carries additional request parameters sent in the query URL.
type Params map[string][]string

// Add the given values for the specified parameter, in addition to any
// previously-defined values for that name.
func (p Params) Add(name string, values ...string) {
	if len(values) == 0 {
		return
	}
	p[name] = append(p[name], values...)
}

// Set sets the value of the specified parameter name, removing any
// previously-defined values for that name.
func (p Params) Set(name, value string) { p[name] = []string{value} }

// Reset removes any existing values for the specified parameter.
func (p Params) Reset(name string) { delete(p, name) }

func (p Params) addQueryTerms(query url.Values) {
	for name, values := range p {
		query.Set(name, strings.Join(values, ","))
	}
}

func (req *Request) addQueryTerms(u *url.URL) {
	if len(req.Params) == 0 {
		return // nothing to do
	}
	query := make(url.Values)
	req.Params.addQueryTerms(query)
	u.RawQuery = query.Encode()
}

// A Reply is a wrapper for the reply object returned by successful calls to
// the Twitter API v2.
type Reply struct {
	// The root reply object from the query.
	Data json.RawMessage `json:"data"`

	// For expansions that generate attachments, a map of attachment type to
	// JSON arrays of attachment objects.
	Includes map[string]json.RawMessage `json:"includes,omitempty"`
}

// IncludedMedia decodes any media objects in the includes of r.
// It returns nil without error if there are no media inclusions.
func (r *Reply) IncludedMedia() (types.Medias, error) {
	media, ok := r.Includes["media"]
	if !ok || len(media) == 0 {
		return nil, nil
	}
	var out types.Medias
	if err := json.Unmarshal(media, &out); err != nil {
		return nil, fmt.Errorf("decoding media: %v", err)
	}
	return out, nil
}

// IncludedTweets decodes any tweet objects in the includes of r.
// It returns nil without error if there are no tweet inclusions.
func (r *Reply) IncludedTweets() (types.Tweets, error) {
	tweets, ok := r.Includes["tweets"]
	if !ok || len(tweets) == 0 {
		return nil, nil
	}
	var out types.Tweets
	if err := json.Unmarshal(tweets, &out); err != nil {
		return nil, fmt.Errorf("decoding tweets: %v", err)
	}
	return out, nil
}

// IncludedUsers decodes any user objects in the includes of r.
// It returns nil without error if there are user media inclusions.
func (r *Reply) IncludedUsers() (types.Users, error) {
	users, ok := r.Includes["users"]
	if !ok || len(users) == 0 {
		return nil, nil
	}
	var out types.Users
	if err := json.Unmarshal(users, &out); err != nil {
		return nil, fmt.Errorf("decoding users: %v", err)
	}
	return out, nil
}

// IncludedPolls decodes any poll objects in the includes of r.
// It returns nil without error if there are poll media inclusions.
func (r *Reply) IncludedPolls() (types.Polls, error) {
	polls, ok := r.Includes["polls"]
	if !ok || len(polls) == 0 {
		return nil, nil
	}
	var out types.Polls
	if err := json.Unmarshal(polls, &out); err != nil {
		return nil, fmt.Errorf("decoding polls: %v", err)
	}
	return out, nil
}

// IncludedPlaces decodes any place objects in the includes of r.
// It returns nil without error if there are no place inclusions.
func (r *Reply) IncludedPlaces() (types.Places, error) {
	places, ok := r.Includes["places"]
	if !ok || len(places) == 0 {
		return nil, nil
	}
	var out types.Places
	if err := json.Unmarshal(places, &out); err != nil {
		return nil, fmt.Errorf("decoding places: %v", err)
	}
	return out, nil
}
