package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"urlwatch/internal/api"
	"urlwatch/internal/checker"
	"urlwatch/internal/domain"
	"urlwatch/internal/store"

	// SQLite GORM (CGO-free)
	gsqlite "github.com/glebarez/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func main() {
	// ============================================================================
	// 1. CONFIGURATION DU LOGGER STRUCTURÉ (slog JSON)
	// ============================================================================
	logLevelStr := os.Getenv("LOG_LEVEL")
	var level slog.Level
	switch strings.ToUpper(logLevelStr) {
	case "DEBUG":
		level = slog.LevelDebug
	case "WARN":
		level = slog.LevelWarn
	case "ERROR":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))
	slog.SetDefault(logger)

	logger.Info("Initialisation d'URLWatch", "log_level", level.String())

	// ============================================================================
	// 2. CONFIGURATION DES VARIABLES D'ENVIRONNEMENT ET INITIALISATION DU STORE
	// ============================================================================
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	storeType := os.Getenv("STORE_TYPE")
	if storeType == "" {
		// Par défaut, nous utilisons SQLite pour valoriser la persistance
		storeType = "sqlite"
	}

	var dataStore domain.Store

	if strings.ToLower(storeType) == "sqlite" {
		dbPath := os.Getenv("DATABASE_PATH")
		if dbPath == "" {
			dbPath = "urlwatch.db"
		}
		logger.Info("Initialisation de la base SQLite...", "path", dbPath)
		
		// Ouverture avec le driver SQLite pur Go GORM (CGO-free)
		gormDB, err := gorm.Open(gsqlite.Open(dbPath), &gorm.Config{
			Logger: gormlogger.Default.LogMode(gormlogger.Silent),
		})
		if err != nil {
			logger.Error("Impossible d'ouvrir la base SQLite", "error", err)
			os.Exit(1)
		}

		sqliteStore, err := store.NewSQLiteStore(gormDB)
		if err != nil {
			logger.Error("Echec de la migration SQLite", "error", err)
			os.Exit(1)
		}
		dataStore = sqliteStore
		logger.Info("Persistance SQLite GORM configurée.")
	} else {
		dataStore = store.NewMemoryStore()
		logger.Info("Persistance en mémoire configurée.")
	}

	// Initialisation du vérificateur d'URLs réel
	httpChecker := checker.NewHTTPChecker(nil)

	// Initialisation de la couche API
	appAPI := api.NewAPI(dataStore, httpChecker, logger)

	// ============================================================================
	// 3. ENREGISTREMENT DES ROUTES ET MIDDLEWARES
	// ============================================================================
	mux := http.NewServeMux()
	appAPI.RegisterRoutes(mux)

	// Application des middlewares dans l'ordre : Recovery puis Logging
	var handler http.Handler = mux
	handler = api.LoggingMiddleware(logger)(handler)
	handler = api.RecoveryMiddleware(logger)(handler)

	// Configuration du serveur HTTP
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 40 * time.Second, // Doit être supérieur au timeout max des lots (30s)
		IdleTimeout:  60 * time.Second,
	}

	// ============================================================================
	// 4. DÉMARRAGE ASYNCHRONE ET ARRÊT GRACIEUX (GRACEFUL SHUTDOWN)
	// ============================================================================
	// Canal pour intercepter les signaux d'arrêt système
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Lancement du serveur dans une goroutine
	go func() {
		logger.Info(fmt.Sprintf("Serveur URLWatch en ecoute sur http://localhost:%s", port))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("Erreur fatale du serveur HTTP", "error", err)
			os.Exit(1)
		}
	}()

	// Blocage jusqu'à réception d'un signal
	sig := <-shutdownChan
	logger.Info("Signal d'arret recu, initiation de l'arret gracieux...", "signal", sig.String())

	// Contexte de 10 secondes pour laisser les requêtes en cours se terminer
	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()

	if err := srv.Shutdown(ctxShutdown); err != nil {
		logger.Error("Echec de l'arret gracieux", "error", err)
		os.Exit(1)
	}

	logger.Info("Serveur URLWatch arrete proprement.")
}
