package registry

import (
	econfig "github.com/TheAngryPit/meetcrawl/internal/exportcrawl/config"
	gconfig "github.com/TheAngryPit/meetcrawl/internal/gmeet/config"
	grainconfig "github.com/TheAngryPit/meetcrawl/internal/grain/config"
	wconfig "github.com/TheAngryPit/meetcrawl/internal/whisp/config"
)

// ConfigView is the slice of meet config registry entries need (avoids importing meetcrawl/config).
type ConfigView interface {
	OpenWhisprArchiveDB() string
	GMeetArchiveDB() string
	ExportFileArchiveDB() string
	GrainArchiveDB() string
	WhispConfig() wconfig.Config
	GMeetConfig() gconfig.Config
	ExportFileConfig() econfig.Config
	GrainConfig() grainconfig.Config
}
