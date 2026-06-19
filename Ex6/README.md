# URLWatch - Service de Vérification d'URLs en Masse

**URLWatch** est un microservice robuste en Go conçu pour interroger et analyser en parallèle une liste d'URLs. Il orchestre les requêtes de manière concurrente avec un pool de travailleurs (Worker Pool) borné, gère les timeouts unitaires et globaux via `context`, calcule un résumé statistique et persiste les résultats dans un store en mémoire ou une base SQLite.

---

## Fonctionnalités Clés du Projet

- **Worker Pool Borné** : Distribution des tâches via canal bufferisé avec une limite stricte de concurrence (pas de débordement).
- **Fan-Out / Fan-In** : Alimentation asynchrone des tâches et collecte centralisée des résultats sans blocage.
- **Context propagation** : Timeout configurable par URL (`timeout_ms`) et annulation globale.
- **Zéro Dépendance Web externe** : Utilisation de la bibliothèque standard `net/http` avec les fonctionnalités de routage et de paramètres de route de **Go 1.22+**.
- **Persistance Flexible (Mémoire ou SQLite pur Go)** : GORM configuré avec un pilote pur Go SQLite sans dépendance CGO pour une portabilité totale.
- **Logging structuré JSON** : Implémenté via `log/slog` avec niveau de verbosité configurable par variable d'environnement (`LOG_LEVEL`).
- **Middlewares intégrés** : Logging JSON automatique des requêtes (excluant `/healthz`) et Recovery sécurisé en cas de panic.

---

## Instructions de Construction et d'Exécution

### 1. Cloner et se positionner dans le dossier
```bash
cd d:/Bureau/Go/Ex6
```

### 2. Lancer les tests unitaires et d'intégration
```bash
$env:Path += ';C:\Program Files\Go\bin'; go test -v ./...
```
*Note : Si le compilateur C/CGO est disponible sur votre machine (ce qui est souvent le cas sur les serveurs de correction), vous pouvez ajouter l'option `-race` pour vérifier l'absence de race conditions :*
```bash
$env:Path += ';C:\Program Files\Go\bin'; go test -race -v ./...
```
### 3. Compiler l'application
```bash
$env:Path += ';C:\Program Files\Go\bin'; go build -o bin/urlwatch.exe ./cmd/urlwatch
```

### 4. Démarrer l'application
Vous pouvez démarrer l'application directement avec `go run` ou exécuter le binaire compilé.
Voici les variables d'environnement configurables :
- `PORT` : Port d'écoute HTTP (défaut `8080`).
- `STORE_TYPE` : `sqlite` ou `memory` (défaut `sqlite`).
- `DATABASE_PATH` : Nom du fichier SQLite (défaut `urlwatch.db`).
- `LOG_LEVEL` : Niveau de log `DEBUG`, `INFO`, `WARN`, `ERROR` (défaut `INFO`).

#### Commande de démarrage par défaut (SQLite, Port 8080) :
```bash
$env:Path += ';C:\Program Files\Go\bin'; go run ./cmd/urlwatch
```

---

## Guide d'utilisation de l'API (Commandes `curl`)

Le serveur écoute par défaut sur `http://localhost:8080`.

### 1. Sonde de vivacité (`GET /healthz`)
Sonde de santé simple. Cet endpoint n'apparaît pas dans les logs JSON applicatifs pour ne pas les polluer.
* **Commande** :
  ```bash
  curl -i -X GET http://localhost:8080/healthz
  ```
* **Réponse attendue** (200 OK) :
  ```json
  {"status":"ok"}
  ```

---

### 2. Soumettre un lot d'URLs à vérifier (`POST /v1/checks`)
Envoie un lot d'URLs à vérifier en parallèle.
* **Commande** :
  ```bash
  curl -i -X POST http://localhost:8080/v1/checks \
       -H "Content-Type: application/json" \
       -d '{"urls": ["https://go.dev", "https://google.com", "https://exemple.invalid"], "options": {"concurrency": 4, "timeout_ms": 2000}}'
  ```
* **Réponse attendue** (201 Created) :
  ```json
  {
    "batch_id": "b_1a2b3c4d",
    "created_at": "2026-06-19T12:00:00Z",
    "summary": {
      "total": 3,
      "up": 2,
      "down": 1,
      "duration_ms": 150
    },
    "results": [
      { "url": "https://go.dev", "status_code": 200, "ok": true, "latency_ms": 45 },
      { "url": "https://google.com", "status_code": 200, "ok": true, "latency_ms": 32 },
      { "url": "https://exemple.invalid", "ok": false, "error": "Get \"https://exemple.invalid\": dial tcp: lookup exemple.invalid: no such host", "latency_ms": 110 }
    ]
  }
  ```
  *Note : Conservez la valeur du champ `batch_id` retourné pour tester la route de lecture.*

---

### 3. Lire un lot existant (`GET /v1/checks/{id}`)
Récupère les résultats et statistiques d'un lot précédemment exécuté.
* **Commande** (remplacez `b_xxxxxx` par l'ID du lot réel) :
  ```bash
  curl -i -X GET http://localhost:8080/v1/checks/b_xxxxxx
  ```
* **Réponse attendue** (200 OK) :
  Le lot complet JSON contenant le résumé et les résultats individuels.
* **Si le lot n'existe pas** (404 Not Found) :
  ```json
  {
    "error": {
      "code": "batch_not_found",
      "message": "aucun lot avec l'id b_xxxxxx"
    }
  }
  ```
