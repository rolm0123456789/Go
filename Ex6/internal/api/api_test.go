package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"urlwatch/internal/checker"
	"urlwatch/internal/domain"
	"urlwatch/internal/store"
)

func newTestAPI() (*API, *store.MemoryStore, *checker.MockChecker) {
	ms := store.NewMemoryStore()
	mc := checker.NewMockChecker(nil)
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil)) // Logger muet pour les tests
	return NewAPI(ms, mc, logger), ms, mc
}

func TestHealthz(t *testing.T) {
	api, _, _ := newTestAPI()
	req := httptest.NewRequest("GET", "/healthz", nil)
	rec := httptest.NewRecorder()

	api.Healthz(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Attendu statut 200, obtenu %d", rec.Code)
	}

	body := strings.TrimSpace(rec.Body.String())
	if body != `{"status":"ok"}` {
		t.Errorf("Attendu body '{\"status\":\"ok\"}', obtenu '%s'", body)
	}
}

func TestCreateCheckBatch_Success(t *testing.T) {
	apiInstance, ms, mc := newTestAPI()
	
	// Simulation de la réponse HTTP de go.dev
	mc.Responses["https://go.dev"] = domain.CheckResult{
		URL:        "https://go.dev",
		StatusCode: 200,
		OK:         true,
		LatencyMS:  10,
	}

	payload := `{
		"urls": ["https://go.dev"],
		"options": {
			"concurrency": 2,
			"timeout_ms": 1000
		}
	}`

	req := httptest.NewRequest("POST", "/v1/checks", bytes.NewBufferString(payload))
	rec := httptest.NewRecorder()

	// Configuration du routeur pour assurer le bon typage et contexte
	mux := http.NewServeMux()
	apiInstance.RegisterRoutes(mux)

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("Attendu statut 201 Created, obtenu %d. Corps: %s", rec.Code, rec.Body.String())
	}

	var resp domain.Batch
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Impossible de décoder la réponse JSON : %v", err)
	}

	if resp.ID == "" {
		t.Error("ID de lot vide")
	}
	if resp.Summary.Total != 1 || resp.Summary.Up != 1 || resp.Summary.Down != 0 {
		t.Errorf("Résumé invalide: %+v", resp.Summary)
	}

	// Vérification de la persistance dans le store
	saved, err := ms.Get(context.Background(), resp.ID)
	if err != nil {
		t.Fatalf("Le lot n'a pas été enregistré dans le store : %v", err)
	}
	if saved.ID != resp.ID {
		t.Errorf("Attendu lot sauvegardé avec ID %s, obtenu %s", resp.ID, saved.ID)
	}
}

func TestCreateCheckBatch_ValidationError(t *testing.T) {
	apiInstance, _, _ := newTestAPI()

	tests := []struct {
		name           string
		payload        string
		expectedStatus int
		expectedCode   string
	}{
		{
			name:           "URLs vides",
			payload:        `{"urls": []}`,
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "invalid_request",
		},
		{
			name:           "URL invalide",
			payload:        `{"urls": ["ftp://invalide.com"]}`,
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "invalid_request",
		},
		{
			name:           "Concurrence trop élevée",
			payload:        `{"urls": ["https://go.dev"], "options": {"concurrency": 99}}`,
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "invalid_request",
		},
		{
			name:           "Timeout unitaire hors limites",
			payload:        `{"urls": ["https://go.dev"], "options": {"timeout_ms": 50}}`,
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "invalid_request",
		},
	}

	mux := http.NewServeMux()
	apiInstance.RegisterRoutes(mux)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/v1/checks", bytes.NewBufferString(tt.payload))
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("Attendu statut %d, obtenu %d", tt.expectedStatus, rec.Code)
			}

			var errResp ErrorResponse
			_ = json.Unmarshal(rec.Body.Bytes(), &errResp)
			if errResp.Error.Code != tt.expectedCode {
				t.Errorf("Attendu code d'erreur '%s', obtenu '%s'", tt.expectedCode, errResp.Error.Code)
			}
		})
	}
}

func TestGetCheckBatch_NotFound(t *testing.T) {
	apiInstance, _, _ := newTestAPI()

	req := httptest.NewRequest("GET", "/v1/checks/b_inexistant", nil)
	rec := httptest.NewRecorder()

	mux := http.NewServeMux()
	apiInstance.RegisterRoutes(mux)

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("Attendu statut 404, obtenu %d", rec.Code)
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("Impossible de décoder le JSON : %v", err)
	}
	if errResp.Error.Code != "batch_not_found" {
		t.Errorf("Attendu code 'batch_not_found', obtenu '%s'", errResp.Error.Code)
	}
}

func TestMethodNotAllowed_JSONError(t *testing.T) {
	apiInstance, _, _ := newTestAPI()

	mux := http.NewServeMux()
	apiInstance.RegisterRoutes(mux)
	handler := MethodNotAllowedMiddleware(mux)

	req := httptest.NewRequest("DELETE", "/v1/checks/b_test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("Attendu statut 405, obtenu %d", rec.Code)
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("Impossible de décoder le JSON : %v", err)
	}
	if errResp.Error.Code != "method_not_allowed" {
		t.Errorf("Attendu code 'method_not_allowed', obtenu '%s'", errResp.Error.Code)
	}
}
