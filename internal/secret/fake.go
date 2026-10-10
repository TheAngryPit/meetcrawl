package secret

import (
	"context"
	"fmt"
	"strings"
)

type MapProvider map[string]string

func (m MapProvider) Resolve(_ context.Context, ref string) (string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", fmt.Errorf("%w: empty reference", ErrInvalid)
	}
	v, ok := m[ref]
	if !ok || strings.TrimSpace(v) == "" {
		return "", ErrNotFound
	}
	return strings.TrimSpace(v), nil
}
