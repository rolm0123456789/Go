package store

import (
	"context"
	"errors"

	"urlwatch/internal/domain"

	"gorm.io/gorm"
)

// SQLiteStore est l'implémentation de persistances basée sur SQLite (via GORM).
type SQLiteStore struct {
	db *gorm.DB
}

// NewSQLiteStore construit un nouveau store SQLite et applique les auto-migrations.
func NewSQLiteStore(db *gorm.DB) (*SQLiteStore, error) {
	// Auto-migration de la structure Batch
	// GORM va utiliser le tag `gorm:"serializer:json"` sur la slice Results pour la stocker en JSON TEXT.
	if err := db.AutoMigrate(&domain.Batch{}); err != nil {
		return nil, err
	}
	return &SQLiteStore{db: db}, nil
}

// Save persiste le lot d'URLs dans la base de données SQLite.
func (ss *SQLiteStore) Save(ctx context.Context, b domain.Batch) error {
	// Utilisation de Create avec propagation du contexte
	return ss.db.WithContext(ctx).Create(&b).Error
}

// Get récupère un lot par ID depuis la base SQLite.
func (ss *SQLiteStore) Get(ctx context.Context, id string) (domain.Batch, error) {
	var b domain.Batch
	err := ss.db.WithContext(ctx).First(&b, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.Batch{}, domain.ErrBatchNotFound
		}
		return domain.Batch{}, err
	}
	return b, nil
}
