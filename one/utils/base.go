package utils

import (
	"net/http"
)

// GetQueryParamByKey get param for get method
func GetQueryParamByKey(r *http.Request, key string) string {
	if r == nil || r.URL == nil {
		return ""
	}
	keys, ok := r.URL.Query()[key]
	if !ok || len(keys) == 0 || keys[0] == "" {
		return ""
	}
	return keys[0]
}
