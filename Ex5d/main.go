package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/glebarez/go-sqlite" // Pilote SQLite pur Go (sans CGO)
	"github.com/glebarez/sqlite"     // Pilote SQLite GORM pur Go (sans CGO)
	"github.com/jmoiron/sqlx"
	"gorm.io/gorm"
)

// ============================================================================
// STRUCTURE DE DONNÉES
// ============================================================================

// User représente un utilisateur dans la base de données.
type User struct {
	ID    int    `db:"id" gorm:"primaryKey;autoIncrement"`
	Name  string `db:"name"`
	Email string `db:"email"`
}

// ============================================================================
// EXERCICE 1.2 : OPÉRATIONS CRUD AVEC LE PACKAGE STANDARD database/sql
// ============================================================================

// CreateUserSQL insère un nouvel utilisateur et retourne son ID.
func CreateUserSQL(db *sql.DB, user User) (int64, error) {
	query := "INSERT INTO users (name, email) VALUES (?, ?)"
	result, err := db.Exec(query, user.Name, user.Email)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// GetUsersSQL récupère tous les utilisateurs de la table.
func GetUsersSQL(db *sql.DB) ([]User, error) {
	query := "SELECT id, name, email FROM users"
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		// Scan nécessite de mapper manuellement chaque colonne à un champ
		if err := rows.Scan(&u.ID, &u.Name, &u.Email); err != nil {
			return nil, err
		}
		users = append(users, u)
	}

	// Toujours vérifier les erreurs sur la boucle rows
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return users, nil
}

// GetUserByIDSQL récupère un utilisateur par son ID. Retourne nil si non trouvé.
func GetUserByIDSQL(db *sql.DB, id int) (*User, error) {
	query := "SELECT id, name, email FROM users WHERE id = ?"
	var u User
	err := db.QueryRow(query, id).Scan(&u.ID, &u.Name, &u.Email)
	if err == sql.ErrNoRows {
		return nil, nil // Non trouvé
	} else if err != nil {
		return nil, err
	}
	return &u, nil
}

// UpdateUserSQL met à jour le nom et l'email d'un utilisateur par son ID.
func UpdateUserSQL(db *sql.DB, user User) error {
	query := "UPDATE users SET name = ?, email = ? WHERE id = ?"
	_, err := db.Exec(query, user.Name, user.Email, user.ID)
	return err
}

// DeleteUserSQL supprime un utilisateur par son ID.
func DeleteUserSQL(db *sql.DB, id int) error {
	query := "DELETE FROM users WHERE id = ?"
	_, err := db.Exec(query, id)
	return err
}

// ============================================================================
// EXERCICE 1.3 : AMÉLIORATION AVEC sqlx
// ============================================================================

