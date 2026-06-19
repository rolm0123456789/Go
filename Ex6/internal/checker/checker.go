package checker

import (
	"context"
	"net/http"
	"time"

	"urlwatch/internal/domain"
)

// HTTPChecker est l'implémentation réelle utilisant net/http pour vérifier les URLs.
type HTTPChecker struct {
	client *http.Client
}

// NewHTTPChecker construit un nouveau vérificateur HTTP.
func NewHTTPChecker(client *http.Client) *HTTPChecker {
	if client == nil {
		client = &http.Client{
			// Timeout global de sécurité par défaut si aucun contexte n'en définit
			Timeout: 10 * time.Second,
		}
	}
	return &HTTPChecker{client: client}
}

// Check effectue un appel HTTP GET pour tester la disponibilité de l'URL.
func (hc *HTTPChecker) Check(ctx context.Context, url string) domain.CheckResult {
	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return domain.CheckResult{
			URL:       url,
			OK:        false,
			LatencyMS: time.Since(start).Milliseconds(),
			Error:     err.Error(),
		}
	}

	resp, err := hc.client.Do(req)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		return domain.CheckResult{
			URL:       url,
			OK:        false,
			LatencyMS: latency,
			Error:     err.Error(),
		}
	}
	defer resp.Body.Close()

	// On considère OK si le code de statut HTTP est dans la plage 2xx ou 3xx
	ok := resp.StatusCode >= 200 && resp.StatusCode < 400

	return domain.CheckResult{
		URL:        url,
		StatusCode: resp.StatusCode,
		OK:         ok,
		LatencyMS:  latency,
	}
}

// MockChecker est une implémentation déterministe pour les tests.
type MockChecker struct {
	Responses map[string]domain.CheckResult
}

// NewMockChecker construit un MockChecker.
func NewMockChecker(responses map[string]domain.CheckResult) *MockChecker {
	if responses == nil {
		responses = make(map[string]domain.CheckResult)
	}
	return &MockChecker{Responses: responses}
}

// Check simule la vérification d'une URL en respectant la latence et l'annulation du contexte.
func (mc *MockChecker) Check(ctx context.Context, url string) domain.CheckResult {
	start := time.Now()
	res, exists := mc.Responses[url]
	if !exists {
		// Réponse par défaut positive immédiate
		return domain.CheckResult{
			URL:        url,
			StatusCode: 200,
			OK:         true,
			LatencyMS:  0,
		}
	}

	// Simulation de la latence de traitement
	if res.LatencyMS > 0 {
		select {
		case <-time.After(time.Duration(res.LatencyMS) * time.Millisecond):
			// Latence passée
		case <-ctx.Done():
			// Le contexte a été annulé avant la fin de la latence
			return domain.CheckResult{
				URL:       url,
				OK:        false,
				LatencyMS: time.Since(start).Milliseconds(),
				Error:     ctx.Err().Error(),
			}
		}
	}

	// Retourne le résultat simulé avec la latence finale
	res.LatencyMS = time.Since(start).Milliseconds()
	return res
}
