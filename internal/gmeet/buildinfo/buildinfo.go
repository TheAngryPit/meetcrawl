package buildinfo

// Version is set at release; dev builds use this default.
var Version = "0.0.0-dev"

type Info struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func Current() Info {
	return Info{Name: "gmeetcrawl", Version: Version}
}
