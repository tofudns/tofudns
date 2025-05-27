package recordmanager

import (
	"database/sql"
	"encoding/json"
	"net"

	"github.com/google/uuid"
)

type Record struct {
	ID         int64
	UserID     uuid.UUID
	Zone       string
	Name       string
	RecordType string
	Ttl        sql.NullInt32
	Content    sql.NullString

	// Type specific data
	A     *AData
	AAAA  *AAAAData
	TXT   *TXTData
	CNAME *CNAMEData
	NS    *NSData
	MX    *MXData
	SRV   *SRVData
	SOA   *SOAData
	CAA   *CAAData
}

// IPAddr wraps net.IP to provide custom JSON marshaling
type IPAddr struct {
	net.IP
}

// MarshalJSON converts the IP to a string for JSON
func (ip IPAddr) MarshalJSON() ([]byte, error) {
	if ip.IP == nil {
		return json.Marshal("")
	}
	return json.Marshal(ip.IP.String())
}

// UnmarshalJSON converts a string to an IP for JSON
func (ip *IPAddr) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	if s == "" {
		ip.IP = nil
		return nil
	}
	ip.IP = net.ParseIP(s)
	return nil
}

type AData struct {
	Ip IPAddr `json:"ip"`
}

type AAAAData struct {
	Ip IPAddr `json:"ip"`
}

type TXTData struct {
	Text string `json:"text"`
}

type CNAMEData struct {
	Host string `json:"host"`
}

type NSData struct {
	Host string `json:"host"`
}

type MXData struct {
	Host       string `json:"host"`
	Preference uint16 `json:"preference"`
}

type SRVData struct {
	Priority uint16 `json:"priority"`
	Weight   uint16 `json:"weight"`
	Port     uint16 `json:"port"`
	Target   string `json:"target"`
}

type SOAData struct {
	Ns      string `json:"ns"`
	MBox    string `json:"mbox"`
	Refresh uint32 `json:"refresh"`
	Retry   uint32 `json:"retry"`
	Expire  uint32 `json:"expire"`
	MinTtl  uint32 `json:"minttl"`
}

type CAAData struct {
	Flag  uint8  `json:"flag"`
	Tag   string `json:"tag"`
	Value string `json:"value"`
}
