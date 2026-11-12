package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHomepage(t *testing.T) {
	handler := newHandler("OpsPulse", "1.0.0", "testing", "Test Deployment")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()

	expectedValues := []string{
		"OpsPulse",
		"1.0.0",
		"testing",
		"Test Deployment",
		"Application Running",
	}

	for _, value := range expectedValues {
		if !strings.Contains(body, value) {
			t.Errorf("Homepage does not contain expected value: %s", value)
		}
	}
}

func TestHealthEndpoint(t *testing.T) {
	handler := newHandler("OpsPulse", "1.0.0", "testing", "Test Deployment")

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	if strings.TrimSpace(rec.Body.String()) != "healthy" {
		t.Errorf("Expected healthy response, got %q", rec.Body.String())
	}
}

func TestMetricsEndpoint(t *testing.T) {
	handler := newHandler("OpsPulse", "1.0.0", "testing", "Test Deployment")

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()

	expectedMetrics := []string{
		"godeploy_app_info",
		"godeploy_http_requests_total",
		"godeploy_app_uptime_seconds",
		`app_name="OpsPulse"`,
		`version="1.0.0"`,
		`environment="testing"`,
	}

	for _, metric := range expectedMetrics {
		if !strings.Contains(body, metric) {
			t.Errorf("Metrics response does not contain: %s", metric)
		}
	}
}

func TestHomepageRequestCounter(t *testing.T) {
	handler := newHandler("OpsPulse", "1.0.0", "testing", "Test Deployment")

	for i := 1; i <= 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)
	}

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	expected := "godeploy_http_requests_total 3"

	if !strings.Contains(rec.Body.String(), expected) {
		t.Errorf("Expected request counter %q, got:\n%s",
			expected, rec.Body.String())
	}
}

func TestUnknownRoute(t *testing.T) {
	handler := newHandler("OpsPulse", "1.0.0", "testing", "Test Deployment")

	req := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rec.Code)
	}
}

func TestGetEnv(t *testing.T) {
	t.Setenv("GODEPLOY_TEST_ENV", "production")

	value := getEnv("GODEPLOY_TEST_ENV", "development")

	if value != "production" {
		t.Errorf("Expected production, got %s", value)
	}
}

func TestGetEnvFallback(t *testing.T) {
	t.Setenv("GODEPLOY_EMPTY_ENV", "")

	value := getEnv("GODEPLOY_EMPTY_ENV", "development")

	if value != "development" {
		t.Errorf("Expected development fallback, got %s", value)
	}
}

func ExamplegetEnv() {
	fmt.Println(getEnv("NON_EXISTENT_GODEPLOY_VARIABLE", "default"))
	// Output: default
}
