package registry

import (
	circlebackconfig "github.com/TheAngryPit/meetcrawl/internal/circleback/config"
	econfig "github.com/TheAngryPit/meetcrawl/internal/exportcrawl/config"
	fathomconfig "github.com/TheAngryPit/meetcrawl/internal/fathom/config"
	firefliesconfig "github.com/TheAngryPit/meetcrawl/internal/fireflies/config"
	gconfig "github.com/TheAngryPit/meetcrawl/internal/gmeet/config"
	grainconfig "github.com/TheAngryPit/meetcrawl/internal/grain/config"
	granolaconfig "github.com/TheAngryPit/meetcrawl/internal/granola/config"
	krispconfig "github.com/TheAngryPit/meetcrawl/internal/krisp/config"
	tldvconfig "github.com/TheAngryPit/meetcrawl/internal/tldv/config"
	wconfig "github.com/TheAngryPit/meetcrawl/internal/whisp/config"
)

// ConfigView is the slice of meet config registry entries need (avoids importing meetcrawl/config).
type ConfigView interface {
	OpenWhisprArchiveDB() string
	GMeetArchiveDB() string
	ExportFileArchiveDB() string
	GrainArchiveDB() string
	GranolaArchiveDB() string
	FirefliesArchiveDB() string
	FathomArchiveDB() string
	KrispArchiveDB() string
	CirclebackArchiveDB() string
	TldvArchiveDB() string
	WhispConfig() wconfig.Config
	GMeetConfig() gconfig.Config
	ExportFileConfig() econfig.Config
	GrainConfig() grainconfig.Config
	GranolaConfig() granolaconfig.Config
	FirefliesConfig() firefliesconfig.Config
	FathomConfig() fathomconfig.Config
	KrispConfig() krispconfig.Config
	CirclebackConfig() circlebackconfig.Config
	TldvConfig() tldvconfig.Config
}
