package frontend

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/tofudns/tofudns/internal/storage"
)

const (
	// Authentication settings
	cookieName    = "tofudns_auth"
	jwtExpiration = 24 * time.Hour
	otpExpiration = 10 * time.Minute
	otpLength     = 6
)

// Claims defines the JWT claims structure
type Claims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

// Context key type to avoid collisions
type contextKey string

// UserEmailKey is the context key for the user email
const UserEmailKey contextKey = "userEmail"

// UserIDKey is the context key for the user ID (UUID)
const UserIDKey contextKey = "userID"

// authMiddleware checks for a valid JWT token and redirects to login if not present
func (s *Service) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for login pages
		if strings.HasPrefix(r.URL.Path, "/auth/") {
			next.ServeHTTP(w, r)
			return
		}

		// Get the JWT token from cookie
		cookie, err := r.Cookie(cookieName)
		if err != nil {
			// Redirect to login page if cookie not present
			http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
			return
		}

		// Parse and validate the JWT token
		token, err := jwt.ParseWithClaims(cookie.Value, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			// Validate the signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			// Return the secret key used for signing
			return []byte(s.jwtSecret), nil
		})

		if err != nil || !token.Valid {
			// Clear the invalid cookie
			http.SetCookie(w, &http.Cookie{
				Name:     cookieName,
				Value:    "",
				Path:     "/",
				Expires:  time.Unix(0, 0),
				HttpOnly: true,
				Secure:   r.TLS != nil,
				SameSite: http.SameSiteLaxMode,
			})
			// Redirect to login page
			http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
			return
		}

		// Extract the claims
		if claims, ok := token.Claims.(*Claims); ok && token.Valid {
			email := claims.Email

			// Look up the user ID based on email
			ctx := r.Context()
			user, err := s.db.GetUserByEmail(ctx, email)
			// If user doesn't exist, create the user
			if err != nil {
				// Only create if it's a "not found" error
				newUser, createErr := s.db.CreateUser(ctx, email)
				if createErr != nil {
					s.logger.Error("Failed to create user", "error", createErr, "email", email)
					http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
					return
				}
				user = newUser
			}

			// Add both email and UUID to the context
			ctx = context.WithValue(ctx, UserEmailKey, email)
			ctx = context.WithValue(ctx, UserIDKey, user.ID)

			// Continue with the updated context
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// If we can't extract claims, redirect to login
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
	})
}

// setupAuthRoutes registers authentication-related routes
func (s *Service) setupAuthRoutes(r chi.Router) {
	r.Get("/auth/login", s.handleLoginPage)
	r.Post("/auth/login", s.handleLoginSubmit)
	r.Get("/auth/verify", s.handleVerifyPage)
	r.Post("/auth/verify", s.handleVerifyOTP)
	r.Get("/auth/logout", s.handleLogout)
}

// handleLoginPage displays the login form
func (s *Service) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Error": r.URL.Query().Get("error"),
	}
	s.templates.ExecuteTemplate(w, "login.html", data)
}

