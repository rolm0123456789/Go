package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"runtime/debug"
	"strings"
	"time"

	"urlwatch/internal/domain"
	"urlwatch/internal/pool"

	"github.com/google/uuid"
)

// ============================================================================
// DTOs ET CONFIGURATION
// ============================================================================

type OptionsDTO struct {
	Concurrency int `json:"concurrency,omitempty"`
	TimeoutMS   int `json:"timeout_ms,omitempty"`
}

type CheckRequest struct {
	URLs    []string    `json:"urls"`
	Options OptionsDTO  `json:"options"`
}

type ErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type API struct {
	store   domain.Store
	checker domain.Checker
	logger  *slog.Logger
}

// NewAPI initialise l'API avec ses dépendances.
func NewAPI(store domain.Store, checker domain.Checker, logger *slog.Logger) *API {
	return &API{
		store:   store,
		checker: checker,
		logger:  logger,
	}
}

// ============================================================================
// HELPERS ET ERREURS
// ============================================================================

func writeError(w http.ResponseWriter, code string, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	resp := ErrorResponse{}
	resp.Error.Code = code
	resp.Error.Message = message
	
	_ = json.NewEncoder(w).Encode(resp)
}

// validateRequest valide les URLs et initialise les valeurs par défaut pour les options.
func validateRequest(req *CheckRequest) error {
	if len(req.URLs) == 0 {
		return &domain.ValidationError{Field: "urls", Message: "la liste des URLs ne peut pas etre vide"}
	}
	if len(req.URLs) > 100 {
		return &domain.ValidationError{Field: "urls", Message: "le nombre maximum d'URLs autorise est 100"}
	}

	// Validation du format des URLs
	for _, uStr := range req.URLs {
		parsed, err := url.ParseRequestURI(uStr)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return &domain.ValidationError{Field: "urls", Message: fmt.Sprintf("l'URL '%s' est invalide ou n'utilise pas le protocole http/https", uStr)}
		}
	}

	// Valeurs par défaut et bornes de parallélisme
	if req.Options.Concurrency <= 0 {
		req.Options.Concurrency = 8
	} else if req.Options.Concurrency > 50 {
		return &domain.ValidationError{Field: "options.concurrency", Message: "la concurrence maximum est bornee a 50"}
	}

	// Valeurs par défaut et bornes de timeout
	if req.Options.TimeoutMS <= 0 {
		req.Options.TimeoutMS = 5000
	} else if req.Options.TimeoutMS < 100 || req.Options.TimeoutMS > 30000 {
		return &domain.ValidationError{Field: "options.timeout_ms", Message: "le timeout_ms doit etre compris entre 100 et 30000 ms"}
	}

	return nil
}

// ============================================================================
// HANDLERS HTTP
// ============================================================================

// RegisterRoutes configure et enregistre toutes les routes HTTP.
func (api *API) RegisterRoutes(mux *http.ServeMux) {
	// Go 1.22+ permet de spécifier la méthode dans le chemin
	mux.HandleFunc("POST /v1/checks", api.CreateCheckBatch)
	mux.HandleFunc("GET /v1/checks/{id}", api.GetCheckBatch)
	mux.HandleFunc("GET /healthz", api.Healthz)
}

// GET /healthz
func (api *API) Healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

