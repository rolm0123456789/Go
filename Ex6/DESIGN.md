# URLWatch - Choix de Conception et d'Architecture

Ce document détaille les décisions architecturales et techniques prises lors de l'implémentation du microservice **URLWatch**.

---

## 1. Découpage en Packages et Inversion de Dépendance

Le projet respecte une structure découplée inspirée des principes de la *Clean Architecture* :
- **`internal/domain`** : Contient uniquement les structures de données fondamentales (`Batch`, `CheckResult`) et les interfaces clés (`Checker`, `Store`). C'est le cœur du système. Aucun autre package ne s'impose à lui ; c'est le point d'ancrage de l'inversion de dépendance.
- **`internal/checker` et `internal/store`** : Implémentent les interfaces du domaine (vérification HTTP et persistance SQLite/Memory). Ces packages dépendent de `domain` mais sont indépendants les uns des autres.
- **`internal/pool`** : Package découplé chargé de l'orchestration concurrente de masse. Il utilise `domain.Checker` pour valider les URLs.
- **`internal/api`** : Gère la couche transport HTTP, les DTOs JSON de validation, et les middlewares.
- **`cmd/urlwatch/main.go`** : Câble l'ensemble des dépendances (injection) et démarre le serveur. Il est volontairement très léger.

---

## 2. Modèle de Concurrence : Bounded Worker Pool et Canaux

Le pool de travailleurs concurrents dans `internal/pool` est conçu pour être à la fois efficace, robuste et économe en ressources :
- **Taille bornée** : La concurrence est limitée par le paramètre `concurrency` (entre 1 et 50). Un pool de goroutines fixes est lancé via `sync.WaitGroup`. Aucun lancement incontrôlé de goroutines n'est effectué, évitant ainsi le dépassement de descripteurs de fichiers système ou le blocage par surcharge réseau.
- **Bufferisation** :
  - Le canal `tasks` (fan-out) est alimenté en totalité au début de la méthode et fermé immédiatement. Sa taille de buffer est égale au nombre d'URLs (maximum 100), éliminant tout blocage du thread principal à l'alimentation.
  - Le canal `resultsChan` (fan-in) est également bufferisé à la taille des URLs. Les workers peuvent y écrire leurs résultats de manière asynchrone et se terminer dès que `tasks` est vide.
- **Gestion des échecs partiels** : Si une URL échoue (erreur de DNS, timeout unitaire), cela ne fait pas échouer le lot entier. L'erreur est encapsulée dans le champ `Error` du `CheckResult` avec le statut `OK: false`. Le lot continue et retourne les résultats des autres URLs réussies.

---

## 3. Gestion des Fuites de Goroutines et Cycle de Vie des Contextes

Pour éviter toute fuite de ressources (goroutines bloquées sur un canal ou appels HTTP en suspens) :
- Tous les canaux (`tasks`, `resultsChan`) sont fermés par l'expéditeur de manière systématique après l'envoi ou l'attente du `WaitGroup` des workers.
- Le cycle de vie de chaque requête HTTP est strictement borné. Nous utilisons `context.WithTimeout` pour chaque vérification d'URL individuelle (`timeout_ms`), garantissant qu'aucun appel HTTP ne reste suspendu indéfiniment.
- De plus, les workers vérifient `ctx.Err() != nil` (annulation globale du contexte de lot) avant d'entamer une nouvelle URL. Si le lot global est annulé ou expiré, ils abandonnent les tâches restantes proprement sans bloquer.

---

## 4. Gestion des Erreurs et Correspondance HTTP

- **Erreurs Sentinelles & Types personnalisés** : Nous définissons `ErrBatchNotFound` au niveau du store. La couche API intercepte cette erreur via `errors.Is` pour renvoyer un statut `404 Not Found` propre. Les validations renvoient un type personnalisé `ValidationError`, qui est traduit par la couche API en `400 Bad Request`.
- **Wrapping** : Les erreurs de base de données ou de connexion sont encapsulées (wrapping `%w`) pour conserver la trace de l'erreur d'origine dans les logs structurés tout en renvoyant une réponse claire à l'utilisateur.

---

## 5. Choix Technologique : Bibliothèque Standard `net/http` et Drivers pur Go

- **Standard `net/http` (Go 1.22+)** : Nous avons choisi d'utiliser le multiplexeur de la bibliothèque standard plutôt que Gin. Grâce aux améliorations de Go 1.22 (support natif des méthodes de routage et des paramètres de chemin `/v1/checks/{id}`), le routage reste extrêmement propre et performant, sans dépendance tierce à maintenir.
- **SQLite pur Go** : Afin de garantir une compatibilité et portabilité maximale (sans compilateur CGO dans l'environnement), nous avons utilisé le pilote `github.com/glebarez/sqlite` qui compile en pur Go.

---

## 6. Philosophie Go : Comparatif et Limites

### 3 Arguments en faveur de Go pour ce projet :
1. **Concurrence native** : La gestion du pool concurrent avec les channels et `sync.WaitGroup` se fait en moins de 50 lignes de code lisible, sans bibliothèque asynchrone externe complexe.
2. **Légèreté et binaire unique** : L'utilisation de `net/http` natif permet de compiler un microservice autonome extrêmement léger de quelques mégaoctets, facile à déployer dans un conteneur minimal.
3. **Productivité et typage** : Le typage strict de Go associé à la sérialisation automatique via tags JSON permet d'avoir des contrats d'API très robustes et sécurisés dès la compilation.

### 1 Limite ressentie :
- L'absence de génériques complexes ou d'abstractions de transport rend l'implémentation de certains middlewares de routage ou de parsing de paramètres HTTP un peu répétitive par rapport à des frameworks orientés réflexion dans d'autres langages (comme Spring en Java).
