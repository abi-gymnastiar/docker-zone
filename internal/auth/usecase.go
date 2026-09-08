package auth

import (
	"errors"
	"net/http"
	"time"
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
