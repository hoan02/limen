package limen

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
)

type bodyContextKey struct{}

// normalizePath normalizes the base path to start with a slash.
func normalizePath(basePath string) string {
	if !strings.HasPrefix(basePath, "/") {
		basePath = "/" + basePath
	}

	basePath = strings.TrimSuffix(basePath, "/")
	return basePath
}

// GetBody extracts the parsed JSON body from the request context
func GetJSONBody(req *http.Request) map[string]any {
	if req == nil {
		return nil
	}

	if body, ok := req.Context().Value(bodyContextKey{}).(map[string]any); ok {
		return body
	}

	return nil
}

func queryToMap(r *http.Request) map[string]any {
	q := r.URL.Query()
	out := make(map[string]any, len(q))
	for key, values := range q {
		if len(values) == 1 {
			out[key] = values[0]
		} else {
			out[key] = values
		}
	}
	return out
}

// shouldParseBody checks if the request body should be parsed as JSON
func shouldParseBody(req *http.Request) bool {
	if req.Method != "POST" && req.Method != "PUT" && req.Method != "PATCH" {
		return false
	}
	contentType := req.Header.Get("Content-Type")
	return strings.HasPrefix(contentType, "application/json") && req.Body != nil
}

// parseJSONBody reads and parses the JSON body from the request
// Returns the parsed body map and the original body bytes for restoration
func parseJSONBody(req *http.Request) (map[string]any, []byte, error) {
	defer req.Body.Close()
	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, nil, err
	}

	var body map[string]any
	if err := json.Unmarshal(bodyBytes, &body); err != nil {
		return nil, bodyBytes, err
	}

	return body, bodyBytes, nil
}

// parseAndStoreBody parses the JSON body if needed and stores it in context.
func parseAndStoreBody(req *http.Request) *http.Request {
	if GetJSONBody(req) != nil {
		return req
	}

	if !shouldParseBody(req) {
		return req
	}

	body, bodyBytes, err := parseJSONBody(req)
	if err != nil {
		return req
	}

	req = req.WithContext(context.WithValue(req.Context(), bodyContextKey{}, body))

	// Restore body for handlers that need to read it
	req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	return req
}

func getCurrentRouteFromContext(ctx context.Context) *route {
	if route, ok := ctx.Value(currentRouteContextKey{}).(*route); ok {
		return route
	}
	return nil
}

// ParsePagination reads page/per_page from the request query, applying defaults
// and clamping per_page to MaxPerPage, and converts them to a limit/offset
// QueryOptions.
func ParsePagination(r *http.Request) *QueryOptions {
	q := r.URL.Query()

	perPage := min(queryInt(q.Get("per_page"), DefaultPerPage), MaxPerPage)
	page := queryInt(q.Get("page"), DefaultPage)

	return &QueryOptions{
		Limit:  perPage,
		Offset: (page - 1) * perPage,
	}
}

func queryInt(raw string, fallback int) int {
	if n, err := strconv.Atoi(strings.TrimSpace(raw)); err == nil && n > 0 {
		return n
	}
	return fallback
}
