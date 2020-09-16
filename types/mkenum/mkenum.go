// Copyright (C) 2020 Michael J. Fromberger. All Rights Reserved.

// Program mkenum constructs  type enumerations for the optional fields
// that may be requested in Twitter API v2.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"io"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/creachadair/twitter/types"
)

var _ types.Tweet

var outputPath = flag.String("output", "", "Output path (required)")

func main() {
	flag.Parse()
	if *outputPath == "" {
		log.Fatal("You must provide a non-empty -output path")
	}

	var code bytes.Buffer

	// N.B. The comment suppresses golint checks in this file.
	// See: https://golang.org/s/generatedcode
	fmt.Fprintf(&code, "package types\n// Code generated by %[1]s. DO NOT EDIT.\n\n",
		filepath.Base(os.Args[0]))
	generateEnum(&code, "Tweet", (*types.Tweet)(nil))
	generateSearchableSlice(&code, "Tweet", "ID")
	generateEnum(&code, "User", (*types.User)(nil))
	generateSearchableSlice(&code, "User", "ID", "Username")
	generateEnum(&code, "Media", (*types.Media)(nil))
	generateSearchableSlice(&code, "Media", "Key")
	generateEnum(&code, "Poll", (*types.Poll)(nil))
	generateSearchableSlice(&code, "Poll", "ID")
	generateEnum(&code, "Place", (*types.Place)(nil))
	generateSearchableSlice(&code, "Place", "ID")

	clean, err := format.Source(code.Bytes())
	if err != nil {
		log.Fatalf("Formatting generated code: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(*outputPath), 0700); err != nil {
		log.Fatalf("Creating output directory: %v", err)
	}
	f, err := os.Create(*outputPath)
	if err != nil {
		log.Fatalf("Creating output file: %v", err)
	}
	_, err = f.Write(clean)
	cerr := f.Close()
	if err != nil {
		log.Fatalf("Writing generated code: %v", err)
	}
	if cerr != nil {
		log.Fatalf("Closing output: %v", err)
	}
}

func generateEnum(w io.Writer, base string, v interface{}) {
	typeName := base + "Fields"                    // e.g., TweetFields
	typeLabel := strings.ToLower(base) + ".fields" // e.g., tweet.fields

	fmt.Fprintf(w, "// %s defines optional %s field parameters.\n", typeName, base)
	fmt.Fprintf(w, "type %s struct{\n", typeName)

	fields := fieldKeys(v)

	// Order fields lexicographically by JSON name, for consistency.
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].paramName < fields[j].paramName
	})

	for _, f := range fields {
		fmt.Fprintf(w, "\t%s bool\t// %s\n", f.fieldName, f.paramName)
	}
	fmt.Fprint(w, "}\n\n")

	// Support methods.
	fmt.Fprintf(w, `// Label returns the parameter tag for optional %[1]s fields.
func (%[2]s) Label() string { return %q }

`, base, typeName, typeLabel)

	fmt.Fprintf(w, `// Values returns a slice of the selected field names from f.
func (f %[1]s) Values() []string {
  var values []string
`, typeName)
	for _, f := range fields {
		fmt.Fprintf(w, "\tif f.%[1]s { values=append(values, %[2]q) }\n", f.fieldName, f.paramName)
	}
	fmt.Fprintln(w, "\treturn values\n}")
}

func generateSearchableSlice(w io.Writer, base string, fields ...string) {
	typeName := base + "s"
	recvName := strings.ToLower(base[:1]) + "s"
	fmt.Fprintf(w, "// %s is a searchable slice of %s values.\n", typeName, base)
	fmt.Fprintf(w, "type %s []*%s\n", typeName, base)
	for _, field := range fields {
		funcName := fmt.Sprintf("FindBy%s", field)
		paramName := strings.ToLower(field)

		fmt.Fprintln(w)
		fmt.Fprintf(w, "// %s returns the first %s in %s whose %s matches, or nil.\n",
			funcName, base, recvName, field)
		fmt.Fprintf(w, `func (%[1]s %[2]s) %[4]s(%[5]s string) *%[6]s {
  for _, v := range %[1]s {
    if v.%[3]s == %[5]s {
      return v
    }
  }
  return nil
}
`, recvName, typeName, field, funcName, paramName, base)
	}
}

// fieldInfo records the name and details about a specific field.
type fieldInfo struct {
	fieldName   string
	paramName   string
	userContext bool
}

// fieldKeys returns a map of JSON field keys to the associated struct field
// names, extracted from the struct tags of v, which must be of type *T for
// some struct type T.  This function panics if v's type does not have this
// form.
func fieldKeys(v interface{}) []fieldInfo {
	typ := reflect.TypeOf(v).Elem() // panics if not a pointer
	if typ.Kind() != reflect.Struct {
		panic("pointer target is not a struct")
	}
	var tags []fieldInfo
	for i := 0; i < typ.NumField(); i++ {
		next := typ.Field(i)
		if isDefaultField(next.Tag) {
			continue // default fields do not require enumerators
		}

		name, ok := jsonFieldName(next.Tag)
		if ok {
			tags = append(tags, fieldInfo{
				fieldName:   next.Name,
				paramName:   name,
				userContext: isUserContextField(next.Tag),
			})
			continue
		}

		// If next is an embedded anonymous struct, visit its subfields.
		if next.Anonymous && next.Type.Kind() == reflect.Struct {
			for j := 0; j < next.Type.NumField(); j++ {
				sub := next.Type.Field(j)
				name, ok := jsonFieldName(sub.Tag)
				if ok {
					tags = append(tags, fieldInfo{
						fieldName:   sub.Name,
						paramName:   name,
						userContext: isUserContextField(sub.Tag),
					})
				}
			}
		}
	}
	return tags
}

func jsonFieldName(tag reflect.StructTag) (string, bool) {
	val, ok := tag.Lookup("json")
	if ok {
		name := strings.SplitN(val, ",", 2)[0]
		return name, name != "-"
	}
	return "", false
}

func isDefaultField(tag reflect.StructTag) bool     { return tagHasValue(tag, "default") }
func isUserContextField(tag reflect.StructTag) bool { return tagHasValue(tag, "user-context") }

func tagHasValue(tag reflect.StructTag, value string) bool {
	for _, s := range strings.Split(tag.Get("twitter"), ",") {
		if s == value {
			return true
		}
	}
	return false
}
