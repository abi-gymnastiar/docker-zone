package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"dashboard/internal/domain"
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
		CREATE TABLE IF NOT EXISTS service_groups (
			service_name TEXT NOT NULL,
			group_id INTEGER NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
			PRIMARY KEY (service_name, group_id)
		);
		CREATE TABLE IF NOT EXISTS services (
			name TEXT PRIMARY KEY,
			container TEXT NOT NULL,
			container_id TEXT NOT NULL DEFAULT '',
			description TEXT NOT NULL DEFAULT '',
			actions TEXT NOT NULL DEFAULT 'start,stop,restart',
			orphaned INTEGER NOT NULL DEFAULT 0,
			enabled INTEGER NOT NULL DEFAULT 0
		);
	`)
	if err != nil {
		return err
	}
	_, err = r.db.Exec("ALTER TABLE services ADD COLUMN enabled INTEGER NOT NULL DEFAULT 0")
	if err != nil && !strings.Contains(err.Error(), "duplicate column") {
		return err
	}
	return nil
}

func (r *Repository) UpsertService(service domain.ServiceConfig, containerID string, orphaned bool) error {
	_, err := r.db.Exec(`INSERT INTO services (name, container, container_id, description, actions, orphaned, enabled)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(name) DO UPDATE SET container=excluded.container, container_id=excluded.container_id,
		orphaned=excluded.orphaned`,
		service.Name, service.Container, containerID, service.Description, strings.Join(service.Actions, ","), boolInt(orphaned), boolInt(service.Enabled))
	return err
}

func (r *Repository) ListServices() ([]domain.ServiceConfig, error) {
	rows, err := r.db.Query("SELECT name, container, container_id, description, actions, orphaned, enabled FROM services ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	services := make([]domain.ServiceConfig, 0)
	for rows.Next() {
		var service domain.ServiceConfig
		var actions string
		var orphaned, enabled int
		if err := rows.Scan(&service.Name, &service.Container, &service.ContainerID, &service.Description, &actions, &orphaned, &enabled); err != nil {
			return nil, err
		}
		service.Actions = splitCSV(actions)
		service.Orphaned = orphaned == 1
		service.Enabled = enabled == 1
		service.Groups, err = r.ServiceGroups(service.Name, nil)
		if err != nil {
			return nil, err
		}
		services = append(services, service)
	}
	return services, rows.Err()
}

func (r *Repository) SyncServices(discovered []domain.DiscoveredService) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec("UPDATE services SET orphaned = 1"); err != nil {
		_ = tx.Rollback()
		return err
	}
	for _, item := range discovered {
		actions := strings.Join(item.Actions, ",")
		if _, err := tx.Exec(`INSERT INTO services (name, container, container_id, description, actions, orphaned, enabled)
			VALUES (?, ?, ?, ?, ?, 0, 0)
			ON CONFLICT(name) DO UPDATE SET container=excluded.container, container_id=excluded.container_id,
			orphaned=0`,
			item.Name, item.Container, item.ContainerID, item.Description, actions); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func (r *Repository) SetServiceMetadata(name, description string, enabled bool, actions []string) error {
	_, err := r.db.Exec("UPDATE services SET description = ?, enabled = ?, actions = ? WHERE name = ?",
		description, boolInt(enabled), strings.Join(actions, ","), name)
	return err
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func splitCSV(value string) []string {
	var result []string
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			result = append(result, item)
		}
	}
	return result
}

func (r *Repository) EnsureGroups(names []string) error {
	for _, name := range names {
		if _, err := r.db.Exec("INSERT INTO groups (name) VALUES (?) ON CONFLICT(name) DO NOTHING", name); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) ListUsers() ([]AdminUser, error) {
	rows, err := r.db.Query("SELECT id, username, role FROM users ORDER BY username")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	users := make([]AdminUser, 0)
	for rows.Next() {
		var user AdminUser
		if err := rows.Scan(&user.ID, &user.Username, &user.Role); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

func (r *Repository) CreateUser(username, password, role string) error {
	if role != "admin" && role != "viewer" {
		return errors.New("invalid user role")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = r.db.Exec("INSERT INTO users (username, password_hash, role) VALUES (?, ?, ?)", username, hash, role)
	return err
}

func (r *Repository) ListGroups() ([]AdminGroup, error) {
	rows, err := r.db.Query("SELECT id, name FROM groups ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	groups := make([]AdminGroup, 0)
	for rows.Next() {
		var group AdminGroup
		if err := rows.Scan(&group.ID, &group.Name); err != nil {
			return nil, err
		}
		group.Members, err = r.groupMembers(group.ID)
		if err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}
	return groups, rows.Err()
}

func (r *Repository) groupMembers(groupID int64) ([]GroupMember, error) {
	rows, err := r.db.Query(`SELECT u.id, u.username, gm.role FROM group_members gm
		JOIN users u ON u.id = gm.user_id WHERE gm.group_id = ? ORDER BY u.username`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	members := make([]GroupMember, 0)
	for rows.Next() {
		var member GroupMember
		if err := rows.Scan(&member.UserID, &member.Username, &member.Role); err != nil {
			return nil, err
		}
		members = append(members, member)
	}
	return members, rows.Err()
}

func (r *Repository) CreateGroup(name string) error {
	_, err := r.db.Exec("INSERT INTO groups (name) VALUES (?)", name)
	return err
}

func (r *Repository) SetGroupMember(groupID, userID int64, role string) error {
	if role != "viewer" && role != "log_viewer" && role != "operator" {
		return errors.New("invalid group role")
	}
	_, err := r.db.Exec(`INSERT INTO group_members (user_id, group_id, role) VALUES (?, ?, ?)
		ON CONFLICT(user_id, group_id) DO UPDATE SET role = excluded.role`, userID, groupID, role)
	return err
}

func (r *Repository) SetServiceGroups(serviceName string, groups []string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM service_groups WHERE service_name = ?", serviceName); err != nil {
		_ = tx.Rollback()
		return err
	}
	for _, group := range groups {
		if _, err := tx.Exec(`INSERT INTO service_groups (service_name, group_id)
			SELECT ?, id FROM groups WHERE name = ?`, serviceName, group); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func (r *Repository) ServiceGroups(serviceName string, defaults []string) ([]string, error) {
	rows, err := r.db.Query(`SELECT g.name FROM service_groups sg JOIN groups g ON g.id = sg.group_id
		WHERE sg.service_name = ? ORDER BY g.name`, serviceName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var groups []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		groups = append(groups, name)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(groups) == 0 {
		return defaults, nil
	}
	return groups, nil
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
