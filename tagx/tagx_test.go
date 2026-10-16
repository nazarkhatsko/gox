package tagx

import (
	"fmt"
	"strings"
	"testing"
)

type options struct {
	Name    string            `key:"name" type:"string" description:"who" validate:"required"`
	Count   int               `key:"count" default:"3"`
	Ratio   float64           `key:"ratio"`
	On      bool              `key:"on"`
	Headers map[string]any    `key:"headers"`
	Tags    []string          `key:"tags"`
	Body    any               `key:"body"`
	Token   string            `key:"token" validate:"required"`
	Ignored string            // no key: not bound
	Extra   map[string]string `key:"extra"`
}

func TestFields(t *testing.T) {
	fields, err := Fields[options]()
	if err != nil {
		t.Fatal(err)
	}

	if len(fields) != 9 {
		t.Fatalf("got %d fields, want 9 (untagged skipped)", len(fields))
	}

	name := fields[0]
	if name.Key != "name" || name.Type != "string" || name.Description != "who" || !name.Required {
		t.Errorf("name = %+v", name)
	}

	if c := fields[1]; c.Type != "int" || c.Default != "3" || c.Required {
		t.Errorf("count = %+v", c)
	}

	if tok := fields[7]; tok.Key != "token" || !tok.Required {
		t.Errorf("token = %+v", tok)
	}

	if got := []string{fields[2].Type, fields[4].Type, fields[5].Type, fields[6].Type}; strings.Join(got, ",") != "float,map,list,any" {
		t.Errorf("inferred types = %v", got)
	}
}

func TestFieldsBadTags(t *testing.T) {
	tests := map[string]error{}

	type typeMismatch struct {
		X string `key:"x" type:"int"`
	}

	type badRule struct {
		X string `key:"x" validate:"requred"`
	}

	type badValidatorRule struct {
		X int `key:"x" validate:"required,nope=3"`
	}

	type badDefault struct {
		X int `key:"x" default:"three"`
	}

	type emptyKey struct {
		X string `key:""`
	}

	_, tests["type mismatch"] = Fields[typeMismatch]()
	_, tests["bad rule"] = Fields[badRule]()
	_, tests["bad validator rule"] = Fields[badValidatorRule]()
	_, tests["bad default"] = Fields[badDefault]()
	_, tests["empty key"] = Fields[emptyKey]()

	for name, err := range tests {
		if err == nil {
			t.Errorf("%s: want an error", name)
		}
	}

	if _, err := Fields[int](); err == nil {
		t.Error("non-struct: want an error")
	}
}

func TestDecode(t *testing.T) {
	got, err := Decode[options](map[string]any{
		"name":    "n",
		"ratio":   2, // an int where a float is wanted
		"on":      true,
		"token":   "t",
		"unknown": "ignored",
		"headers": map[string]any{"A": "1"},
		"tags":    []any{"a", "b"},
		"body":    []any{1},
	})
	if err != nil {
		t.Fatal(err)
	}

	if got.Name != "n" || got.Count != 3 || got.Ratio != 2 || !got.On || got.Token != "t" {
		t.Errorf("scalars = %+v", got)
	}

	if got.Headers["A"] != "1" || len(got.Tags) != 2 || got.Tags[1] != "b" || got.Body == nil {
		t.Errorf("collections = %+v", got)
	}
}

func TestDecodeErrors(t *testing.T) {
	tests := []struct {
		name string
		raw  map[string]any
		want string
	}{
		{"missing required", map[string]any{"token": "t"}, "name is required"},
		{"missing second required", map[string]any{"name": "n"}, "token is required"},
		{"wrong type", map[string]any{"name": 1, "token": "t"}, "name: want string, got int"},
		{"fractional int", map[string]any{"name": "n", "token": "t", "count": 1.5}, "count: want an int, got 1.5"},
		{"bad list item", map[string]any{"name": "n", "token": "t", "tags": []any{"a", 2}}, "tags: [1]: want string, got int"},
		{"map of the wrong type", map[string]any{"name": "n", "token": "t", "extra": map[string]any{"k": "v"}}, "extra: want map, got map[string]interface {}"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Decode[options](tt.raw)
			if err == nil || err.Error() != tt.want {
				t.Fatalf("got %v, want %q", err, tt.want)
			}
		})
	}
}

// rules are options with validator rules besides required.
type rules struct {
	Port  int    `key:"port" validate:"required,min=1,max=65535"`
	Mode  string `key:"mode" validate:"oneof=fast slow"`
	URL   string `key:"url" validate:"omitempty,url"`
	Retry int    `key:"retry" default:"0" validate:"required,gte=0"`
}

