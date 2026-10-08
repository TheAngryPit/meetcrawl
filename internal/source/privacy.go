package source

import "fmt"

// PrivacyClass controls agent visibility. It is set by config rules, never inferred from content.
type PrivacyClass string

const (
	PrivacyPrivate    PrivacyClass = "private"
	PrivacyRestricted PrivacyClass = "restricted"
	PrivacyShareable  PrivacyClass = "shareable"
)

func (p PrivacyClass) String() string { return string(p) }

func (p PrivacyClass) validate() error {
	switch p {
	case PrivacyPrivate, PrivacyRestricted, PrivacyShareable:
		return nil
	default:
		return fmt.Errorf("source: unknown privacy_class %q", p)
	}
}
