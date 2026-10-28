package target

// Profile is the compatibility identity, not a claim of successful Android testing.
type Profile struct {
	ID             string `json:"id"`
	Client         string `json:"client"`
	ClientVersion  string `json:"client_version"`
	LibraryVersion string `json:"library_version"`
	CoreVersion    string `json:"core_version"`
	ClientCommit   string `json:"client_commit"`
	CoreCommit     string `json:"core_commit"`
}

func Pinned() Profile {
	return Profile{
		ID:     "v2rayng-2.3.6-lib40-xray-26.7.28",
		Client: "v2rayNG", ClientVersion: "2.3.6", LibraryVersion: "40", CoreVersion: "26.7.28",
		ClientCommit: "13138dddcab291ba3618cc54721ce4c7dd926be0",
		CoreCommit:   "5ca6f4b7d4dc20a881d4330e498892697627ec0c",
	}
}
