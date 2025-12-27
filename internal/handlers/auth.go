package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/findosh/truenorth/internal/middleware"
	"github.com/findosh/truenorth/internal/services/auth"
)

// LoginPage renders the login page
func (h *Handler) LoginPage(w http.ResponseWriter, r *http.Request) {
	// If already logged in, redirect to dashboard
	if user := middleware.GetUser(r); user != nil {
		h.redirect(w, r, "/dashboard")
		return
	}

	data := map[string]interface{}{
		"Title": "Login - TrueNorth",
		"Error": r.URL.Query().Get("error"),
	}
	h.render(w, "login.html", data)
}

// Login handles login form submission
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.redirect(w, r, "/login?error=Invalid+request")
		return
	}

	email := strings.TrimSpace(r.FormValue("email"))
	password := r.FormValue("password")

	if email == "" || password == "" {
		h.redirect(w, r, "/login?error=Email+and+password+required")
		return
	}

	result, err := h.authService.Login(auth.LoginInput{
		Email:    email,
		Password: password,
	})
	if err != nil {
		h.redirect(w, r, "/login?error=Invalid+credentials")
		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    result.Token,
		Path:     "/",
		Expires:  result.Expires,
		HttpOnly: true,
		Secure:   h.cfg.IsProduction(),
		SameSite: http.SameSiteLaxMode,
	})

	h.redirect(w, r, "/dashboard")
}

// RegisterPage renders the registration page
func (h *Handler) RegisterPage(w http.ResponseWriter, r *http.Request) {
	// If already logged in, redirect to dashboard
	if user := middleware.GetUser(r); user != nil {
		h.redirect(w, r, "/dashboard")
		return
	}

	data := map[string]interface{}{
		"Title": "Register - TrueNorth",
		"Error": r.URL.Query().Get("error"),
	}
	h.render(w, "register.html", data)
}

// Register handles registration form submission
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.redirect(w, r, "/register?error=Invalid+request")
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	email := strings.TrimSpace(r.FormValue("email"))
	password := r.FormValue("password")
	confirmPassword := r.FormValue("confirm_password")

	// Validation
	if name == "" || email == "" || password == "" {
		h.redirect(w, r, "/register?error=All+fields+required")
		return
	}

	if len(password) < 8 {
		h.redirect(w, r, "/register?error=Password+must+be+at+least+8+characters")
		return
	}

	if password != confirmPassword {
		h.redirect(w, r, "/register?error=Passwords+do+not+match")
		return
	}

	// Create user
	user, err := h.authService.Register(auth.RegisterInput{
		Email:    email,
		Password: password,
		Name:     name,
	})
	if err != nil {
		if err == auth.ErrEmailExists {
			h.redirect(w, r, "/register?error=Email+already+registered")
			return
		}
		h.redirect(w, r, "/register?error=Registration+failed")
		return
	}

	// Auto-login after registration
	result, err := h.authService.Login(auth.LoginInput{
		Email:    user.Email,
		Password: password,
	})
	if err != nil {
		h.redirect(w, r, "/login?error=Registration+successful,+please+login")
		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    result.Token,
		Path:     "/",
		Expires:  result.Expires,
		HttpOnly: true,
		Secure:   h.cfg.IsProduction(),
		SameSite: http.SameSiteLaxMode,
	})

	h.redirect(w, r, "/dashboard")
}

// Logout handles user logout
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user != nil {
		h.authService.Logout(user.ID)
	}

	// Clear session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
	})

	h.redirect(w, r, "/login")
}
