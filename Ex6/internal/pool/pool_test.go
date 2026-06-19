package pool

import (
	"context"
	"strings"
	"testing"
	"time"

	"urlwatch/internal/checker"
	"urlwatch/internal/domain"
)

func TestRunBatch_Success(t *testing.T) {
	// Préparation des réponses mockées
	responses := map[string]domain.CheckResult{
		"https://go.dev": {
			URL:        "https://go.dev",
			StatusCode: 200,
			OK:         true,
			LatencyMS:  10,
		},
		"https://google.com": {
			URL:        "https://google.com",
			StatusCode: 200,
			OK:         true,
			LatencyMS:  15,
		},
		"https://invalid.url": {
			URL:       "https://invalid.url",
			OK:        false,
			LatencyMS: 5,
			Error:     "dns lookup failed",
		},
	}
	mc := checker.NewMockChecker(responses)

	urls := []string{"https://go.dev", "https://google.com", "https://invalid.url"}
	
	// Exécution du batch avec 2 workers
	results := RunBatch(context.Background(), urls, 2, 2*time.Second, mc)

	if len(results) != len(urls) {
		t.Fatalf("Attendu %d résultats, obtenu %d", len(urls), len(results))
	}

	// Indexation par URL pour faciliter la vérification (l'ordre de sortie du pool n'est pas garanti)
	resMap := make(map[string]domain.CheckResult)
	for _, res := range results {
		resMap[res.URL] = res
	}

	// Vérification de go.dev
	g, ok := resMap["https://go.dev"]
	if !ok || !g.OK || g.StatusCode != 200 {
		t.Errorf("go.dev devrait être OK (200), obtenu %+v", g)
	}

	// Vérification de invalid.url
	inv, ok := resMap["https://invalid.url"]
	if !ok || inv.OK || inv.Error == "" {
		t.Errorf("invalid.url devrait être en échec avec erreur, obtenu %+v", inv)
	}
}

func TestRunBatch_UnitTimeout(t *testing.T) {
	// Préparation d'une URL lente (latence de 500ms)
	responses := map[string]domain.CheckResult{
		"https://slow.url": {
			URL:        "https://slow.url",
			StatusCode: 200,
			OK:         true,
			LatencyMS:  500, // 500ms
		},
	}
	mc := checker.NewMockChecker(responses)

	urls := []string{"https://slow.url"}

	// On lance le pool avec un timeout unitaire de 100ms
	// La vérification lente doit être interrompue et renvoyer une erreur de timeout
	results := RunBatch(context.Background(), urls, 1, 100*time.Millisecond, mc)

	if len(results) != 1 {
		t.Fatalf("Attendu 1 résultat, obtenu %d", len(results))
	}

	res := results[0]
	if res.OK {
		t.Error("L'URL lente aurait dû expirer (timeout)")
	}
	if !strings.Contains(res.Error, context.DeadlineExceeded.Error()) {
		t.Errorf("Attendu l'erreur '%s', obtenu '%s'", context.DeadlineExceeded.Error(), res.Error)
	}
}

func TestRunBatch_GlobalContextCancel(t *testing.T) {
	responses := map[string]domain.CheckResult{
		"https://url1.com": {URL: "https://url1.com", StatusCode: 200, OK: true, LatencyMS: 50},
		"https://url2.com": {URL: "https://url2.com", StatusCode: 200, OK: true, LatencyMS: 50},
	}
	mc := checker.NewMockChecker(responses)

	ctx, cancel := context.WithCancel(context.Background())
	// Annulation immédiate du contexte global
	cancel()

	urls := []string{"https://url1.com", "https://url2.com"}
	results := RunBatch(ctx, urls, 2, 1*time.Second, mc)

	if len(results) != len(urls) {
		t.Fatalf("Attendu %d résultats, obtenu %d", len(urls), len(results))
	}

	for _, res := range results {
		if res.OK {
			t.Errorf("Le résultat de '%s' aurait dû être annulé", res.URL)
		}
		if res.Error == "" {
			t.Errorf("Le résultat de '%s' devrait contenir une erreur d'annulation", res.URL)
		}
	}
}
