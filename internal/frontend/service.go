package frontend

import (
	"database/sql"
	"embed"
	"html/template"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/tofudns/tofudns/internal/recordmanager"
)

//go:embed templates/*
var templateFS embed.FS

type Service struct {
	logger    *slog.Logger
	records   *recordmanager.RecordManager
	templates *template.Template
}

func New(
	logger *slog.Logger,
	records *recordmanager.RecordManager,
) (*Service, error) {
	tmpl, err := template.ParseFS(templateFS, "templates/*.html")
	if err != nil {
		return nil, err
	}

	return &Service{
		logger:    logger,
		records:   records,
		templates: tmpl,
	}, nil
}

func (s *Service) Router(r chi.Router) {
	r.Get("/", s.handleZoneList)
	r.Post("/new/zone", s.handleNewZone)
	r.Get("/zones/{zone}", s.handleZoneDetail)
	r.Get("/zones/{zone}/records/{recordId}/delete", s.handleRecordDeleteForm)
	r.Post("/zones/{zone}/records/{recordId}/delete", s.handleRecordDelete)
}

func (s *Service) handleZoneList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	zones, err := s.records.ListZones(ctx)
	if err != nil {
		slog.Error("Failed to retrieve zones", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Zones": zones,
	}
	if err := s.templates.ExecuteTemplate(w, "zone_list.html", data); err != nil {
		slog.Error("Failed to execute template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (s *Service) handleNewZone(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		slog.Error("Failed to parse form", "error", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	zone := r.Form.Get("zone")
	if zone == "" {
		http.Error(w, "Zone is required", http.StatusBadRequest)
		return
	}

	record := recordmanager.Record{
		Zone:       zone,
		Name:       "",
		RecordType: "SOA",
		Ttl: sql.NullInt32{
			Int32: 3600,
			Valid: true,
		},
		SOA: &recordmanager.SOAData{
			Ns:      "ns1.tofudns.net",   // primary nameserver (always one)
			MBox:    "admin.tofudns.net", // admin@tofudns.net
			Refresh: 86400,               // 24 hours
			Retry:   7200,                // 2 hours
			Expire:  604800,              // 1 week
			MinTtl:  300,                 // 5 minutes
		},
	}
	ctx := r.Context()
	_, err := s.records.CreateRecord(ctx, &record)
	if err != nil {
		slog.Error("Failed to create SOA record", "error", err)
		http.Error(w, "Failed to create zone", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/zones/"+zone, http.StatusSeeOther)
}

func (s *Service) handleZoneDetail(w http.ResponseWriter, r *http.Request) {
	zone := chi.URLParam(r, "zone")
	if zone == "" {
		http.Error(w, "Zone is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	records, err := s.records.ListRecordsByZone(ctx, zone)
	if err != nil {
		slog.Error("Failed to retrieve zone records", "error", err, "zone", zone)
		http.Error(w, "Failed to retrieve zone records", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Zone":    zone,
		"Records": records,
	}

	if err := s.templates.ExecuteTemplate(w, "zone_detail.html", data); err != nil {
		slog.Error("Failed to execute template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (s *Service) handleRecordDeleteForm(w http.ResponseWriter, r *http.Request) {
	zone := chi.URLParam(r, "zone")
	if zone == "" {
		http.Error(w, "Zone is required", http.StatusBadRequest)
		return
	}

	recordIdStr := chi.URLParam(r, "recordId")
	if recordIdStr == "" {
		http.Error(w, "Record ID is required", http.StatusBadRequest)
		return
	}
	recordId, err := strconv.ParseInt(recordIdStr, 10, 64)
	if err != nil {
		http.Error(w, "Record ID is not a number", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	record, err := s.records.GetRecord(ctx, recordId, zone)
	if err != nil {
		slog.Error("Failed to retrieve record", "error", err, "zone", zone)
		http.Error(w, "Failed to retrieve record", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Zone":   zone,
		"Record": record,
	}
	if err := s.templates.ExecuteTemplate(w, "record_delete.html", data); err != nil {
		slog.Error("Failed to execute template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (s *Service) handleRecordDelete(w http.ResponseWriter, r *http.Request) {
	zone := chi.URLParam(r, "zone")
	if zone == "" {
		http.Error(w, "Zone is required", http.StatusBadRequest)
		return
	}

	recordIdStr := chi.URLParam(r, "recordId")
	if recordIdStr == "" {
		http.Error(w, "Record ID is required", http.StatusBadRequest)
		return
	}
	recordId, err := strconv.ParseInt(recordIdStr, 10, 64)
	if err != nil {
		http.Error(w, "Record ID is not a number", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	s.records.DeleteRecord(ctx, recordId, zone)
	http.Redirect(w, r, "/zones/"+zone, http.StatusSeeOther)
}
