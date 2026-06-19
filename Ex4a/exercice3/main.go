package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// effectuerTache simule une tâche et envoie le résultat dans le canal avant de signaler sa fin au WaitGroup.
func effectuerTache(id int, resultChan chan string, wg *sync.WaitGroup) {
	defer wg.Done()

	fmt.Printf("Goroutine %d: Début de la tâche...\n", id)
	
	// Durée aléatoire entre 50 et 500 millisecondes
	duree := time.Duration(rand.Intn(451)+50) * time.Millisecond
	time.Sleep(duree)
	
	fmt.Printf("Goroutine %d: Tâche terminée.\n", id)

	// Envoi du résultat sur le canal
	resultChan <- fmt.Sprintf("Goroutine %d a terminé avec succès.", id)
}

func main() {
	// Initialisation du générateur de nombres aléatoires
	rand.Seed(time.Now().UnixNano())

	var wg sync.WaitGroup
	
	// On crée un canal bufferisé de taille 5.
	// Pourquoi ? Si le canal n'était pas bufferisé (taille 0), les goroutines se bloqueraient à l'écriture :
	// `resultChan <- ...` car le main() n'est pas encore en train de lire (il est bloqué sur wg.Wait()).
	// Les goroutines ne pourraient pas se terminer, donc wg.Done() ne serait pas appelé, provoquant un DEADLOCK.
	// Avec un buffer de 5, chaque goroutine peut écrire son message et se terminer immédiatement.
	resultChan := make(chan string, 5)

	// Lancement de 5 goroutines concurrentes
	for i := 1; i <= 5; i++ {
		wg.Add(1)
		go effectuerTache(i, resultChan, &wg)
	}

	fmt.Println("Toutes les goroutines lancées.")

	// Attente de la fin des goroutines
	wg.Wait()

	// Fermeture du canal pour signaler qu'il n'y aura plus d'envois
	close(resultChan)

	// Lecture de tous les messages du canal jusqu'à sa fermeture
	for message := range resultChan {
		fmt.Println(message)
	}
}
