package controllers

import (
	"net/http"
	"strconv"
	"strings"
)

func pathParts(r *http.Request) []string {
	return strings.Split(strings.Trim(r.URL.Path, "/"), "/")
}

func lastPathSegment(r *http.Request) string {
	parts := pathParts(r)
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

func pathSegmentAfter(r *http.Request, marker string) string {
	parts := pathParts(r)
	for i, p := range parts {
		if p == marker && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

func pathIntAfter(r *http.Request, marker string) (int, bool) {
	seg := pathSegmentAfter(r, marker)
	if seg == "" {
		return 0, false
	}
	n, err := strconv.Atoi(seg)
	return n, err == nil
}
