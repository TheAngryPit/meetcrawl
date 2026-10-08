package buildinfo

const Version = "0.0.0-dev"

type Info struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func Current() Info {
	return Info{Name: "meetcrawl", Version: Version}
}
