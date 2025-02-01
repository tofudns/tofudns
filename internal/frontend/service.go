package frontend

import (
	"embed"
	"html/template"
	"log/slog"
	"net/http"

	"github.com/tofudns/tofudns/internal/storage"
)

//go:embed templates/*
var templateFS embed.FS

type Service struct {
	templates *template.Template
}

func New(
	logger *slog.Logger,
	dbClient storage.Querier,
) (*Service, error) {
	// Parse all templates from the embedded filesystem
	tmpl, err := template.ParseFS(templateFS, "templates/*.html")
	if err != nil {
		return nil, err
	}

	return &Service{
		templates: tmpl,
	}, nil
}

func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Serve home page for root path
	if r.URL.Path == "/" {
		s.handleHome(w, r)
		return
	}

	// Return 404 for unknown paths
	http.NotFound(w, r)
}

func (s *Service) handleHome(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Welcome to tofu dns",
	}

	if err := s.templates.ExecuteTemplate(w, "home.html", data); err != nil {
		slog.Error("Failed to execute template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
