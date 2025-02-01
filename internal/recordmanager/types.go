package recordmanager

import (
	"database/sql"
	"net"
)

type Record struct {
	ID         int64
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

type AData struct {
	Ip net.IP `json:"ip"`
}

type AAAAData struct {
	Ip net.IP `json:"ip"`
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
