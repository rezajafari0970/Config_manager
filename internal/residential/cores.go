package residential

// Only checksum-verified official binaries are allowed by the supervisor and
// owner. The ordinary/Smart target remains independently pinned.
func CoreID(sha string) string {
	switch sha {
	case SingBoxSHA256:
		return SingBoxID
	case "e87bce586acdb9136aa5b1f1d9c8db9f1ea1d145b6d86368c7f6f662f9f6a66d":
		return "xray-26.11.9"
	case "64d46afb80adea1bf97a0d467e83f4a9ac1ebd0995891e84bca3f1a1d1affb1d":
		return "xray-26.7.28"
	case "b2c525e082cf2fef460499c88838d355f0b9bfb5a00bdb3eaa99b6af63825006":
		return "xray-26.5.9"
	}
	return ""
}

type CoreAttempt struct {
	CoreID           string    `json:"core_id"`
	CoreSHA256       string    `json:"core_sha256"`
	Started          int64     `json:"started"`
	Finished         int64     `json:"finished"`
	InfrastructureOK bool      `json:"infrastructure_ok"`
	Variants         []Variant `json:"variants"`
	ControlsPassed   int       `json:"controls_passed"`
	WallMS           float64   `json:"wall_ms"`
}
