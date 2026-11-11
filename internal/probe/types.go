package probe

const CoreSHA256 = "e87bce586acdb9136aa5b1f1d9c8db9f1ea1d145b6d86368c7f6f662f9f6a66d"

type Settings struct {
	Enabled          bool `json:"enabled"`
	Concurrency      int  `json:"concurrency"`
	TimeoutSeconds   int  `json:"timeout_seconds"`
	RetestSeconds    int  `json:"retest_seconds"`
	HealthTTLSeconds int  `json:"health_ttl_seconds"`
}
type Job struct {
	Hash     string   `json:"hash"`
	Token    string   `json:"token"`
	Revision int64    `json:"revision"`
	Kind     string   `json:"kind"`
	Raw      []byte   `json:"raw"`
	Issue    string   `json:"issue,omitempty"`
	Settings Settings `json:"settings"`
}
type Attempt struct {
	Evidence        map[string]string `json:"evidence,omitempty"`
	SHA256          string            `json:"sha256"`
	Rule            string            `json:"rule"`
	Core            string            `json:"core"`
	Connection      string            `json:"connection"`
	Code            string            `json:"code"`
	RuntimeSHA256   string            `json:"runtime_sha256"`
	EndpointsPassed int               `json:"endpoints_passed"`
}
type Result struct {
	Hash       string `json:"hash"`
	Token      string `json:"token"`
	Revision   int64  `json:"revision"`
	Version    string `json:"version"`
	CoreSHA256 string `json:"core_sha256"`
	Attempt
	Candidate []byte    `json:"candidate"`
	Attempts  []Attempt `json:"attempts"`
}
type WorkerStatus struct {
	Status    string `json:"status"`
	Code      string `json:"code"`
	Version   string `json:"version"`
	Active    int    `json:"active"`
	Heartbeat int64  `json:"heartbeat"`
}
