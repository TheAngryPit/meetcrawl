package secret

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

type OpRunner func(ctx context.Context, command string, ref string) (string, error)

type OpProvider struct {
	Command string
	Runner  OpRunner
}

func (p OpProvider) runner() OpRunner {
	if p.Runner != nil {
		return p.Runner
	}
	return execOpRead
}

func (p OpProvider) command() string {
	if strings.TrimSpace(p.Command) != "" {
		return p.Command
	}
	return "op"
}

func (p OpProvider) Resolve(ctx context.Context, ref string) (string, error) {
	ref = strings.TrimSpace(ref)
	if !strings.HasPrefix(ref, "op://") {
		return "", fmt.Errorf("%w: op provider requires op:// reference", ErrInvalid)
	}
	value, err := p.runner()(ctx, p.command(), ref)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return "", ErrNotFound
		}
		return "", err
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", ErrNotFound
	}
	return value, nil
}

func execOpRead(ctx context.Context, command, ref string) (string, error) {
	cmd := exec.CommandContext(ctx, command, "read", ref, "--no-newline")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		lower := strings.ToLower(msg)
		if strings.Contains(lower, "not found") || strings.Contains(lower, "doesn't exist") {
			return "", fmt.Errorf("%w (%s)", ErrNotFound, sanitizeOpError(msg))
		}
		return "", fmt.Errorf("secret op: %s", sanitizeOpError(msg))
	}
	return string(out), nil
}

func sanitizeOpError(msg string) string {
	msg = strings.TrimSpace(msg)
	if len(msg) > 200 {
		return msg[:200] + "…"
	}
	return msg
}
