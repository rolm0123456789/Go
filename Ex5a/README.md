# TP : API REST Simple en Go

Ce projet contient l'implémentation d'une API REST simple pour gérer des éléments (`Item`) en mémoire, en utilisant uniquement le package standard `net/http` et `github.com/google/uuid` pour générer des identifiants uniques. 

Il est développé dans le cadre de l'exercice **Ex5a**.

---

## Fonctionnalités d'Architecture implémentées

- **Séparation sémantique** : Le code de main.go sépare proprement la couche données (Store de données encapsulé sous `sync.RWMutex` pour garantir la sécurité concurrentielle) de la couche transport (handlers de requête).
- **Routage standard** : Routage HTTP pur réalisé en utilisant les fonctionnalités natives de `net/http.ServeMux` (routage par préfixe pour récupérer les identifiants).
- **Sécurité et concurrence (Thread-safety)** : La slice en mémoire étant partagée entre les goroutines HTTP, toutes les écritures et lectures sont protégées par un verrou `sync.RWMutex` sur le store de données.
- **Robustesse** : Validation des données d'entrée (par exemple, le champ `name` obligatoire), gestion des JSON mal formés (400 Bad Request) et des identifiants inexistants (404 Not Found).

---

## Instructions d'Exécution

Pour lancer le serveur de l'API REST :

1. Ouvrez un terminal dans le dossier `Ex5a` de votre espace de travail.
2. Lancez le serveur avec la commande suivante :
   ```bash
   go run main.go
   ```
3. Le serveur écoute sur le port `8080` (`http://localhost:8080`).

---

## Guide des Endpoints et Commandes de Test (`curl`)

Vous pouvez tester l'API avec des outils comme Postman, Thunder Client ou directement dans votre terminal avec les commandes `curl` ci-dessous :

### 1. Récupérer tous les éléments (`GET /items`)
Récupère la liste de tous les éléments actuellement enregistrés.
* **Commande** :
  ```bash
  curl -i -X GET http://localhost:8080/items
  ```
* **Statut HTTP attendu** : `200 OK`
* **Réponse attendue** : Un tableau JSON contenant les éléments (2 éléments sont déjà présents par défaut pour le test).

---

### 2. Ajouter un nouvel élément (`POST /items`)
Crée un nouvel élément avec un nom et une description. L'identifiant (UUID) est généré automatiquement par le serveur.
* **Commande** :
  ```bash
  curl -i -X POST http://localhost:8080/items \
       -H "Content-Type: application/json" \
       -d '{"name": "Nouvel Outil", "description": "Un outil de test créé via POST."}'
  ```
* **Statut HTTP attendu** : `201 Created`
* **Réponse attendue** : L'objet créé sous format JSON (contenant son nouvel identifiant unique UUID).
  * *Note : Récupérez la valeur du champ `"id"` généré pour tester les routes suivantes.*

---

### 3. Récupérer un élément par son ID (`GET /items/{id}`)
Récupère les détails d'un élément spécifique.
* **Commande** (remplacez `1` par l'ID de votre choix ou le UUID généré au-dessus) :
  ```bash
  curl -i -X GET http://localhost:8080/items/1
  ```
* **Statut HTTP attendu** : `200 OK` (ou `404 Not Found` si l'élément n'existe pas).

---

### 4. Mettre à jour un élément existant (`PUT /items/{id}`)
Modifie le nom et/ou la description d'un élément existant.
* **Commande** (remplacez `1` par l'ID de l'élément à modifier) :
  ```bash
  curl -i -X PUT http://localhost:8080/items/1 \
       -H "Content-Type: application/json" \
       -d '{"name": "Item Modifie 1", "description": "Cette description a ete mise a jour via PUT."}'
  ```
* **Statut HTTP attendu** : `200 OK` (ou `404 Not Found` si l'identifiant n'existe pas).

---

### 5. Supprimer un élément (`DELETE /items/{id}`)
Supprime un élément spécifique de la collection.
* **Commande** (remplacez `2` par l'ID de l'élément à supprimer) :
  ```bash
  curl -i -X DELETE http://localhost:8080/items/2
  ```
* **Statut HTTP attendu** : `204 No Content` (ou `404 Not Found` si l'identifiant n'existe pas).

---

### 6. Gestion des cas d'erreur
Vous pouvez également tester la robustesse des handlers en exécutant des requêtes invalides :

* **Méthode non supportée (ex: PATCH)** :
  ```bash
  curl -i -X PATCH http://localhost:8080/items
  ```
  *Réponse attendue : `405 Method Not Allowed` avec message d'erreur JSON.*

* **JSON mal formé (corps manquant ou brisé)** :
  ```bash
  curl -i -X POST http://localhost:8080/items \
       -H "Content-Type: application/json" \
       -d '{"name":'
  ```
  *Réponse attendue : `400 Bad Request`.*

* **Validation de champ (Nom vide)** :
  ```bash
  curl -i -X POST http://localhost:8080/items \
       -H "Content-Type: application/json" \
       -d '{"name": "", "description": "Test de validation"}'
  ```
  *Réponse attendue : `400 Bad Request`.*
