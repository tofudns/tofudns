package recordmanager

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/tofudns/tofudns/internal/storage"
)

// RecordManager handles CRUD operations for DNS records
type RecordManager struct {
	querier storage.Querier
}

// New creates a new RecordManager instance
func New(querier storage.Querier) *RecordManager {
	return &RecordManager{
		querier: querier,
	}
}

// CreateRecord creates a new DNS record
func (m *RecordManager) CreateRecord(ctx context.Context, record *Record) (*Record, error) {
	// Marshal the content based on record type
	var content interface{}
	switch record.RecordType {
	case "A":
		content = record.A
	case "AAAA":
		content = record.AAAA
	case "TXT":
		content = record.TXT
	case "CNAME":
		content = record.CNAME
	case "NS":
		content = record.NS
	case "MX":
		content = record.MX
	case "SRV":
		content = record.SRV
	case "SOA":
		content = record.SOA
	case "CAA":
		content = record.CAA
	default:
		return nil, fmt.Errorf("unknown record type: %s", record.RecordType)
	}

	contentJSON, err := json.Marshal(content)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal content: %w", err)
	}

	// Create the record
	dbRecord, err := m.querier.CreateRecord(ctx, storage.CreateRecordParams{
		Zone:       record.Zone,
		Name:       record.Name,
		Ttl:        record.Ttl,
		Content:    sql.NullString{String: string(contentJSON), Valid: true},
		RecordType: record.RecordType,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create record: %w", err)
	}

	return m.storageToRecord(&dbRecord)
}

// GetRecord retrieves a DNS record by ID and zone
func (m *RecordManager) GetRecord(ctx context.Context, id int64, zone string) (*Record, error) {
	record, err := m.querier.GetRecordByID(ctx, storage.GetRecordByIDParams{
		ID:   id,
		Zone: zone,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get record: %w", err)
	}

	return m.storageToRecord(&record)
}

// UpdateRecord updates a DNS record
func (m *RecordManager) UpdateRecord(ctx context.Context, record *Record) (*Record, error) {
	// Marshal the content based on record type
	var content interface{}
	switch record.RecordType {
	case "A":
		content = record.A
	case "AAAA":
		content = record.AAAA
	case "TXT":
		content = record.TXT
	case "CNAME":
		content = record.CNAME
	case "NS":
		content = record.NS
	case "MX":
		content = record.MX
	case "SRV":
		content = record.SRV
	case "SOA":
		content = record.SOA
	case "CAA":
		content = record.CAA
	default:
		return nil, fmt.Errorf("unknown record type: %s", record.RecordType)
	}

	contentJSON, err := json.Marshal(content)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal content: %w", err)
	}

	// Update the record
	dbRecord, err := m.querier.UpdateRecord(ctx, storage.UpdateRecordParams{
		ID:         record.ID,
		Zone:       record.Zone,
		Name:       record.Name,
		Ttl:        record.Ttl,
		Content:    sql.NullString{String: string(contentJSON), Valid: true},
		RecordType: record.RecordType,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update record: %w", err)
	}

	return m.storageToRecord(&dbRecord)
}

// DeleteRecord deletes a DNS record
func (m *RecordManager) DeleteRecord(ctx context.Context, id int64, zone string) error {
	err := m.querier.DeleteRecord(ctx, storage.DeleteRecordParams{
		ID:   id,
		Zone: zone,
	})
	if err != nil {
		return fmt.Errorf("failed to delete record: %w", err)
	}

	return nil
}

// ListRecordsByZone lists all records in a zone
func (m *RecordManager) ListRecordsByZone(ctx context.Context, zone string) ([]*Record, error) {
	records, err := m.querier.ListRecordsByZone(ctx, zone)
	if err != nil {
		return nil, fmt.Errorf("failed to list records: %w", err)
	}

	result := make([]*Record, len(records))
	for i, record := range records {
		result[i], err = m.storageToRecord(&record)
		if err != nil {
			return nil, fmt.Errorf("failed to convert record %d: %w", record.ID, err)
		}
	}

	return result, nil
}

// ListZones lists all zones
func (m *RecordManager) ListZones(ctx context.Context) ([]string, error) {
	zones, err := m.querier.ListZones(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list zones: %w", err)
	}

	return zones, nil
}

// storageToRecord converts a storage.CorednsRecord to a Record
func (m *RecordManager) storageToRecord(dbRecord *storage.CorednsRecord) (*Record, error) {
	record := &Record{
		ID:         dbRecord.ID,
		Zone:       dbRecord.Zone,
		Name:       dbRecord.Name,
		RecordType: dbRecord.RecordType,
		Ttl:        dbRecord.Ttl,
		Content:    dbRecord.Content,
	}

	if !dbRecord.Content.Valid {
		return record, nil
	}

	// Unmarshal the content based on record type
	switch dbRecord.RecordType {
	case "A":
		record.A = &AData{}
		err := json.Unmarshal([]byte(dbRecord.Content.String), record.A)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal A record: %w", err)
		}
	case "AAAA":
		record.AAAA = &AAAAData{}
		err := json.Unmarshal([]byte(dbRecord.Content.String), record.AAAA)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal AAAA record: %w", err)
		}
	case "TXT":
		record.TXT = &TXTData{}
		err := json.Unmarshal([]byte(dbRecord.Content.String), record.TXT)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal TXT record: %w", err)
		}
	case "CNAME":
		record.CNAME = &CNAMEData{}
		err := json.Unmarshal([]byte(dbRecord.Content.String), record.CNAME)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal CNAME record: %w", err)
		}
	case "NS":
		record.NS = &NSData{}
		err := json.Unmarshal([]byte(dbRecord.Content.String), record.NS)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal NS record: %w", err)
		}
	case "MX":
		record.MX = &MXData{}
		err := json.Unmarshal([]byte(dbRecord.Content.String), record.MX)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal MX record: %w", err)
		}
	case "SRV":
		record.SRV = &SRVData{}
		err := json.Unmarshal([]byte(dbRecord.Content.String), record.SRV)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal SRV record: %w", err)
		}
	case "SOA":
		record.SOA = &SOAData{}
		err := json.Unmarshal([]byte(dbRecord.Content.String), record.SOA)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal SOA record: %w", err)
		}
	case "CAA":
		record.CAA = &CAAData{}
		err := json.Unmarshal([]byte(dbRecord.Content.String), record.CAA)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal CAA record: %w", err)
		}
	default:
		return nil, fmt.Errorf("unknown record type: %s", dbRecord.RecordType)
	}

	return record, nil
}
