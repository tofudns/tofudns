package frontend

import (
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/tofudns/tofudns/internal/recordmanager"
	"github.com/tofudns/tofudns/internal/storage"
)

//go:embed templates/*
var templateFS embed.FS

// EmailService defines the interface for sending emails
type EmailService interface {
	SendOTP(email, otp string) error
}

type Service struct {
	logger       *slog.Logger
	records      *recordmanager.RecordManager
	templates    *template.Template
	db           *storage.Queries
	emailService EmailService
	jwtSecret    string
}

func New(
	logger *slog.Logger,
	records *recordmanager.RecordManager,
	db *storage.Queries,
	emailService EmailService,
	jwtSecret string,
) (*Service, error) {
	tmpl, err := template.ParseFS(templateFS, "templates/*.html")
	if err != nil {
		return nil, err
	}

	return &Service{
		logger:       logger,
		records:      records,
		templates:    tmpl,
		db:           db,
		emailService: emailService,
		jwtSecret:    jwtSecret,
	}, nil
}

func (s *Service) Router(r chi.Router) {
	// Apply auth middleware to all routes
	r.Use(s.authMiddleware)

	// Set up auth routes
	s.setupAuthRoutes(r)

	// DNS management routes
	r.Get("/", s.handleZoneList)
	r.Post("/new/zone", s.handleNewZone)
	r.Get("/zones/{zone}", s.handleZoneDetail)
	r.Get("/zones/{zone}/records/{recordId}/delete", s.handleRecordDeleteForm)
	r.Post("/zones/{zone}/records/{recordId}/delete", s.handleRecordDelete)
	r.Post("/zones/{zone}/records/create", s.handleRecordCreate)
	r.Post("/zones/{zone}/records/{recordId}/update", s.handleRecordUpdate)
}

