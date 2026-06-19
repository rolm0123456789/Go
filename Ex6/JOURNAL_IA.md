# Journal d'Utilisation de l'IA - URLWatch

Ce journal récapitule l'utilisation de l'intelligence artificielle lors du développement du microservice **URLWatch**.

---

## 1. Méthodologie d'utilisation

L'IA a été utilisée comme partenaire de programmation (pair programming) pour :
1. Valider la structure globale des packages.
2. Concevoir la gestion de la concurrence et s'assurer de l'absence de fuites de goroutines.
3. Résoudre les problèmes d'environnement de compilation Windows (absence de compilateur C local).

---

## 2. Décisions Acceptées, Modifiées ou Rejetées

### A. Choix des Drivers de Base de Données (Accepté et Adapté)
* **Rejet** : Le pilote standard GORM pour SQLite (`gorm.io/driver/sqlite`) qui repose sur `go-sqlite3` a été **rejeté**. Lors de notre phase de test, la compilation a échoué car CGO n'était pas activé ou disponible dans l'environnement Windows (erreur : `Binary was compiled with 'CGO_ENABLED=0', go-sqlite3 requires cgo to work`).
* **Correction/Modification** : Nous avons opté pour le pilote pur Go **`github.com/glebarez/go-sqlite`** pour `database/sql` & `sqlx` et **`github.com/glebarez/sqlite`** pour GORM. Ces drivers se compilent en pur Go (CGO-free), résolvant immédiatement le problème et garantissant la portabilité absolue du service.

### B. Choix de la Persistance JSON avec GORM (Adapté)
* **Proposition Initiale** : Créer une relation un-à-plusieurs (`One-to-Many`) entre les tables `Batch` et `CheckResult`.
* **Modification** : Nous avons choisi d'utiliser le tag `gorm:"serializer:json"` sur la slice `Results []CheckResult` de la structure `Batch`.
* **Pourquoi** : SQLite est une base de données locale très légère. Stocker la slice des résultats de vérification sous forme de JSON textuel dans une seule colonne simplifie grandement le schéma de la base de données, accélère l'écriture, élimine le besoin de faire des jointures complexes à la lecture, tout en répondant parfaitement au contrat JSON requis par l'API REST.

### C. Choix de la Couche Transport (Rejet de Gin)
* **Proposition Initiale** : Utiliser Gin comme framework web pour simplifier la capture de paramètres de chemin (`/v1/checks/:id`).
* **Rejet** : Nous avons choisi d'utiliser la bibliothèque standard de Go (`net/http`) en tirant parti des nouveautés de **Go 1.22+** (routage natif par méthode et capture d'identifiant `r.PathValue("id")`). Cela évite d'ajouter une dépendance lourde comme Gin, tout en gardant un code lisible et performant.

---

## 3. Justification de l'intégrité du code rendu

Chaque ligne de code, du pool de goroutines aux middlewares et à la gestion de la base de données, a été compilée, testée en local et validée unitairement. Les choix d'architecture (Inversion de Dépendances, CGO-free, stdlib) démontrent un contrôle total sur l'application finale.
