package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/google/uuid"
)

// ============================================================================
// 1. DÉFINITION DU MODÈLE ET COUCHE DE DONNÉES (THREAD-SAFE)
// ============================================================================

// Item représente un élément de notre collection.
type Item struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// DataStore encapsule le stockage en mémoire et le verrou RWMutex pour les accès concurrents.
type DataStore struct {
	sync.RWMutex
	items []Item
}

// store est notre base de données en mémoire.
var store = &DataStore{
	items: []Item{
		{ID: "1", Name: "Item Initial 1", Description: "Ceci est le premier élément de test."},
		{ID: "2", Name: "Item Initial 2", Description: "Ceci est le second élément de test."},
	},
}

// GetAll retourne une copie du slice des items pour éviter les conflits d'accès lors de l'encodage.
func (ds *DataStore) GetAll() []Item {
	ds.RLock()
	defer ds.RUnlock()
	
	// Copie superficielle (shallow copy) pour éviter des race conditions
	copiedItems := make([]Item, len(ds.items))
	copy(copiedItems, ds.items)
	return copiedItems
}

// GetByID recherche un item par son ID. Retourne l'item et un booléen indiquant s'il a été trouvé.
func (ds *DataStore) GetByID(id string) (Item, bool) {
	ds.RLock()
	defer ds.RUnlock()

	for _, item := range ds.items {
		if item.ID == id {
			return item, true
		}
	}
	return Item{}, false
}

// Create ajoute un nouvel item au store en générant un identifiant unique (UUID).
func (ds *DataStore) Create(name, description string) Item {
	ds.Lock()
	defer ds.Unlock()

	newItem := Item{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
	}
	ds.items = append(ds.items, newItem)
	return newItem
}

// Update met à jour un item existant. Retourne l'item mis à jour et un booléen de réussite.
func (ds *DataStore) Update(id string, name, description string) (Item, bool) {
	ds.Lock()
	defer ds.Unlock()

	for i, item := range ds.items {
		if item.ID == id {
			ds.items[i].Name = name
			ds.items[i].Description = description
			return ds.items[i], true
		}
	}
	return Item{}, false
}

// Delete supprime un item par son ID. Retourne un booléen indiquant si l'élément existait.
func (ds *DataStore) Delete(id string) bool {
	ds.Lock()
	defer ds.Unlock()

	for i, item := range ds.items {
		if item.ID == id {
			// Suppression de l'élément dans le slice
			ds.items = append(ds.items[:i], ds.items[i+1:]...)
			return true
		}
	}
	return false
}

// ============================================================================
// 2. COUCHE TRANSPORT & HANDLERS HTTP
// ============================================================================

// writeJSONError est un helper pour renvoyer des messages d'erreur formatés en JSON.
func writeJSONError(w http.ResponseWriter, message string, statusCode int) {
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// itemsHandler aiguille les requêtes faites sur "/items" (sans ID).
func itemsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		getItemsHandler(w, r)
	case http.MethodPost:
		createItemHandler(w, r)
	default:
		w.Header().Set("Allow", "GET, POST")
		writeJSONError(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
	}
}

// itemByIDHandler aiguille les requêtes faites sur "/items/{id}".
func itemByIDHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Extraction de l'ID à partir du chemin de l'URL
	// /items/{id} -> on enlève le préfixe "/items/"
	id := strings.TrimPrefix(r.URL.Path, "/items/")
	
	// Si l'ID est vide, cela signifie que la requête a touché "/items/" sans ID
	if id == "" {
		writeJSONError(w, "ID requis", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		getItemHandler(w, r, id)
	case http.MethodPut:
		updateItemHandler(w, r, id)
	case http.MethodDelete:
		deleteItemHandler(w, r, id)
	default:
		w.Header().Set("Allow", "GET, PUT, DELETE")
		writeJSONError(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
	}
}

// GET /items - Récupérer tous les éléments
func getItemsHandler(w http.ResponseWriter, r *http.Request) {
	itemsList := store.GetAll()
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(itemsList); err != nil {
		writeJSONError(w, "Erreur interne lors de l'encodage JSON", http.StatusInternalServerError)
	}
}

// GET /items/{id} - Récupérer un élément par ID
func getItemHandler(w http.ResponseWriter, r *http.Request, id string) {
	item, found := store.GetByID(id)
	if !found {
		writeJSONError(w, fmt.Sprintf("Élément avec l'ID '%s' introuvable", id), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(item); err != nil {
		writeJSONError(w, "Erreur interne lors de l'encodage JSON", http.StatusInternalServerError)
	}
}

// POST /items - Ajouter un nouvel élément
func createItemHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	// Décodage du corps de la requête
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSONError(w, "Corps de requête invalide ou JSON mal formé", http.StatusBadRequest)
		return
	}

	// Validation des entrées obligatoires
	if strings.TrimSpace(input.Name) == "" {
		writeJSONError(w, "Le champ 'name' est obligatoire et ne peut pas être vide", http.StatusBadRequest)
		return
	}

	// Création du nouvel item
	newItem := store.Create(input.Name, input.Description)

	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(newItem); err != nil {
		writeJSONError(w, "Erreur lors de l'encodage de la réponse", http.StatusInternalServerError)
	}
}

// PUT /items/{id} - Mettre à jour un élément existant
func updateItemHandler(w http.ResponseWriter, r *http.Request, id string) {
	var input struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	// Décodage du corps de la requête
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSONError(w, "Corps de requête invalide ou JSON mal formé", http.StatusBadRequest)
		return
	}

	// Validation
	if strings.TrimSpace(input.Name) == "" {
		writeJSONError(w, "Le champ 'name' est obligatoire", http.StatusBadRequest)
		return
	}

	// Mise à jour de l'item
	updatedItem, found := store.Update(id, input.Name, input.Description)
	if !found {
		writeJSONError(w, fmt.Sprintf("Élément avec l'ID '%s' introuvable pour la mise à jour", id), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(updatedItem); err != nil {
		writeJSONError(w, "Erreur lors de l'encodage de la réponse", http.StatusInternalServerError)
	}
}

// DELETE /items/{id} - Supprimer un élément
func deleteItemHandler(w http.ResponseWriter, r *http.Request, id string) {
	deleted := store.Delete(id)
	if !deleted {
		writeJSONError(w, fmt.Sprintf("Élément avec l'ID '%s' introuvable pour la suppression", id), http.StatusNotFound)
		return
	}

	// 204 No Content est le statut HTTP idéal pour une suppression réussie sans corps de retour
	w.WriteHeader(http.StatusNoContent)
}

// ============================================================================
// 3. POINT D'ENTRÉE ET CONFIGURATION DU SERVEUR
// ============================================================================

func main() {
	mux := http.NewServeMux()

	// Association des routes aux handlers
	// /items gère GET (tous) et POST (création)
	mux.HandleFunc("/items", itemsHandler)
	
	// /items/ gère le routage par préfixe pour GET /{id}, PUT /{id}, DELETE /{id}
	// Note : net/http fait correspondre par préfixe tout ce qui commence par "/items/"
	mux.HandleFunc("/items/", itemByIDHandler)

	port := ":8080"
	log.Printf("Serveur démarré avec succès sur http://localhost%s\n", port)
	
	if err := http.ListenAndServe(port, mux); err != nil {
		log.Fatalf("Erreur lors du démarrage du serveur : %s\n", err)
	}
}
