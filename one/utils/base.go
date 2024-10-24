package utils

import (
	"net/http"
)

//http base

//GetQueryParamByKey get param for get method
func GetQueryParamByKey(r *http.Request, key string) string {

	keys, ok := r.URL.Query()[key]
	if !ok || len(keys[0]) < 1 {
		return ""
	}

	return keys[0]
}
