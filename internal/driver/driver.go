// Package driver defines independent, versioned pipeline boundaries.
// Implementations must not conflate parsing, core validation and Android acceptance.
package driver

import (
	"context"
	"time"
)

type Artifact struct {
	Raw        []byte
	SHA256     string
	Format     string
	SourceID   string
	Generation uint64
}
type Finding struct {
	Code     string `json:"code"`
	Path     string `json:"path"`
	Severity string `json:"severity"`
}
type Patch struct {
	RuleID       string `json:"rule_id"`
	Path         string `json:"path"`
	BeforeSHA256 string `json:"before_sha256"`
	AfterSHA256  string `json:"after_sha256"`
}
type Evidence struct {
	Stage          string
	Status         string
	ArtifactSHA256 string
	TargetID       string
	DriverVersion  string
	TestedAt       time.Time
	ExpiresAt      time.Time
	Vantage        string
}
type SourceDriver interface {
	ID() string
	Fetch(context.Context, string) (Artifact, error)
}
type FormatDriver interface {
	ID() string
	Detect([]byte) bool
	Extract(context.Context, Artifact) ([]Artifact, error)
}
type ProtocolDriver interface {
	ID() string
	Diagnose(context.Context, Artifact, string) ([]Finding, error)
}
type RepairDriver interface {
	ID() string
	Propose(context.Context, Artifact, []Finding, string) ([]Artifact, []Patch, error)
}
type CoreDriver interface {
	ID() string
	Validate(context.Context, Artifact, string) (Evidence, error)
}
type ClientDriver interface {
	ID() string
	CheckImport(context.Context, Artifact, string) (Evidence, error)
}
type ProbeDriver interface {
	ID() string
	Probe(context.Context, Artifact, string) (Evidence, error)
}