func TestValidatorRules(t *testing.T) {
	fields := MustFields[rules]()
	if f := fields[0]; !f.Required || f.Rules != "min=1,max=65535" {
		t.Errorf("port = %+v, want required split from the validator rules", f)
	}

	// Rules run only on set options: mode is absent, so oneof is not checked;
	// retry is 0 from its default, which counts as set.
	got, err := Decode[rules](map[string]any{"port": 8080})
	if err != nil || got.Port != 8080 || got.Retry != 0 {
		t.Fatalf("valid: %+v, %v", got, err)
	}

	tests := []struct {
		name string
		raw  map[string]any
		want string
	}{
		{"min", map[string]any{"port": 0}, "port: 0 fails min=1"},
		{"max", map[string]any{"port": 70000}, "port: 70000 fails max=65535"},
		{"oneof", map[string]any{"port": 1, "mode": "medium"}, "mode: medium fails oneof=fast slow"},
		{"url", map[string]any{"port": 1, "url": "not a url"}, "url: not a url fails url"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Decode[rules](tt.raw)
			if err == nil || err.Error() != tt.want {
				t.Fatalf("got %v, want %q", err, tt.want)
			}
		})
	}
}

func TestCheck(t *testing.T) {
	tests := []struct {
		typ, rules string
		value      any
		want       string
	}{
		{"string", "", "x", ""},
		{"string", "", 1, "want string, got int"},
		{"int", "min=1", 2, ""},
		{"int", "min=1", 0, "0 fails min=1"},
		{"int", "", 1.5, "want an int, got 1.5"},
		{"float", "", 2, ""},
		{"list", "", []any{1}, ""},
		{"map", "", "x", "want map, got string"},
		{"", "min=1", "anything", ""}, // no type: nothing to check
	}
	for _, tt := range tests {
		err := Check(tt.typ, tt.rules, tt.value)
		if got := fmt.Sprint(err); (tt.want == "" && err != nil) || (tt.want != "" && got != tt.want) {
			t.Errorf("Check(%q, %q, %v) = %v, want %q", tt.typ, tt.rules, tt.value, err, tt.want)
		}
	}
}

// response has a nested struct and the kinds Encode must keep or null.
type response struct {
	Status int `key:"status" description:"HTTP status"`
	Body   struct {
		ID   int    `key:"id"`
		Name string `key:"name"`
	} `key:"body" description:"parsed body"`
	Headers map[string]any `key:"headers"`
	Tags    []string       `key:"tags"`
	Raw     any            `key:"raw"`
	Skipped string         // no key
}

func TestNestedFields(t *testing.T) {
	fields := MustFields[response]()

	body := fields[1]
	if body.Key != "body" || body.Type != "object" || body.Description != "parsed body" || len(body.Fields) != 2 || body.Fields[0].Key != "id" || body.Fields[1].Type != "string" {
		t.Errorf("body = %+v", body)
	}
}

func TestDecodeNested(t *testing.T) {
	got, err := Decode[response](map[string]any{"status": 200, "body": map[string]any{"id": 7, "name": "a"}})
	if err != nil || got.Status != 200 || got.Body.ID != 7 || got.Body.Name != "a" {
		t.Fatalf("Decode = %+v, %v", got, err)
	}

	if _, err := Decode[response](map[string]any{"body": map[string]any{"id": "x"}}); err == nil || err.Error() != `body: id: want an int, got "x"` {
		t.Errorf("nested type error = %v", err)
	}

	if _, err := Decode[response](map[string]any{"body": "x"}); err == nil || err.Error() != "body: want object, got string" {
		t.Errorf("not an object = %v", err)
	}
}

func TestEncode(t *testing.T) {
	var r response
	r.Status = 404
	r.Body.ID = 1

	got, err := Encode(r)
	if err != nil {
		t.Fatal(err)
	}

	if got["status"] != 404 || len(got) != 5 {
		t.Errorf("Encode = %#v", got)
	}

	body, ok := got["body"].(map[string]any)
	if !ok || body["id"] != 1 || body["name"] != "" {
		t.Errorf("body = %#v, want a nested map", got["body"])
	}

	// Nil collections are untyped nil: null in JSON, None in Starlark.
	for _, k := range []string{"headers", "tags", "raw"} {
		if v, ok := got[k]; !ok || v != nil {
			t.Errorf("%s = %#v, want an untyped nil", k, v)
		}
	}

	r.Headers = map[string]any{"A": "1"}
	r.Tags = []string{"x"}

	got, _ = Encode(r)
	if h, _ := got["headers"].(map[string]any); h["A"] != "1" {
		t.Errorf("headers = %#v", got["headers"])
	}

	if tags, _ := got["tags"].([]string); len(tags) != 1 {
		t.Errorf("tags = %#v", got["tags"])
	}
}
