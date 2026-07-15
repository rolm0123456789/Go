package store

import (
	"context"
	"fmt"
	"sync"

	"urlwatch/internal/domain"
)

// MemoryStore est un stockage en mémoire thread-safe des lots de vérification.
type MemoryStore struct {
	sync.RWMutex
	batches map[string]domain.Batch
}

// NewMemoryStore construit un nouveau MemoryStore.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		batches: make(map[string]domain.Batch),
	}
}

// Save enregistre un lot dans la mémoire.
func (ms *MemoryStore) Save(ctx context.Context, b domain.Batch) error {
	ms.Lock()
	defer ms.Unlock()

	// Vérification de l'annulation du contexte
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("annulation de la sauvegarde du lot %s: %w", b.ID, err)
	}

	ms.batches[b.ID] = b
	return nil
}

// Get récupère un lot par son identifiant unique. Retourne ErrBatchNotFound si absent.
func (ms *MemoryStore) Get(ctx context.Context, id string) (domain.Batch, error) {
	ms.RLock()
	defer ms.RUnlock()

	if err := ctx.Err(); err != nil {
		return domain.Batch{}, fmt.Errorf("annulation de la lecture du lot %s: %w", id, err)
	}

	b, exists := ms.batches[id]
	if !exists {
		return domain.Batch{}, domain.ErrBatchNotFound
	}
	return b, nil
}
