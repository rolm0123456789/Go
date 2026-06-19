package main

import (
	"fmt"
	"math/rand"
	"time"
)

// effectuerTache simule le traitement d'une tâche d'une durée aléatoire.
func effectuerTache(id int) {
	fmt.Printf("Goroutine %d: Début de la tâche...\n", id)
	
	// Durée aléatoire entre 50 et 500 millisecondes
	duree := time.Duration(rand.Intn(451)+50) * time.Millisecond
	time.Sleep(duree)
	
	fmt.Printf("Goroutine %d: Tâche terminée.\n", id)
}

func main() {
	// Initialisation du générateur de nombres aléatoires
	rand.Seed(time.Now().UnixNano())

	// Lancement de 5 goroutines concurrentes
	for i := 1; i <= 5; i++ {
		go effectuerTache(i)
	}

	// Message de fin de la goroutine principale (main)
	fmt.Println("Toutes les goroutines lancées.")
}
