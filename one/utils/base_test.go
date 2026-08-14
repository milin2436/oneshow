package utils

import (
	"net/http/httptest"
	"testing"
)

func TestGetQueryParamByKey(t *testing.T) {
	r := httptest.NewRequest("GET", "/?path=/a/b&flag&empty=&other=a&other=b", nil)

	if got := GetQueryParamByKey(r, "path"); got != "/a/b" {
		t.Fatalf("path = %q, want /a/b", got)
	}
	if got := GetQueryParamByKey(r, "missing"); got != "" {
		t.Fatalf("missing = %q, want empty", got)
	}
	// a flag with no value must not panic
	if got := GetQueryParamByKey(r, "flag"); got != "" {
		t.Fatalf("flag = %q, want empty", got)
	}
	// a key with an empty value must not panic
	if got := GetQueryParamByKey(r, "empty"); got != "" {
		t.Fatalf("empty = %q, want empty", got)
	}
	// the first value of a repeated key is returned
	if got := GetQueryParamByKey(r, "other"); got != "a" {
		t.Fatalf("other = %q, want a", got)
	}

	if got := GetQueryParamByKey(nil, "x"); got != "" {
		t.Fatalf("nil request = %q, want empty", got)
	}
}
