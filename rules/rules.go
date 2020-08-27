// Copyright (C) 2020 Michael J. Fromberger. All Rights Reserved.

// Package rules implements queries for reading and modifying the
// rules used by streaming search queries.
//
// Reading Rules
//
// Use rules.Get to query for existing rules by ID. If no IDs are given, Get
// will return all available rules.
//
//   myRules := rules.Get(id1, id2, id3)
//   allRules := rules.Get()
//
// Invoke the query to fetch the rules:
//
//   rsp, err := allRules.Invoke(ctx, cli)
//
// The Rules field of the response contains the requested rules.
//
// Updating Rules
//
// Each rule update must either add or delete rules, but not both.  Use Add to
// build a Set of rules to add, or Delete to identify a Set of rules to
// delete. For example:
//
//    r, err := rules.Add(rules.Rule{
//       Value: `cat has:images lang:en`,
//       Tag:   "cat pictures in English",
//    })
//
// Once you have a set, you can build a query to Update or Validate.  Update
// applies the rule change; Validate just reports whether the update would have
// succeeded (this corresponds to the "dry_run" parameter in the API):
//
//    apply := rules.Update(r)
//    check := rules.Validate(r)
//
// Invoke the query to execute the change or check:
//
//    rsp, err := apply.Invoke(ctx, cli)
//
// The response will include the updated rules, along with server metadata
// indicating the effective time of application and summary statistics.
//
package rules

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/creachadair/twitter"
	"github.com/creachadair/twitter/types"
)

// Get constructs a query to fetch the specified streaming search rule IDs.  If
// no rule IDs are given, all known rules are fetched.
func Get(ids ...string) Query {
	req := &twitter.Request{Method: "tweets/search/stream/rules"}
	if len(ids) != 0 {
		req.Params = twitter.Params{"ids": ids}
	}
	return Query{request: req}
}

// Update constructs a query to add and/or delete streaming search rules.
func Update(r Set) Query {
	req := &twitter.Request{
		Method:      "tweets/search/stream/rules",
		HTTPMethod:  "POST",
		Data:        bytes.NewReader(r.encoded),
		ContentType: "application/json",
	}
	return Query{request: req}
}

// Validate constructs a query to validate addition and/or deletion of
// streaming search rules, without actually modifying the rules.
func Validate(r Set) Query {
	req := &twitter.Request{
		Method:      "tweets/search/stream/rules",
		HTTPMethod:  "POST",
		Params:      twitter.Params{"dry_run": []string{"true"}},
		Data:        bytes.NewReader(r.encoded),
		ContentType: "application/json",
	}
	return Query{request: req}
}

// A Query performs a rule fetch or update query.
type Query struct {
	request *twitter.Request
}

// Invoke executes the query on the given context and client.
func (q Query) Invoke(ctx context.Context, cli *twitter.Client) (*Reply, error) {
	rsp, err := cli.Call(ctx, q.request)
	if err != nil {
		return nil, err
	}
	out := new(Reply)
	if len(rsp.Data) == 0 {
		// no rules returned
	} else if err := json.Unmarshal(rsp.Data, &out.Rules); err != nil {
		return nil, &twitter.Error{Data: rsp.Data, Message: "decoding rules data", Err: err}
	}
	if err := json.Unmarshal(rsp.Meta, &out.Meta); err != nil {
		return nil, &twitter.Error{Data: rsp.Meta, Message: "decoding rules metadata", Err: err}
	}
	return out, nil
}

// A Rule encodes a single streaming search rule.
type Rule struct {
	ID    string `json:"id,omitempty"`
	Value string `json:"value"`
	Tag   string `json:"tag,omitempty"`
}

// A Reply is the response from a Query.
type Reply struct {
	*twitter.Reply
	Rules []Rule
	Meta  *Meta
}

// Meta records rule set metadata reported by the service.
type Meta struct {
	Sent    *types.Date `json:"sent"`
	Summary struct {
		Created    int `json:"created"`
		NotCreated int `json:"not_created"`
		Deleted    int `json:"deleted"`
		NotDeleted int `json:"not_deleted"`
	} `json:"summary,omitempty"`
}

// A Set encodes a set of rule additions and/or deletions.
type Set struct {
	encoded []byte
}

// Add constructs a set of add rules.
func Add(rules ...Rule) (Set, error) {
	enc, err := json.Marshal(struct {
		A []Rule `json:"add"`
	}{A: rules})
	if err != nil {
		return Set{}, err
	}
	return Set{encoded: enc}, nil
}

// Delete constructs a set of delete rules.
func Delete(ids ...string) (Set, error) {
	type del struct {
		I []string `json:"ids"`
	}
	enc, err := json.Marshal(struct {
		D del `json:"delete"`
	}{
		D: del{I: ids},
	})
	if err != nil {
		return Set{}, err
	}
	return Set{encoded: enc}, nil
}
