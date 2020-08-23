package types

import (
	"encoding/json"
)

// A Tweet is the decoded form of a single tweet.  The fields marked "default"
// will always be populated by the API; other fields are filled in based on the
// parameters in the request.
type Tweet struct {
	ID   string `json:"id" twitter:"default"`
	Text string `json:"text" twitter:"default"`

	AuthorID       string    `json:"author_id"`
	ConversationID string    `json:"conversation_id"`
	CreatedAt      *Date     `json:"created_at"`
	InReplyTo      string    `json:"in_reply_to_user_id"`
	Language       string    `json:"lang"` // https://tools.ietf.org/html/bcp47
	Location       *Location `json:"geo"`
	Sensitive      bool      `json:"possibly_sensitive"`
	Referenced     []*Ref    `json:"referenced_tweets"`
	Source         string    `json:"source"` // e.g., "Twitter Web App"

	ContextAnnotations []*ContextAnnotation `json:"context_annotations"`
	Withheld           *Withholding         `json:"withheld"`
	Attachments        `json:"attachments"`
	MetricSet
}

// Attachments is a map of attachment type keys to string IDs for objects
// attached to a reply.
type Attachments map[string][]string

// A ContextAnnotation is a collection of domain and/or entity labels, inferred
// based on the text of a tweet.  Context annotations can yield one or many
// domains.
type ContextAnnotation struct {
	Domain *Domain `json:"domain"`
	Entity *Entity `json:"entity"`
}

// A Domain is a single domain label associated with a tweet.
//
// See https://developer.twitter.com/en/docs/twitter-api/annotations for a
// table of defined annotation domains.
type Domain struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// An Entity identifies a programmatically-defined entity annotation associated
// with a particular span of a tweet (see Annotation).
type Entity struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// A Span denotes a span of text associated with an annotation.
type Span struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

// Entities captures annotations and other embedded entities in a text.
type Entities struct {
	Annotations []*Annotation `json:"annotations"`
	CashTags    []*Tag        `json:"cashtags"`
	HashTags    []*Tag        `json:"hashtags"`
	Mentions    []*Tag        `json:"mentions"`
	URLs        []*URL        `json:"urls"`
}

// An Annotation records the location, and type of a programmatic annotation.
type Annotation struct {
	Span
	Probability    float64 `json:"probability"`
	Type           string  `json:"type"`
	NormalizedText string  `json:"normalized_text"`
}

// A Tag denotes a meaningful span of text such as a hashtag (#foo).
type Tag struct {
	Span
	Tag string `json:"tag"`
}

// A URL denotes a span of text encoding a URL.
type URL struct {
	Span
	URL         string `json:"url"`
	Expanded    string `json:"expanded_url"`
	Display     string `json:"display_url"`
	Unwound     string `json:"unwound_url"`
	HTTPStatus  string `json:"status"` // e.g., "200"
	Title       string `json:"title"`
	Description string `json:"description"`
}

// A Location carries the content of a place ("geo"). The payload is encoded as
// GeoJSON, see https://geojson.org. It is captured here as raw JSON.
type Location struct {
	PlaceID     string          `json:"place_id"`
	Coordinates json.RawMessage `json:"coordinates"` // as GeoJSON
}

// Metrics are counter values provided by the API; see MetricSet.
type Metrics map[string]int

// A MetricSet collects the metric types that can be requested from the API.
type MetricSet struct {
	// Metric totals that are available for anyone to access on Twitter, such as
	// number of likes and number of retweets.
	PublicMetrics Metrics `json:"public_metrics"`

	// Metrics totals that are not available for anyone to view on Twitter, such
	// as number of impressions and video view quartiles.
	// Requires OAuth 1.0a User Context authentication.
	NonPublicMetrics Metrics `json:"non_public_metrics"`

	// A grouping of public and non-public metrics attributed to an organic
	// context (posted and viewed in a regular manner).
	// Requires OAuth 1.0a User Context authentication.
	OrganicMetrics Metrics `json:"organic_metrics"`

	// A grouping of public and non-public metrics attributed to a promoted
	// context (posted or viewed as part of an Ads campaign).
	// Requires OAuth 1.0a User Context authentication, and that the tweet was
	// promoted in an Ad.
	//
	// Promoted metrics are NOT included in these counts when a Twitter user is
	// using their own Ads account to promote another Twitter user's tweets.
	//
	// Promoted metrics ARE included in these counts when a Twitter user
	// promotes their own Tweets in an Ads account for a specific handle, the
	// admin for that account may add another Twitter user as an account user so
	// this second account user can promote Tweets for the handle.
	PromotedMetrics Metrics `json:"promoted_metrics"`
}

// A Ref is a reference to another entity, giving its type and ID.
type Ref struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

// Withholding describes content restrictions.
type Withholding struct {
	Copyright    bool     `json:"copyright"`
	CountryCodes []string `json:"country_codes"`
}