// CreateUserSQLX insère un utilisateur en utilisant sqlx (similaire à sql).
func CreateUserSQLX(db *sqlx.DB, user User) (int64, error) {
	query := "INSERT INTO users (name, email) VALUES (?, ?)"
	result, err := db.Exec(query, user.Name, user.Email)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// GetUsersSQLX récupère tous les utilisateurs en mappant directement vers la struct.
func GetUsersSQLX(db *sqlx.DB) ([]User, error) {
	var users []User
	query := "SELECT id, name, email FROM users"
	// Select gère l'exécution, la boucle Next/Scan et la fermeture des lignes automatiquement
	err := db.Select(&users, query)
	return users, err
}

// GetUserByIDSQLX récupère un utilisateur par son ID directement dans la struct.
func GetUserByIDSQLX(db *sqlx.DB, id int) (*User, error) {
	var u User
	query := "SELECT id, name, email FROM users WHERE id = ?"
	// Get permet de charger un seul enregistrement directement dans le pointeur struct
	err := db.Get(&u, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &u, nil
}

// UpdateUserSQLX met à jour un utilisateur en utilisant sqlx.
func UpdateUserSQLX(db *sqlx.DB, user User) error {
	query := "UPDATE users SET name = ?, email = ? WHERE id = ?"
	_, err := db.Exec(query, user.Name, user.Email, user.ID)
	return err
}

// DeleteUserSQLX supprime un utilisateur par ID en utilisant sqlx.
func DeleteUserSQLX(db *sqlx.DB, id int) error {
	query := "DELETE FROM users WHERE id = ?"
	_, err := db.Exec(query, id)
	return err
}

// ============================================================================
// EXERCICE 2.2 : OPÉRATIONS CRUD AVEC GORM
// ============================================================================

// CreateUserGORM insère un nouvel utilisateur (l'ID est mis à jour dans le pointeur user).
// Note : Nous passons un pointeur *User pour permettre à GORM de renseigner l'ID auto-incrémenté.
func CreateUserGORM(db *gorm.DB, user *User) error {
	result := db.Create(user)
	return result.Error
}

// GetUsersGORM récupère tous les utilisateurs.
func GetUsersGORM(db *gorm.DB) ([]User, error) {
	var users []User
	result := db.Find(&users)
	return users, result.Error
}

// GetUserByIDGORM récupère un utilisateur par son ID.
func GetUserByIDGORM(db *gorm.DB, id int) (*User, error) {
	var u User
	result := db.First(&u, id)
	if result.Error == gorm.ErrRecordNotFound {
		return nil, nil
	} else if result.Error != nil {
		return nil, result.Error
	}
	return &u, nil
}

// UpdateUserGORM met à jour un utilisateur existant.
func UpdateUserGORM(db *gorm.DB, user *User) error {
	// Save enregistre tous les champs dans la base
	result := db.Save(user)
	return result.Error
}

// DeleteUserGORM supprime un utilisateur par ID.
func DeleteUserGORM(db *gorm.DB, id int) error {
	// GORM nécessite de spécifier le type ou le modèle à supprimer
	result := db.Delete(&User{}, id)
	return result.Error
}

// ============================================================================
// EXECUTION ET TESTS DANS main()
// ============================================================================

func main() {
	// 0. Nettoyage des anciennes bases de données de test pour repartir sur du propre
	_ = os.Remove("./test.db")
	_ = os.Remove("./gorm_test.db")

	fmt.Println("=== ÉTAPE 1 : INITIALISATION DATABASE/SQL & SQLX (test.db) ===")
	
	// Connexion database/sql - Remarque : Le driver pur Go "github.com/glebarez/go-sqlite" s'enregistre sous le nom "sqlite"
	rawDB, err := sql.Open("sqlite", "./test.db")
	if err != nil {
		log.Fatal("Échec sql.Open:", err)
	}
	defer rawDB.Close()

	if err = rawDB.Ping(); err != nil {
		log.Fatal("Échec rawDB.Ping:", err)
	}
	fmt.Println("[SQL] Connecté à la base SQLite pur Go (test.db).")

	// Création manuelle de la table users
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		email TEXT UNIQUE NOT NULL
	);`
	_, err = rawDB.Exec(createTableSQL)
	if err != nil {
		log.Fatal("Échec de la création de la table avec database/sql:", err)
	}
	fmt.Println("[SQL] Table 'users' créée avec succès.")

	// --- Tests CRUD avec database/sql standard ---
	fmt.Println("\n--- Tests CRUD avec database/sql ---")
	
	// C - Create
	alice := User{Name: "Alice SQL", Email: "alice.sql@example.com"}
	aliceID, err := CreateUserSQL(rawDB, alice)
	if err != nil {
		log.Fatal("Erreur CreateUserSQL:", err)
	}
	fmt.Printf("[SQL] Utilisateur créé : Alice SQL avec l'ID %d\n", aliceID)

	bob := User{Name: "Bob SQL", Email: "bob.sql@example.com"}
	bobID, err := CreateUserSQL(rawDB, bob)
	if err != nil {
		log.Fatal("Erreur CreateUserSQL:", err)
	}
	fmt.Printf("[SQL] Utilisateur créé : Bob SQL avec l'ID %d\n", bobID)

	// R - Read All
	users, err := GetUsersSQL(rawDB)
	if err != nil {
		log.Fatal("Erreur GetUsersSQL:", err)
	}
	fmt.Printf("[SQL] Liste complète des utilisateurs (%d) :\n", len(users))
	for _, u := range users {
		fmt.Printf(" - ID: %d, Nom: %s, Email: %s\n", u.ID, u.Name, u.Email)
	}

	// R - Read One by ID
	userByID, err := GetUserByIDSQL(rawDB, int(aliceID))
	if err != nil {
		log.Fatal("Erreur GetUserByIDSQL:", err)
	}
	if userByID != nil {
		fmt.Printf("[SQL] Récupération ID %d -> Nom: %s, Email: %s\n", aliceID, userByID.Name, userByID.Email)
	}

	// U - Update
	if userByID != nil {
		userByID.Name = "Alice SQL Modifiée"
		err = UpdateUserSQL(rawDB, *userByID)
		if err != nil {
			log.Fatal("Erreur UpdateUserSQL:", err)
		}
		fmt.Printf("[SQL] Utilisateur ID %d mis à jour.\n", userByID.ID)
	}

	// D - Delete
	err = DeleteUserSQL(rawDB, int(bobID))
	if err != nil {
		log.Fatal("Erreur DeleteUserSQL:", err)
	}
	fmt.Printf("[SQL] Utilisateur ID %d (Bob) supprimé.\n", bobID)

	// Vérification finale database/sql
	usersFinal, _ := GetUsersSQL(rawDB)
	fmt.Printf("[SQL] Liste après opérations CRUD (%d restants) :\n", len(usersFinal))
	for _, u := range usersFinal {
		fmt.Printf(" - ID: %d, Nom: %s, Email: %s\n", u.ID, u.Name, u.Email)
	}


	// --- Tests CRUD avec sqlx ---
	fmt.Println("\n=== ÉTAPE 2 : TESTS CRUD AVEC SQLX (Sur la même base test.db) ===")
	
	// Connexion sqlx.Open
	dbX, err := sqlx.Open("sqlite", "./test.db")
	if err != nil {
		log.Fatal("Échec sqlx.Open:", err)
	}
	defer dbX.Close()

	if err = dbX.Ping(); err != nil {
		log.Fatal("Échec dbX.Ping:", err)
	}
	fmt.Println("[SQLX] Connecté à sqlite via sqlx.Open (pur Go).")

	// C - Create
	charlie := User{Name: "Charlie SQLX", Email: "charlie.sqlx@example.com"}
	charlieID, err := CreateUserSQLX(dbX, charlie)
	if err != nil {
		log.Fatal("Erreur CreateUserSQLX:", err)
	}
	fmt.Printf("[SQLX] Utilisateur créé : Charlie avec l'ID %d\n", charlieID)

	// R - Read All (avec db.Select)
	usersX, err := GetUsersSQLX(dbX)
	if err != nil {
		log.Fatal("Erreur GetUsersSQLX:", err)
	}
	fmt.Printf("[SQLX] Liste complète récupérée via db.Select (%d) :\n", len(usersX))
	for _, u := range usersX {
		fmt.Printf(" - ID: %d, Nom: %s, Email: %s\n", u.ID, u.Name, u.Email)
	}

	// R - Read One by ID (avec db.Get)
	userXByID, err := GetUserByIDSQLX(dbX, int(charlieID))
	if err != nil {
		log.Fatal("Erreur GetUserByIDSQLX:", err)
	}
	if userXByID != nil {
		fmt.Printf("[SQLX] Récupération ID %d via db.Get -> Nom: %s, Email: %s\n", charlieID, userXByID.Name, userXByID.Email)
	}

	// U - Update
	if userXByID != nil {
		userXByID.Name = "Charlie SQLX Modifié"
		err = UpdateUserSQLX(dbX, *userXByID)
		if err != nil {
			log.Fatal("Erreur UpdateUserSQLX:", err)
		}
		fmt.Printf("[SQLX] Utilisateur ID %d mis à jour.\n", userXByID.ID)
	}

	// D - Delete
	err = DeleteUserSQLX(dbX, int(charlieID))
	if err != nil {
		log.Fatal("Erreur DeleteUserSQLX:", err)
	}
	fmt.Printf("[SQLX] Utilisateur ID %d (Charlie) supprimé.\n", charlieID)


	// --- Tests CRUD avec GORM ---
	fmt.Println("\n=== ÉTAPE 3 : TESTS CRUD AVEC GORM (gorm_test.db) ===")

	// Connexion GORM
	gormDB, err := gorm.Open(sqlite.Open("gorm_test.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("Échec gorm.Open:", err)
	}
	fmt.Println("[GORM] Connecté à sqlite via gorm.Open (pur Go).")

	// Auto-migration
	err = gormDB.AutoMigrate(&User{})
	if err != nil {
		log.Fatal("Échec AutoMigrate GORM:", err)
	}
	fmt.Println("[GORM] Auto-migration effectuée avec succès.")

	// C - Create
	david := User{Name: "David GORM", Email: "david.gorm@example.com"}
	err = CreateUserGORM(gormDB, &david)
	if err != nil {
		log.Fatal("Erreur CreateUserGORM:", err)
	}
	// L'ID est maintenant mis à jour directement dans la struct
	fmt.Printf("[GORM] Utilisateur créé : %s avec l'ID %d (assigné par GORM)\n", david.Name, david.ID)

	eva := User{Name: "Eva GORM", Email: "eva.gorm@example.com"}
	err = CreateUserGORM(gormDB, &eva)
	if err != nil {
		log.Fatal("Erreur CreateUserGORM:", err)
	}
	fmt.Printf("[GORM] Utilisateur créé : %s avec l'ID %d\n", eva.Name, eva.ID)

	// R - Read All
	usersG, err := GetUsersGORM(gormDB)
	if err != nil {
		log.Fatal("Erreur GetUsersGORM:", err)
	}
	fmt.Printf("[GORM] Liste complète récupérée (%d) :\n", len(usersG))
	for _, u := range usersG {
		fmt.Printf(" - ID: %d, Nom: %s, Email: %s\n", u.ID, u.Name, u.Email)
	}

	// R - Read One by ID
	userGByID, err := GetUserByIDGORM(gormDB, david.ID)
	if err != nil {
		log.Fatal("Erreur GetUserByIDGORM:", err)
	}
	if userGByID != nil {
		fmt.Printf("[GORM] Récupération ID %d -> Nom: %s, Email: %s\n", david.ID, userGByID.Name, userGByID.Email)
	}

	// U - Update
	if userGByID != nil {
		userGByID.Name = "David GORM Modifié"
		err = UpdateUserGORM(gormDB, userGByID)
		if err != nil {
			log.Fatal("Erreur UpdateUserGORM:", err)
		}
		fmt.Printf("[GORM] Utilisateur ID %d mis à jour.\n", userGByID.ID)
	}

	// D - Delete
	err = DeleteUserGORM(gormDB, eva.ID)
	if err != nil {
		log.Fatal("Erreur DeleteUserGORM:", err)
	}
	fmt.Printf("[GORM] Utilisateur ID %d (Eva) supprimé.\n", eva.ID)

	// Liste finale GORM
	usersGFinal, _ := GetUsersGORM(gormDB)
	fmt.Printf("[GORM] Liste finale après CRUD (%d restants) :\n", len(usersGFinal))
	for _, u := range usersGFinal {
		fmt.Printf(" - ID: %d, Nom: %s, Email: %s\n", u.ID, u.Name, u.Email)
	}
}
