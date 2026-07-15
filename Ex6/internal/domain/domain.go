package domain

import (
	"context"
	"errors"
	"fmt"
)

// CheckResult représente le résultat de la vérification d'une URL individuelle.
type CheckResult struct {
	URL        string `json:"url"`
	StatusCode int    `json:"status_code,omitempty"`
	OK         bool   `json:"ok"`
	LatencyMS  int64  `json:"latency_ms"`
	Error      string `json:"error,omitempty"`
}

// BatchSummary contient l'agrégation statistique des résultats d'un lot.
type BatchSummary struct {
	Total      int   `json:"total"`
	Up         int   `json:"up"`
	Down       int   `json:"down"`
	DurationMS int64 `json:"duration_ms"`
}

// Batch regroupe la liste des résultats de vérification, le résumé et les métadonnées.
type Batch struct {
	ID        string         `json:"batch_id" gorm:"primaryKey"`
	CreatedAt string         `json:"created_at"`
	Summary   BatchSummary   `json:"summary" gorm:"embedded"`
	Results   []CheckResult  `json:"results" gorm:"serializer:json"` // SQLite stockera le JSON sérialisé
}

// ErrBatchNotFound est retournée par le Store lorsque l'ID du lot est inconnu.
var ErrBatchNotFound = errors.New("batch not found")

// ValidationError est une erreur personnalisée pour la validation des champs DTO.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation echouee sur le champ '%s' : %s", e.Field, e.Message)
}

// Checker définit l'interface de vérification d'une URL individuelle.
type Checker interface {
	Check(ctx context.Context, url string) CheckResult
}

// Store régit la persistance et la lecture des lots (batches).
type Store interface {
	Save(ctx context.Context, b Batch) error
	Get(ctx context.Context, id string) (Batch, error)
}

// ComputeSummary agrège les résultats d'un lot en un résumé statistique.
func ComputeSummary(results []CheckResult, durationMS int64) BatchSummary {
	up := 0
	for _, res := range results {
		if res.OK {
			up++
		}
	}
	total := len(results)
	return BatchSummary{
		Total:      total,
		Up:         up,
		Down:       total - up,
		DurationMS: durationMS,
	}
}