// POST /v1/checks
func (api *API) CreateCheckBatch(w http.ResponseWriter, r *http.Request) {
	var req CheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid_request", "JSON mal forme ou corps de requete manquant", http.StatusBadRequest)
		return
	}

	// Validation du DTO
	if err := validateRequest(&req); err != nil {
		var valErr *domain.ValidationError
		if errors.As(err, &valErr) {
			writeError(w, "invalid_request", valErr.Error(), http.StatusBadRequest)
		} else {
			writeError(w, "invalid_request", err.Error(), http.StatusBadRequest)
		}
		return
	}

	// Mesure du temps total de traitement du lot
	startTime := time.Now()

	// Contexte global pour le lot d'URLs
	// Note : Le timeout_ms s'applique par URL (pool unitaire), mais pour le lot global
	// on ajoute une marge de sécurité de 5 secondes pour le réseau/scheduler.
	globalTimeout := time.Duration(req.Options.TimeoutMS)*time.Millisecond + 5*time.Second
	ctx, cancel := context.WithTimeout(r.Context(), globalTimeout)
	defer cancel()

	// Exécution concurrente via le Worker Pool
	results := pool.RunBatch(ctx, req.URLs, req.Options.Concurrency, time.Duration(req.Options.TimeoutMS)*time.Millisecond, api.checker)

	totalDuration := time.Since(startTime).Milliseconds()

	// Agrégation du résumé
	upCount := 0
	downCount := 0
	for _, res := range results {
		if res.OK {
			upCount++
		} else {
			downCount++
		}
	}

	summary := domain.BatchSummary{
		Total:      len(results),
		Up:         upCount,
		Down:       downCount,
		DurationMS: totalDuration,
	}

	// ID court b_xxxxx (8 caractères de UUID pour correspondre à "b_4f3c1a")
	batchID := fmt.Sprintf("b_%s", strings.ReplaceAll(uuid.New().String()[:8], "-", ""))

	batch := domain.Batch{
		ID:        batchID,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Summary:   summary,
		Results:   results,
	}

	// Persistance du lot
	if err := api.store.Save(r.Context(), batch); err != nil {
		api.logger.Error("Echec de la sauvegarde du lot", "batch_id", batchID, "error", err)
		writeError(w, "internal_error", "impossible de persister le lot de verifications", http.StatusInternalServerError)
		return
	}

	// Enregistrement de l'ID du lot dans le logger contextuel s'il existe
	// (Le middleware de log pourra récupérer cet identifiant)
	r.Header.Set("X-Batch-ID", batchID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(batch)
}

// GET /v1/checks/{id}
func (api *API) GetCheckBatch(w http.ResponseWriter, r *http.Request) {
	// Récupération du paramètre de chemin dans Go 1.22
	id := r.PathValue("id")
	if id == "" {
		writeError(w, "invalid_request", "L'identifiant du lot est requis", http.StatusBadRequest)
		return
	}

	batch, err := api.store.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrBatchNotFound) {
			writeError(w, "batch_not_found", fmt.Sprintf("aucun lot avec l'id %s", id), http.StatusNotFound)
			return
		}
		api.logger.Error("Erreur lors de la lecture du lot", "batch_id", id, "error", err)
		writeError(w, "internal_error", "impossible de recuperer le lot", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(batch)
}

// ============================================================================
// MIDDLEWARES (SLOG JSON & RECOVERY)
// ============================================================================

type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriterWrapper) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// LoggingMiddleware journalise chaque requête HTTP en JSON, à l'exclusion de /healthz.
func LoggingMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Exclure /healthz pour ne pas polluer les logs applicatifs
			if r.URL.Path == "/healthz" {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()
			
			// Wrapper pour intercepter le code de statut HTTP
			wrapper := &responseWriterWrapper{ResponseWriter: w, statusCode: http.StatusOK}
			
			next.ServeHTTP(wrapper, r)
			
			duration := time.Since(start).Milliseconds()

			// Extraction du X-Batch-ID s'il a été mis par le handler POST
			batchID := r.Header.Get("X-Batch-ID")

			logger.Info("requete HTTP traitee",
				"method", r.Method,
				"path", r.URL.Path,
				"status", wrapper.statusCode,
				"duration_ms", duration,
				"batch_id", batchID,
			)
		})
	}
}

// RecoveryMiddleware intercepte les panics et renvoie une erreur 500 structurée.
func RecoveryMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					// Log structuré de la panic avec stack trace
					logger.Error("panic interceptée dans le handler",
						"error", err,
						"stack", string(debug.Stack()),
					)
					writeError(w, "internal_error", "un probleme interne inattendu est survenu", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