// handleLoginSubmit processes the login form submission
func (s *Service) handleLoginSubmit(w http.ResponseWriter, r *http.Request) {
	// Parse the form
	if err := r.ParseForm(); err != nil {
		s.logger.Error("Failed to parse login form", "error", err)
		http.Redirect(w, r, "/auth/login?error=Invalid+request", http.StatusSeeOther)
		return
	}

	// Get the email from the form
	email := strings.TrimSpace(r.Form.Get("email"))
	if email == "" {
		http.Redirect(w, r, "/auth/login?error=Email+is+required", http.StatusSeeOther)
		return
	}

	// Generate a random OTP code
	otp, err := generateOTP(otpLength)
	if err != nil {
		s.logger.Error("Failed to generate OTP", "error", err)
		http.Redirect(w, r, "/auth/login?error=Server+error", http.StatusSeeOther)
		return
	}

	// Store the OTP in the database
	ctx := r.Context()
	expiresAt := time.Now().Add(otpExpiration)
	_, err = s.db.CreateOTP(ctx, storage.CreateOTPParams{
		Email:     email,
		Code:      otp,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		s.logger.Error("Failed to create OTP", "error", err)
		http.Redirect(w, r, "/auth/login?error=Server+error", http.StatusSeeOther)
		return
	}

	// Send the OTP via email
	err = s.emailService.SendOTP(email, otp)
	if err != nil {
		s.logger.Error("Failed to send OTP email", "error", err)
		http.Redirect(w, r, "/auth/login?error=Failed+to+send+email", http.StatusSeeOther)
		return
	}

	// Redirect to verification page
	http.Redirect(w, r, fmt.Sprintf("/auth/verify?email=%s", email), http.StatusSeeOther)
}

// handleVerifyPage displays the OTP verification form
func (s *Service) handleVerifyPage(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	if email == "" {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	data := map[string]interface{}{
		"Email": email,
		"Error": r.URL.Query().Get("error"),
	}
	s.templates.ExecuteTemplate(w, "verify_otp.html", data)
}

// handleVerifyOTP processes the OTP verification
func (s *Service) handleVerifyOTP(w http.ResponseWriter, r *http.Request) {
	// Parse the form
	if err := r.ParseForm(); err != nil {
		s.logger.Error("Failed to parse verification form", "error", err)
		http.Redirect(w, r, "/auth/login?error=Invalid+request", http.StatusSeeOther)
		return
	}

	// Get email and code from the form
	email := strings.TrimSpace(r.Form.Get("email"))
	code := strings.TrimSpace(r.Form.Get("code"))

	if email == "" || code == "" {
		http.Redirect(w, r, "/auth/login?error=Invalid+request", http.StatusSeeOther)
		return
	}

	ctx := r.Context()

	// Validate and consume the OTP
	_, err := s.db.ValidateAndConsumeOTP(ctx, storage.ValidateAndConsumeOTPParams{
		Email: email,
		Code:  code,
	})
	if err != nil {
		s.logger.Error("Failed to validate OTP", "error", err)
		http.Redirect(w, r, fmt.Sprintf("/auth/verify?email=%s&error=Invalid+code", email), http.StatusSeeOther)
		return
	}

	// If we get here, the OTP is valid and consumed
	// Create a JWT token for the user
	token, err := s.createJWTToken(email)
	if err != nil {
		s.logger.Error("Failed to create JWT token", "error", err)
		http.Redirect(w, r, "/auth/login?error=Server+error", http.StatusSeeOther)
		return
	}

	// Set the JWT as a cookie
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		Expires:  time.Now().Add(jwtExpiration),
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})

	// Redirect to the home page
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// handleLogout logs out the user by clearing the auth cookie
func (s *Service) handleLogout(w http.ResponseWriter, r *http.Request) {
	// Clear the auth cookie
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})

	// Redirect to login page
	http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
}

// Helper functions

// generateOTP generates a random OTP code of the specified length
func generateOTP(length int) (string, error) {
	// Create a byte slice to hold the random bytes
	randomBytes := make([]byte, length)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}

	// Convert the random bytes to a numeric string
	var otpBuilder strings.Builder
	for _, b := range randomBytes {
		digit := int(b) % 10
		otpBuilder.WriteString(strconv.Itoa(digit))
	}

	// Trim to required length
	return otpBuilder.String()[:length], nil
}

// createJWTToken creates a JWT token for the email
func (s *Service) createJWTToken(email string) (string, error) {
	expirationTime := time.Now().Add(jwtExpiration)
	claims := &Claims{
		Email: email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	// Create a new token with the claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign the token with the secret key
	return token.SignedString([]byte(s.jwtSecret))
}

// Configuration helpers

// getJWTSecret returns the JWT secret key
func getJWTSecret() string {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		// Use a default secret for development
		// In production, this should be properly configured
		return "tofudns_jwt_secret_replace_in_production"
	}
	return secret
}

// getPostmarkServerToken returns the Postmark server token
func getPostmarkServerToken() string {
	token := os.Getenv("POSTMARK_SERVER_TOKEN")
	if token == "" {
		// Log a warning but don't fail
		// In development, emails won't be sent
		return "POSTMARK_SERVER_TOKEN_NOT_SET"
	}
	return token
}

// getEmailFrom returns the sender email address
func getEmailFrom() string {
	from := os.Getenv("EMAIL_FROM")
	if from == "" {
		return "auth@tofudns.net"
	}
	return from
}

// getUserEmail gets the user email from the request context
func getUserEmail(r *http.Request) string {
	if email, ok := r.Context().Value(UserEmailKey).(string); ok {
		return email
	}
	return ""
}

// getUserID gets the user ID (UUID) from the request context
func getUserID(r *http.Request) uuid.UUID {
	if userID, ok := r.Context().Value(UserIDKey).(uuid.UUID); ok {
		return userID
	}
	return uuid.UUID{} // Return a zero UUID if not found
}
