package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireAdminBasicAuthRequiresConfiguredCredentials(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/admin/submissions", nil)
	recorder := httptest.NewRecorder()

	if RequireAdminBasicAuth(recorder, req) {
		t.Fatalf("RequireAdminBasicAuth returned true, want false")
	}
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
}

func TestRequireAdminBasicAuthRejectsMissingCredentials(t *testing.T) {
	t.Setenv("ADMIN_USERNAME", "admin")
	t.Setenv("ADMIN_PASSWORD", "secret")

	req := httptest.NewRequest(http.MethodGet, "/api/admin/submissions", nil)
	recorder := httptest.NewRecorder()

	if RequireAdminBasicAuth(recorder, req) {
		t.Fatalf("RequireAdminBasicAuth returned true, want false")
	}
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
	if recorder.Header().Get("WWW-Authenticate") == "" {
		t.Fatalf("WWW-Authenticate header is empty")
	}
}

func TestRequireAdminBasicAuthRejectsWrongCredentials(t *testing.T) {
	t.Setenv("ADMIN_USERNAME", "admin")
	t.Setenv("ADMIN_PASSWORD", "secret")

	req := httptest.NewRequest(http.MethodGet, "/api/admin/submissions", nil)
	req.SetBasicAuth("admin", "wrong")
	recorder := httptest.NewRecorder()

	if RequireAdminBasicAuth(recorder, req) {
		t.Fatalf("RequireAdminBasicAuth returned true, want false")
	}
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestRequireAdminBasicAuthAcceptsCredentials(t *testing.T) {
	t.Setenv("ADMIN_USERNAME", "admin")
	t.Setenv("ADMIN_PASSWORD", "secret")

	req := httptest.NewRequest(http.MethodGet, "/api/admin/submissions", nil)
	req.SetBasicAuth("admin", "secret")
	recorder := httptest.NewRecorder()

	if !RequireAdminBasicAuth(recorder, req) {
		t.Fatalf("RequireAdminBasicAuth returned false, want true")
	}
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want default %d before handler writes", recorder.Code, http.StatusOK)
	}
}
