# TP : Accès aux Bases de Données en Go

Ce projet contient l'implémentation de la gestion d'une entité `User` dans une base SQLite en comparant trois approches d'accès aux données :
1. **`database/sql`** : Le package standard natif de Go.
2. **`sqlx`** : Une extension légère pour simplifier le mapping objet-relationnel sans cacher le SQL.
3. **`GORM`** : Un ORM complet offrant une abstraction totale de SQL via une API objet orientée structures Go.

---

## Instructions d'Exécution

Pour exécuter le projet et observer le déroulement complet des tests CRUD des trois bibliothèques :

1. Ouvrez votre terminal dans le dossier `Ex5d`.
2. Lancez le programme à l'aide de la commande :
   ```bash
   go run main.go
   ```
3. Cela va automatiquement :
   - Supprimer les fichiers de base de données précédents (`test.db` et `gorm_test.db`).
   - Lancer les tests CRUD de `database/sql`.
   - Lancer les tests CRUD de `sqlx` sur la même base.
   - Créer `gorm_test.db`, exécuter l'Auto-migration GORM et dérouler ses tests CRUD.
   - Afficher tous les logs et validations en console.

---

## Explications Techniques Utiles

- **`db.Ping()`** : Permet de vérifier que la connexion avec la base de données est toujours active et valide. `sql.Open` se contente de préparer la configuration de la connexion sans l'établir immédiatement. C'est `Ping()` qui force l'établissement de la première connexion physique pour s'assurer que les identifiants et l'adresse sont corrects.
- **`db.Exec()`** : Utilisé pour les requêtes SQL qui ne renvoient pas de lignes de résultats (requêtes de modification de schéma ou de données comme `INSERT`, `UPDATE`, `DELETE`, `CREATE TABLE`). Il retourne un objet `sql.Result` qui contient le nombre de lignes affectées (`RowsAffected()`) et le dernier ID inséré (`LastInsertId()`).
- **`db.Query()`** : Utilisé pour les requêtes de lecture (`SELECT`) qui renvoient plusieurs lignes. Il renvoie un objet `*sql.Rows` que l'on parcourt avec une boucle `for rows.Next()` et que l'on doit impérativement fermer (`rows.Close()`) pour libérer la connexion réseau/fichiers.

---

## Comparatif des Approches : `database/sql` vs `sqlx` vs `GORM`

| Caractéristique | `database/sql` (Standard) | `sqlx` (Extension) | `GORM` (ORM) |
| :--- | :--- | :--- | :--- |
| **Niveau d'Abstraction** | Bas niveau | Moyen niveau (SQL transparent) | Haut niveau (Abstraction SQL complète) |
| **Écriture du SQL** | Manuelle obligatoire | Manuelle obligatoire | Automatique (généré par l'ORM) |
| **Mapping Struct Go** | Entièrement manuel (`rows.Scan`) | Automatique (`db.Select`, `db.Get`) | Automatique via les méthodes d'API GORM |
| **Auto-migration** | Non existant | Non existant | Intégré (`db.AutoMigrate`) |
| **Performances** | Maximales (sans surcoût d'abstraction) | Excellentes (très proche du natif) | Légère perte de performance (dû à la réflexion) |
| **Boilerplate Code** | Très élevé (Next, Scan, gestion des erreurs fine) | Très faible (lecture directe en 1 ligne) | Quasi nul pour les requêtes simples |
| **Gestion des relations** | Manuelle (avec JOIN complexes et scans) | Manuelle (simplifiée par les structures imbriquées) | Automatisée via les tags GORM (Preload, Associations) |

### 1. `database/sql`
* **Avantages** : Aucun package externe (bibliothèque standard), contrôle absolu sur les requêtes SQL et les performances, aucun surcoût mémoire ou CPU.
* **Inconvénients** : Énormément de code répétitif (boilerplate), gestion des pointeurs de structures et du mapping par colonne pénible, les erreurs de types ou d'ordre de colonnes sont faciles à faire.
* **Idéal pour** : Des applications à haute performance, des requêtes très spécifiques ou optimisées manuellement, ou des microservices ultra-légers.

### 2. `sqlx`
* **Avantages** : Conserve la puissance de contrôle du SQL pur, réduit drastiquement le boilerplate grâce à `Select` et `Get` qui lisent directement dans des slices/structs en se basant sur les tags `db:"..."`, se substitue de façon transparente à `database/sql`.
* **Inconvénients** : Nécessite toujours d'écrire tout le SQL manuellement et de gérer la structure des tables.
* **Idéal pour** : La majorité des applications qui veulent écrire du SQL propre et performant tout en gardant une base de code Go concise et lisible.

### 3. `GORM`
* **Avantages** : Productivité maximale. Génère le SQL automatiquement pour toutes les opérations courantes. Auto-migration robuste. Abstraction totale des relations complexes (1-to-many, many-to-many).
* **Inconvénients** : Courbe d'apprentissage des spécificités et comportements de l'API GORM élevée, pertes de performance liées à la génération dynamique et la réflexion, débogage des requêtes SQL complexes difficile (GORM peut parfois générer du SQL sous-optimal).
* **Idéal pour** : Des projets CRUD classiques (rapidité de développement), des prototypes, ou des projets complexes avec de nombreuses relations de tables complexes où le SQL brut deviendrait lourd à maintenir.
