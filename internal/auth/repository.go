package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

type Repository struct{ db *sql.DB }

func Open(path string) (*Repository, error) {
	db, err := sql.Open("sqlite", path+"?_foreign_keys=on")
	if err != nil {
		return nil, err
	}
	repo := &Repository{db: db}
	if err := repo.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return repo, nil
}

func (r *Repository) migrate() error {
	_, err := r.db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'viewer',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS sessions (
			token_hash TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			expires_at DATETIME NOT NULL
		);
		CREATE TABLE IF NOT EXISTS groups (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE
		);
		CREATE TABLE IF NOT EXISTS group_members (
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			group_id INTEGER NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
			role TEXT NOT NULL DEFAULT 'viewer',
			PRIMARY KEY (user_id, group_id)
		);
	`)
	return err
}

func (r *Repository) Bootstrap(username, password string) error {
	var count int
	if err := r.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = r.db.Exec("INSERT INTO users (username, password_hash, role) VALUES (?, ?, 'admin')", username, hash)
	return err
}

func (r *Repository) Authenticate(username, password string) (User, error) {
	var user User
	var hash string
	err := r.db.QueryRow("SELECT id, username, role, password_hash FROM users WHERE username = ?", username).
		Scan(&user.ID, &user.Username, &user.Role, &hash)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, errors.New("invalid username or password")
	}
	if err != nil {
		return User{}, err
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return User{}, errors.New("invalid username or password")
	}
	return user, nil
}

func (r *Repository) CreateSession(userID int64, duration time.Duration) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	token := hex.EncodeToString(raw)
	hash := sha256.Sum256([]byte(token))
	_, err := r.db.Exec("INSERT INTO sessions (token_hash, user_id, expires_at) VALUES (?, ?, ?)",
		hex.EncodeToString(hash[:]), userID, time.Now().Add(duration))
	return token, err
}

func (r *Repository) UserForSession(token string) (User, error) {
	hash := sha256.Sum256([]byte(token))
	var user User
	err := r.db.QueryRow(`
		SELECT u.id, u.username, u.role
		FROM sessions s JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = ? AND s.expires_at > CURRENT_TIMESTAMP`,
		hex.EncodeToString(hash[:])).Scan(&user.ID, &user.Username, &user.Role)
	return user, err
}

func (r *Repository) DeleteSession(token string) error {
	hash := sha256.Sum256([]byte(token))
	_, err := r.db.Exec("DELETE FROM sessions WHERE token_hash = ?", hex.EncodeToString(hash[:]))
	return err
}

func (r *Repository) Allowed(user User, serviceGroups []string, permission string) (bool, error) {
	if user.Role == "admin" {
		return true, nil
	}
	if len(serviceGroups) == 0 {
		return false, nil
	}
	roles := rolesForPermission(permission)
	query := `SELECT EXISTS (
		SELECT 1 FROM group_members gm
		JOIN groups g ON g.id = gm.group_id
		WHERE gm.user_id = ? AND gm.role IN (` + placeholders(len(roles)) + `)
		AND g.name IN (` + placeholders(len(serviceGroups)) + `)
	)`
	args := []any{user.ID}
	for _, role := range roles {
		args = append(args, role)
	}
	for _, group := range serviceGroups {
		args = append(args, group)
	}
	var allowed bool
	err := r.db.QueryRow(query, args...).Scan(&allowed)
	return allowed, err
}

func rolesForPermission(permission string) []string {
	roles := []string{"viewer", "log_viewer", "operator"}
	if permission == "services:logs" {
		roles = []string{"log_viewer", "operator"}
	}
	if permission == "services:start" || permission == "services:stop" || permission == "services:restart" {
		roles = []string{"operator"}
	}
	return roles
}

func placeholders(count int) string {
	result := "?"
	for i := 1; i < count; i++ {
		result += ",?"
	}
	return result
}

func (r *Repository) Close() error { return r.db.Close() }
