package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// travailleur lit les tâches du canal taches, simule le travail,
// envoie le résultat sur resultats, et signale sa fin au WaitGroup.
func travailleur(id int, taches <-chan int, resultats chan<- string, wg *sync.WaitGroup) {
	defer wg.Done()

	// Boucle sur le canal de tâches jusqu'à sa fermeture
	for tacheID := range taches {
		fmt.Printf("Worker %d: Début du traitement de la tâche %d...\n", id, tacheID)
		
		// Simulation du travail (durée aléatoire entre 50 et 500 ms)
		duree := time.Duration(rand.Intn(451)+50) * time.Millisecond
		time.Sleep(duree)
		
		fmt.Printf("Worker %d: Tâche %d terminée.\n", id, tacheID)
		
		// Envoi du résultat
		resultats <- fmt.Sprintf("Tâche %d traitée par le Worker %d.", tacheID, id)
	}
}

func main() {
	// Initialisation du générateur de nombres aléatoires
	rand.Seed(time.Now().UnixNano())

	var wg sync.WaitGroup

	// Définition du nombre de workers et de tâches
	nbWorkers := 3
	nbTaches := 10

	// Canaux : taches est bufferisé pour l'envoi asynchrone des tâches
	// resultats DOIT être bufferisé (au moins de la taille du nombre de tâches)
	// si on veut faire wg.Wait() PUIS close(resultats) PUIS lire dans le main.
	// Sans cela, un worker se bloquerait à l'envoi car personne ne lirait le canal
	// pendant que le main() est bloqué dans wg.Wait().
	taches := make(chan int, nbTaches)
	resultats := make(chan string, nbTaches)

	// Lancement de 3 workers
	for w := 1; w <= nbWorkers; w++ {
		wg.Add(1)
		go travailleur(w, taches, resultats, &wg)
	}

	// Envoi des 10 tâches au canal
	for t := 1; t <= nbTaches; t++ {
		taches <- t
	}
	
	// Fermeture du canal taches pour signaler aux workers qu'il n'y a plus de tâches
	close(taches)

	// Attente que tous les workers aient fini de traiter les tâches
	wg.Wait()

	// Fermeture du canal resultats
	close(resultats)

	// Lecture et affichage des résultats
	fmt.Println("\n--- Résultats reçus ---")
	for res := range resultats {
		fmt.Println(res)
	}
}
