package utils

import (
	"net/http"
	"strconv"
)

// GetPathParam extracts a path parameter from the URL using Go 1.22+ ServeMux pattern matching
func GetPathParam(r *http.Request, param string) string {
	return r.PathValue(param)
}

// GetQueryParam gets a query parameter with optional default value
func GetQueryParam(r *http.Request, param, defaultValue string) string {
	value := r.URL.Query().Get(param)
	if value == "" {
		return defaultValue
	}
	return value
}

// GetQueryParamInt gets a query parameter as int with optional default value
func GetQueryParamInt(r *http.Request, param string, defaultValue int) int {
	value := r.URL.Query().Get(param)
	if value == "" {
		return defaultValue
	}

	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return intValue
}
