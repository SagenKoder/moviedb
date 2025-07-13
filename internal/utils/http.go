package utils

import (
	"net/http"
	"strconv"
)

// GetPathParam extracts a path parameter from the URL using Go 1.22+ ServeMux pattern matching
func GetPathParam(r *http.Request, param string) string {
	return r.PathValue(param)
}

// GetPathParamInt extracts a path parameter and converts it to int
func GetPathParamInt(r *http.Request, param string) (int, error) {
	value := r.PathValue(param)
	return strconv.Atoi(value)
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

// RespondJSON sends a JSON response
func RespondJSON(w http.ResponseWriter, data interface{}, statusCode int) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	// TODO: Implement JSON encoding
	return nil
}

// RespondError sends an error response
func RespondError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	// TODO: Implement error response
}