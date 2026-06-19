package pool

import (
	"context"
	"sync"
	"time"

	"urlwatch/internal/domain"
)

// RunBatch exécute la vérification d'un lot d'URLs en parallèle en limitant la concurrence.
// Il garantit qu'aucune goroutine ne fuite et respecte les annulations/timeouts de contexte.
func RunBatch(ctx context.Context, urls []string, concurrency int, urlTimeout time.Duration, chk domain.Checker) []domain.CheckResult {
	if len(urls) == 0 {
		return nil
	}
	if concurrency <= 0 {
		concurrency = 8
	}
	if concurrency > len(urls) {
		concurrency = len(urls)
	}

	// Fan-out : Canal directionnel contenant les URLs à distribuer aux workers.
	// Bufferisé à len(urls) pour éviter tout blocage d'écriture.
	tasks := make(chan string, len(urls))
	
	// Fan-in : Canal de retour collectant les CheckResult de tous les workers.
	resultsChan := make(chan domain.CheckResult, len(urls))

	// Remplissage du canal de tâches.
	for _, url := range urls {
		tasks <- url
	}
	close(tasks) // On ferme le canal car il ne recevra plus d'autres tâches.

	var wg sync.WaitGroup

	// Lancement des workers
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for url := range tasks {
				// Vérification rapide si le contexte global est déjà annulé avant d'appeler le Checker.
				if ctx.Err() != nil {
					resultsChan <- domain.CheckResult{
						URL:       url,
						OK:        false,
						Error:     ctx.Err().Error(),
					}
					continue
				}

				// Contexte spécifique pour cette tâche d'URL avec un timeout unitaire.
				// Il hérite également de l'annulation du contexte global (ctx).
				urlCtx, cancel := context.WithTimeout(ctx, urlTimeout)
				
				// Exécution de la vérification
				res := chk.Check(urlCtx, url)
				cancel() // Libère les ressources du contexte dès que possible

				resultsChan <- res
			}
		}()
	}

	// Attente de la complétion de tous les workers.
	wg.Wait()
	close(resultsChan) // On ferme le canal de réception pour terminer le range dans le main.

	// Agréger tous les résultats.
	results := make([]domain.CheckResult, 0, len(urls))
	for res := range resultsChan {
		results = append(results, res)
	}

	return results
}
