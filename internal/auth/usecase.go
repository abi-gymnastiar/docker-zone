package auth

import (
	"errors"
	"net/http"
	"time"

	"dashboard/internal/domain"
)

const sessionCookie = "dashboard_session"

type UseCase struct{ repo *Repository }

func NewUseCase(repo *Repository) *UseCase { return &UseCase{repo: repo} }

func (u *UseCase) Login(w http.ResponseWriter, username, password string) (User, error) {
	user, err := u.repo.Authenticate(username, password)
	if err != nil {
		return User{}, err
	}
	token, err := u.repo.CreateSession(user.ID, 24*time.Hour)
	if err != nil {
		return User{}, err
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: token, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: 86400})
	return user, nil
}

func (u *UseCase) Logout(w http.ResponseWriter, r *http.Request) error {
	cookie, err := r.Cookie(sessionCookie)
	if err == nil {
		if err := u.repo.DeleteSession(cookie.Value); err != nil {
			return err
		}
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: "/", MaxAge: -1, HttpOnly: true})
	return nil
}

func (u *UseCase) Current(r *http.Request) (User, error) {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil {
		return User{}, errors.New("not authenticated")
	}
	return u.repo.UserForSession(cookie.Value)
}

func (u *UseCase) Allowed(user User, groups []string, permission string) (bool, error) {
	return u.repo.Allowed(user, groups, permission)
}

func (u *UseCase) EnsureGroups(names []string) error { return u.repo.EnsureGroups(names) }
func (u *UseCase) ListUsers() ([]AdminUser, error)   { return u.repo.ListUsers() }
func (u *UseCase) CreateUser(username, password, role string) error {
	return u.repo.CreateUser(username, password, role)
}
func (u *UseCase) ListGroups() ([]AdminGroup, error) { return u.repo.ListGroups() }
func (u *UseCase) CreateGroup(name string) error     { return u.repo.CreateGroup(name) }
func (u *UseCase) SetGroupMember(groupID, userID int64, role string) error {
	return u.repo.SetGroupMember(groupID, userID, role)
}
func (u *UseCase) SetServiceGroups(serviceName string, groups []string) error {
	return u.repo.SetServiceGroups(serviceName, groups)
}
func (u *UseCase) ServiceGroups(serviceName string, defaults []string) ([]string, error) {
	return u.repo.ServiceGroups(serviceName, defaults)
}
func (u *UseCase) SetServiceMetadata(name, description string, enabled bool, actions []string) error {
	return u.repo.SetServiceMetadata(name, description, enabled, actions)
}
func (u *UseCase) UpsertService(service domain.ServiceConfig, containerID string, orphaned bool) error {
	return u.repo.UpsertService(service, containerID, orphaned)
}
func (u *UseCase) ListServices() ([]domain.ServiceConfig, error) { return u.repo.ListServices() }
func (u *UseCase) SyncServices(discovered []domain.DiscoveredService) error {
	if err := u.repo.EnsureGroups(discoveredGroups(discovered)); err != nil {
		return err
	}
	if err := u.repo.SyncServices(discovered); err != nil {
		return err
	}
	for _, service := range discovered {
		groups, err := u.repo.ServiceGroups(service.Name, nil)
		if err != nil {
			return err
		}
		if len(groups) == 0 && len(service.Groups) > 0 {
			if err := u.repo.SetServiceGroups(service.Name, service.Groups); err != nil {
				return err
			}
		}
	}
	return nil
}

func discoveredGroups(services []domain.DiscoveredService) []string {
	seen := map[string]bool{}
	var groups []string
	for _, service := range services {
		for _, group := range service.Groups {
			if !seen[group] {
				seen[group] = true
				groups = append(groups, group)
			}
		}
	}
	return groups
}
