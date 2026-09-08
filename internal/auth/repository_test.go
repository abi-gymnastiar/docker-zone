package auth

import (
	"path/filepath"
	"testing"
	"time"
)

func TestBootstrapLoginSessionAndPermissions(t *testing.T) {
	repo, err := Open(filepath.Join(t.TempDir(), "dashboard.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()

	if err := repo.Bootstrap("admin", "secret"); err != nil {
		t.Fatal(err)
	}
	user, err := repo.Authenticate("admin", "secret")
	if err != nil {
		t.Fatal(err)
	}
	token, err := repo.CreateSession(user.ID, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	sessionUser, err := repo.UserForSession(token)
	if err != nil {
		t.Fatal(err)
	}
	if sessionUser.Username != "admin" || sessionUser.Role != "admin" {
		t.Fatalf("unexpected session user: %+v", sessionUser)
	}
	allowed, err := repo.Allowed(user, []string{"minecraft"}, "services:restart")
	if err != nil {
		t.Fatal(err)
	}
	if !allowed {
		t.Fatal("admin should be allowed to restart services")
	}
}
