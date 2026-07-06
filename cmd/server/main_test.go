package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestRouterRegistersCoursePageInfoEndpoint(t *testing.T) {
	if err := os.Chdir("../.."); err != nil {
		t.Fatalf("chdir to repo root: %v", err)
	}

	router := newRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/courses/pageinfo", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code == http.StatusNotFound {
		t.Fatalf("GET /api/courses/pageinfo returned 404; route is not registered")
	}
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("GET /api/courses/pageinfo status = %d, want %d", resp.Code, http.StatusUnauthorized)
	}
}
