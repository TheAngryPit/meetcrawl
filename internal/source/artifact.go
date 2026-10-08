package source

import "fmt"

// Artifact is normalized meeting text read from one source revision.
type Artifact struct {
	Identity
	SourceRevision string
	Fidelity       Fidelity
	Privacy        PrivacyClass
	NormalizedText string
	Window         Window
	Language       string
}

func (a Artifact) Validate() error {
	if err := a.Identity.validate(); err != nil {
		return err
	}
	if err := a.Fidelity.validate(); err != nil {
		return err
	}
	if err := a.Privacy.validate(); err != nil {
		return err
	}
	if a.NormalizedText == "" {
		return fmt.Errorf("source: empty normalized text")
	}
	return nil
}
