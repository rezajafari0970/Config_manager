package collect

import (
	"context"
	"io"
	"time"

	"github.com/rezajafari0970/Config_manager/internal/blob"
	"github.com/rezajafari0970/Config_manager/internal/repair"
)

type Settings struct {
	RetentionSeconds int64 `json:"retention_seconds"`
	Concurrency      int   `json:"concurrency"`
	TimeoutSeconds   int64 `json:"timeout_seconds"`
}
type Job struct {
	Revalidate                     bool
	ID, URL, Token, ETag, Modified string
	Revision                       int64
	Min, Max                       time.Duration
	TimeoutSeconds                 int64
}
type Item struct {
	blob.Ref
	Identity  string            `json:"identity"`
	Kind      string            `json:"kind"`
	Quality   string            `json:"quality"`
	Repair    repair.Diagnostic `json:"repair"`
	Candidate blob.Ref          `json:"candidate"`
}
type Result struct {
	State, ErrorCode, ETag, Modified       string
	HTTPStatus                             int
	DurationMS                             int64
	Raw                                    blob.Ref
	Items                                  []Item
	Extracted                              int
	Staged                                 bool
	ContentState                           string
	ValidCount, InvalidCount, UnknownCount int
}
type Repository interface {
	Abandon(context.Context, Job) error
	Settings(context.Context) (Settings, error)
	Claim(context.Context, time.Time, int, time.Duration) ([]Job, error)
	Stage(context.Context, Job, Item) (bool, error)
	Complete(context.Context, Job, Result, time.Time) error
	Expire(context.Context, time.Time) error
	Referenced(context.Context, string) (bool, error)
	Recover(context.Context) error
}

// BatchRepository is an optional storage driver capability. Publication is still
// revision/token fenced and atomic in Complete, never in StageBatch.
type BatchRepository interface {
	StageBatch(context.Context, Job, []Item) (int, error)
}

const ProcessingTimeout = 2 * time.Minute
const CompletionTimeout = 30 * time.Second

func LeaseDuration(fetchTimeout time.Duration) time.Duration {
	return fetchTimeout + ProcessingTimeout + CompletionTimeout + 30*time.Second
}

type Response struct {
	Body           io.ReadCloser
	Status         int
	ETag, Modified string
	Chunked        bool
	RecoveryRule   string
}
type FetchDriver interface {
	Name() string
	Fetch(context.Context, Job) (Response, error)
}

// BodyRecoveryDriver can retry a failed representation once, inside the original
// fetch deadline. The engine discards all partial bytes before retrying.
type BodyRecoveryDriver interface {
	RecoverBody(context.Context, Job, Response, error) (Response, error)
}

type Emitter func(context.Context, io.Reader, string) error
type FormatDriver interface {
	Name() string
	Match([]byte) bool
	Extract(context.Context, string, Emitter) error
}

// Inspection drivers annotate content without transforming or executing it.
type InspectionDriver interface {
	Kind() string
	Inspect([]byte, bool) string
}
