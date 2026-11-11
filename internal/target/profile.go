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
		ID:     "v2rayng-2.3.15-lib40-xray-26.10.31-and-26.11.9",
		Client: "v2rayNG", ClientVersion: "2.3.15", LibraryVersion: "40", CoreVersion: "26.10.31",
		ClientCommit: "57907cf354d9769200ed4717b4430d605abe0952",
		CoreCommit:   "a4a3dff5b20c",
	}
}
