package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// effectuerTache simule le traitement d'une tâche d'une durée aléatoire et signale sa fin au WaitGroup.
func effectuerTache(id int, wg *sync.WaitGroup) {
	// defer wg.Done() est appelé juste après le démarrage de la fonction
	// pour s'assurer qu'il sera exécuté dès que la fonction se termine.
	defer wg.Done()

	fmt.Printf("Goroutine %d: Début de la tâche...\n", id)
	
	// Durée aléatoire entre 50 et 500 millisecondes
	duree := time.Duration(rand.Intn(451)+50) * time.Millisecond
	time.Sleep(duree)
	
	fmt.Printf("Goroutine %d: Tâche terminée.\n", id)
}

func main() {
	// Initialisation du générateur de nombres aléatoires
	rand.Seed(time.Now().UnixNano())

	var wg sync.WaitGroup

	// Lancement de 5 goroutines concurrentes
	for i := 1; i <= 5; i++ {
		wg.Add(1)
		go effectuerTache(i, &wg)
	}

	fmt.Println("Toutes les goroutines lancées.")

	// Attente du signal de toutes les goroutines
	wg.Wait()

	fmt.Println("Toutes les goroutines ont terminé leur exécution.")
}
