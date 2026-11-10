// Package residential separates measured connectivity, attributed egress and
// access-network evidence. No geography/ASN heuristic can grant publication.
package residential

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"
)

const Version = "shen.residential.v1"
const DomainsVersion = "google-ads-explicit.2026-11-10.1"

type Policy struct {
	Version              string   `json:"version"`
	DomainsVersion       string   `json:"domains_version"`
	Domains              []string `json:"domains"`
	Families             []int    `json:"families"`
	Transports           []string `json:"transports"`
	Repeats              int      `json:"repeats"`
	EvidenceTTLSeconds   int      `json:"evidence_ttl_seconds"`
	RetestSeconds        int      `json:"retest_seconds"`
	IntelTTLSeconds      int      `json:"intel_ttl_seconds"`
	ConfirmationSeconds  int      `json:"confirmation_seconds"`
	ConfirmationAttempts int      `json:"confirmation_attempts"`
	Concurrency          int      `json:"concurrency"`
	TimeoutSeconds       int      `json:"timeout_seconds"`
}

func DefaultPolicy() Policy {
	return Policy{Version: Version, DomainsVersion: DomainsVersion,
		Domains:  []string{"pagead2.googlesyndication.com", "googleads.g.doubleclick.net", "pubads.g.doubleclick.net", "securepubads.g.doubleclick.net", "www.googleadservices.com", "adservice.google.com", "app-measurement.com", "www.googletagservices.com", "www.googletagmanager.com", "admob.com"},
		Families: []int{4, 6}, Transports: []string{"tcp", "udp", "quic"}, Repeats: 3, EvidenceTTLSeconds: 600, RetestSeconds: 300, IntelTTLSeconds: 600, ConfirmationSeconds: 30, ConfirmationAttempts: 3, Concurrency: 2, TimeoutSeconds: 6}
}
func Digest(b []byte) string    { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func (p Policy) Digest() string { b, _ := json.Marshal(p); return Digest(b) }

type Scope struct {
	Variant   string `json:"variant"`
	Domain    string `json:"domain"`
	Family    int    `json:"family"`
	Transport string `json:"transport"`
	Path      string `json:"path"`
}

func (s Scope) Key() string { b, _ := json.Marshal(s); return string(b) }

// Evidence is bounded structured data, never credentials, a raw config, an ad
// impression, request headers, unredacted logs or an arbitrary response body.
type Observation struct {
	Scope
	Stage                    string  `json:"stage,omitempty"`
	HTTPStatus               int     `json:"http_status,omitempty"`
	DiagnosticStatus         int     `json:"diagnostic_status,omitempty"`
	DiagnosticOriginVerified bool    `json:"diagnostic_origin_verified,omitempty"`
	TLSCertificateSHA256     string  `json:"tls_certificate_sha256,omitempty"`
	Attempts                 int     `json:"attempts,omitempty"`
	At                       int64   `json:"at"`
	Wave                     int     `json:"wave"`
	ConnectionID             string  `json:"connection_id"`
	Nonce                    string  `json:"nonce"`
	IP                       string  `json:"ip"`
	Method                   string  `json:"method"`
	Capability               string  `json:"capability"`
	ConfigHash               string  `json:"config_hash"`
	RuntimeHash              string  `json:"runtime_hash"`
	PolicyHash               string  `json:"policy_hash"`
	Attestation              string  `json:"attestation"`
	TLSVerified              bool    `json:"tls_verified"`
	OriginVerified           bool    `json:"origin_verified"`
	MeaningVerified          bool    `json:"meaning_verified"`
	NonceBound               bool    `json:"nonce_bound"`
	SameConnection           bool    `json:"same_connection"`
	Connections              int     `json:"connections"`
	Redirects                int     `json:"redirects"`
	ActualDomainRequest      bool    `json:"actual_domain_request"`
	ResponseSHA256           string  `json:"response_sha256"`
	Outcome                  string  `json:"outcome"`
	Code                     string  `json:"code"`
	ElapsedMS                float64 `json:"elapsed_ms"`
}
type IPFact struct {
	MAC            string `json:"mac"`
	IP             string `json:"ip"`
	Source         string `json:"source"`
	At             int64  `json:"at"`
	Expires        int64  `json:"expires"`
	Class          string `json:"class"`
	Country        string `json:"country"`
	ASN            int64  `json:"asn"`
	EvidenceSHA256 string `json:"evidence_sha256"`
}
type Variant struct {
	Name          string   `json:"name"`
	ImportRule    string   `json:"import_rule,omitempty"`
	Hash          string   `json:"hash"`
	RuntimeHash   string   `json:"runtime_hash"`
	Core          string   `json:"core"`
	Health        string   `json:"health"`
	Paths         []string `json:"paths"`
	PathsComplete bool     `json:"paths_complete"`
	Code          string   `json:"code"`
}
type Report struct {
	CoreID                 string        `json:"core_id,omitempty"`
	Compatibility          []CoreAttempt `json:"compatibility,omitempty"`
	Version                string        `json:"version"`
	PolicyHash             string        `json:"policy_hash"`
	TemplateVersion        string        `json:"template_version"`
	CoreSHA256             string        `json:"core_sha256"`
	Worker                 string        `json:"worker"`
	Started                int64         `json:"started"`
	Finished               int64         `json:"finished"`
	InfrastructureOK       bool          `json:"infrastructure_ok"`
	InfrastructureEpoch    string        `json:"infrastructure_epoch"`
	InfrastructureControls []Observation `json:"infrastructure_controls"`
	Variants               []Variant     `json:"variants"`
	Observations           []Observation `json:"observations"`
	Controls               []Observation `json:"controls"`
	IPFacts                []IPFact      `json:"ip_facts"`
	WallMS                 float64       `json:"wall_ms"`
	CPUMS                  float64       `json:"cpu_ms"`
	PeakRSSBytes           int64         `json:"peak_rss_bytes"`
}
type Decision struct {
	CoreID           string            `json:"core_id,omitempty"`
	Compatibility    []CoreAttempt     `json:"compatibility,omitempty"`
	Diagnostics      []RouteDiagnostic `json:"diagnostics,omitempty"`
	State            string            `json:"state"`
	Health           string            `json:"health"`
	Egress           string            `json:"egress"`
	Network          string            `json:"network"`
	Reason           string            `json:"reason"`
	Eligible         bool              `json:"eligible"`
	CredibleNegative bool              `json:"credible_negative"`
	ValidUntil       int64             `json:"valid_until"`
	Covered          int               `json:"covered"`
	Required         int               `json:"required"`
	Missing          []Scope           `json:"missing"`
	RotationObserved bool              `json:"rotation_observed"`
}

// Trust is installed by an administrator/compiled plugin, never copied from a
// probe result. Independence groups prevent two resellers counting twice.
type Provider struct {
	Group      string `json:"group"`
	Specialist bool   `json:"specialist"`
}
type Capability struct {
	Method         string   `json:"method"`
	PublicKey      string   `json:"public_key"`
	Domains        []string `json:"domains"`
	Transports     []string `json:"transports"`
	Expires        int64    `json:"expires"`
	EvidenceSHA256 string   `json:"evidence_sha256"`
}
type Trust struct {
	Version      string                `json:"version"`
	Providers    map[string]Provider   `json:"providers"`
	Capabilities map[string]Capability `json:"capabilities"`
}

func (t Trust) Digest() string { b, _ := json.Marshal(t); return Digest(b) }
func EmptyTrust() Trust {
	return Trust{Version: Version, Providers: map[string]Provider{}, Capabilities: map[string]Capability{}}
}

type Job struct {
	Hash      string `json:"hash"`
	Token     string `json:"token"`
	Revision  int64  `json:"revision"`
	Epoch     int64  `json:"epoch"`
	Policy    Policy `json:"policy"`
	TrustHash string `json:"trust_hash"`
	Raw       []byte `json:"raw"`
	Kind      string `json:"kind"`
	Issue     string `json:"issue"`
	Deep      bool   `json:"deep"`
}
type Completion struct {
	Hash      string `json:"hash"`
	Token     string `json:"token"`
	Revision  int64  `json:"revision"`
	Epoch     int64  `json:"epoch"`
	TrustHash string `json:"trust_hash"`
	Candidate []byte `json:"candidate"`
	Report    Report `json:"report"`
}

func Now() int64 { return time.Now().UnixMilli() }
