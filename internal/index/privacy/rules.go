package privacy

import (
	"fmt"
	"regexp"

	"github.com/TheAngryPit/meetcrawl/internal/index/crawler"
	"github.com/TheAngryPit/meetcrawl/internal/source"
)

// Rule assigns a privacy class from config; content is never inspected.
type Rule struct {
	TitleRegex   string `toml:"title_regex,omitempty"`
	CalendarICal string `toml:"calendar_ical,omitempty"`
	Source       string `toml:"source,omitempty"`
	Class        string `toml:"class"`
	titlePattern *regexp.Regexp
}

// Config holds privacy rules loaded from meetcrawl config.
type Config struct {
	Rules []Rule `toml:"rules"`
}

func (c Config) Compile() (Config, error) {
	out := c
	for i := range out.Rules {
		if out.Rules[i].TitleRegex == "" {
			continue
		}
		re, err := regexp.Compile(out.Rules[i].TitleRegex)
		if err != nil {
			return Config{}, err
		}
		out.Rules[i].titlePattern = re
	}
	return out, nil
}

// ClassFor returns the configured class for row, or private when no rule matches.
func (c Config) ClassFor(row crawler.Row, title string) (source.PrivacyClass, error) {
	for _, rule := range c.Rules {
		if rule.Source != "" && rule.Source != row.Source.String() {
			continue
		}
		if rule.CalendarICal != "" && rule.CalendarICal != row.CalendarICal {
			continue
		}
		if rule.titlePattern != nil && !rule.titlePattern.MatchString(title) {
			continue
		}
		class := source.PrivacyClass(rule.Class)
		switch class {
		case source.PrivacyPrivate, source.PrivacyRestricted, source.PrivacyShareable:
			return class, nil
		default:
			return "", fmt.Errorf("privacy: unknown class %q", rule.Class)
		}
	}
	return source.PrivacyPrivate, nil
}