func (s *Service) handleZoneList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := getUserID(r)
	zones, err := s.records.ListZones(ctx, userID)
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

	userID := getUserID(r)
	record := recordmanager.Record{
		UserID:     userID,
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
	userID := getUserID(r)
	records, err := s.records.ListRecordsByZone(ctx, zone, userID)
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
	userID := getUserID(r)
	record, err := s.records.GetRecord(ctx, recordId, zone, userID)
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
	userID := getUserID(r)
	err = s.records.DeleteRecord(ctx, recordId, zone, userID)
	if err != nil {
		slog.Error("Failed to delete record", "error", err)
		http.Error(w, "Failed to delete record", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/zones/"+zone, http.StatusSeeOther)
}

// ValidationError represents a structured validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ErrorResponse represents a structured error response
type ErrorResponse struct {
	Status  string            `json:"status"`
	Message string            `json:"message"`
	Errors  []ValidationError `json:"errors,omitempty"`
}

// respondWithError writes a JSON error response
func respondWithError(w http.ResponseWriter, code int, message string, errors []ValidationError) {
	response := ErrorResponse{
		Status:  "error",
		Message: message,
		Errors:  errors,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(response)
}

// validateRecord performs validation on a record based on its type
func validateRecord(record *recordmanager.Record) []ValidationError {
	var errors []ValidationError

	// Validate name field
	if record.Name == "" {
		errors = append(errors, ValidationError{
			Field:   "name",
			Message: "Name is required",
		})
	}

	// Validate TTL
	if !record.Ttl.Valid || record.Ttl.Int32 <= 0 {
		errors = append(errors, ValidationError{
			Field:   "ttl",
			Message: "TTL must be a positive number",
		})
	}

	// Validate record-specific fields
	switch record.RecordType {
	case "A":
		if record.A == nil || record.A.Ip.IP == nil {
			errors = append(errors, ValidationError{
				Field:   "ip",
				Message: "Valid IP address is required",
			})
		} else if record.A.Ip.IP.To4() == nil {
			errors = append(errors, ValidationError{
				Field:   "ip",
				Message: "IP must be a valid IPv4 address",
			})
		}
	case "CNAME":
		if record.CNAME == nil || record.CNAME.Host == "" {
			errors = append(errors, ValidationError{
				Field:   "host",
				Message: "Target host is required",
			})
		}
	case "MX":
		if record.MX == nil {
			errors = append(errors, ValidationError{
				Field:   "host",
				Message: "Mail server is required",
			})
		} else {
			if record.MX.Host == "" {
				errors = append(errors, ValidationError{
					Field:   "host",
					Message: "Mail server is required",
				})
			}
			// Preference is validated by protobuf/json, but we should still check valid range
			if record.MX.Preference < 0 {
				errors = append(errors, ValidationError{
					Field:   "preference",
					Message: "Preference must be a non-negative number",
				})
			}
		}
	case "TXT":
		if record.TXT == nil || record.TXT.Text == "" {
			errors = append(errors, ValidationError{
				Field:   "text",
				Message: "Text value is required",
			})
		}
	default:
		errors = append(errors, ValidationError{
			Field:   "record_type",
			Message: "Unsupported record type: " + record.RecordType,
		})
	}

	return errors
}

func (s *Service) handleRecordCreate(w http.ResponseWriter, r *http.Request) {
	zone := chi.URLParam(r, "zone")
	if zone == "" {
		respondWithError(w, http.StatusBadRequest, "Zone is required", nil)
		return
	}

	var payload struct {
		Name       string          `json:"name"`
		TTL        int32           `json:"ttl"`
		RecordType string          `json:"record_type"`
		Content    json.RawMessage `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid JSON payload", nil)
		return
	}

	userID := getUserID(r)
	record := recordmanager.Record{
		UserID:     userID,
		Zone:       zone,
		Name:       strings.TrimSpace(payload.Name),
		RecordType: payload.RecordType,
		Ttl: sql.NullInt32{
			Int32: payload.TTL,
			Valid: true,
		},
	}

	// Parse content based on record type
	var contentErr error
	switch payload.RecordType {
	case "A":
		var aContent struct {
			IP string `json:"ip"`
		}
		if err := json.Unmarshal(payload.Content, &aContent); err != nil {
			contentErr = fmt.Errorf("invalid A record content: %w", err)
		} else {
			ip := net.ParseIP(strings.TrimSpace(aContent.IP))
			record.A = &recordmanager.AData{
				Ip: recordmanager.IPAddr{IP: ip},
			}
		}
	case "CNAME":
		record.CNAME = &recordmanager.CNAMEData{}
		if err := json.Unmarshal(payload.Content, record.CNAME); err != nil {
			contentErr = fmt.Errorf("invalid CNAME record content: %w", err)
		} else {
			record.CNAME.Host = strings.TrimSpace(record.CNAME.Host)
		}
	case "MX":
		record.MX = &recordmanager.MXData{}
		if err := json.Unmarshal(payload.Content, record.MX); err != nil {
			contentErr = fmt.Errorf("invalid MX record content: %w", err)
		} else {
			record.MX.Host = strings.TrimSpace(record.MX.Host)
		}
	case "TXT":
		record.TXT = &recordmanager.TXTData{}
		if err := json.Unmarshal(payload.Content, record.TXT); err != nil {
			contentErr = fmt.Errorf("invalid TXT record content: %w", err)
		} else {
			record.TXT.Text = strings.TrimSpace(record.TXT.Text)
		}
	default:
		contentErr = fmt.Errorf("unsupported record type: %s", payload.RecordType)
	}

	if contentErr != nil {
		respondWithError(w, http.StatusBadRequest, contentErr.Error(), nil)
		return
	}

	// Validate record
	if validationErrors := validateRecord(&record); len(validationErrors) > 0 {
		respondWithError(w, http.StatusBadRequest, "Validation failed", validationErrors)
		return
	}

	ctx := r.Context()
	_, err := s.records.CreateRecord(ctx, &record)
	if err != nil {
		slog.Error("Failed to create record", "error", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to create record", nil)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func (s *Service) handleRecordUpdate(w http.ResponseWriter, r *http.Request) {
	zone := chi.URLParam(r, "zone")
	if zone == "" {
		respondWithError(w, http.StatusBadRequest, "Zone is required", nil)
		return
	}

	recordIdStr := chi.URLParam(r, "recordId")
	if recordIdStr == "" {
		respondWithError(w, http.StatusBadRequest, "Record ID is required", nil)
		return
	}
	recordId, err := strconv.ParseInt(recordIdStr, 10, 64)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Record ID is not a number", nil)
		return
	}

	var payload struct {
		Name       string          `json:"name"`
		TTL        int32           `json:"ttl"`
		RecordType string          `json:"record_type"`
		Content    json.RawMessage `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid JSON payload", nil)
		return
	}

	userID := getUserID(r)
	record := recordmanager.Record{
		ID:         recordId,
		UserID:     userID,
		Zone:       zone,
		Name:       strings.TrimSpace(payload.Name),
		RecordType: payload.RecordType,
		Ttl: sql.NullInt32{
			Int32: payload.TTL,
			Valid: true,
		},
	}

	// Parse content based on record type
	var contentErr error
	switch payload.RecordType {
	case "A":
		var aContent struct {
			IP string `json:"ip"`
		}
		if err := json.Unmarshal(payload.Content, &aContent); err != nil {
			contentErr = fmt.Errorf("invalid A record content: %w", err)
		} else {
			ip := net.ParseIP(strings.TrimSpace(aContent.IP))
			record.A = &recordmanager.AData{
				Ip: recordmanager.IPAddr{IP: ip},
			}
		}
	case "CNAME":
		record.CNAME = &recordmanager.CNAMEData{}
		if err := json.Unmarshal(payload.Content, record.CNAME); err != nil {
			contentErr = fmt.Errorf("invalid CNAME record content: %w", err)
		} else {
			record.CNAME.Host = strings.TrimSpace(record.CNAME.Host)
		}
	case "MX":
		record.MX = &recordmanager.MXData{}
		if err := json.Unmarshal(payload.Content, record.MX); err != nil {
			contentErr = fmt.Errorf("invalid MX record content: %w", err)
		} else {
			record.MX.Host = strings.TrimSpace(record.MX.Host)
		}
	case "TXT":
		record.TXT = &recordmanager.TXTData{}
		if err := json.Unmarshal(payload.Content, record.TXT); err != nil {
			contentErr = fmt.Errorf("invalid TXT record content: %w", err)
		} else {
			record.TXT.Text = strings.TrimSpace(record.TXT.Text)
		}
	default:
		contentErr = fmt.Errorf("unsupported record type: %s", payload.RecordType)
	}

	if contentErr != nil {
		respondWithError(w, http.StatusBadRequest, contentErr.Error(), nil)
		return
	}

	// Validate record
	if validationErrors := validateRecord(&record); len(validationErrors) > 0 {
		respondWithError(w, http.StatusBadRequest, "Validation failed", validationErrors)
		return
	}

	ctx := r.Context()
	_, err = s.records.UpdateRecord(ctx, &record)
	if err != nil {
		slog.Error("Failed to update record", "error", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to update record", nil)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}
