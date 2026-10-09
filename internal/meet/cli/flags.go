package cli

import "strings"

type GlobalFlags struct {
	JSON       bool
	ConfigPath string
	Help       bool
	Version    bool
}

func parseGlobalFlags(args []string) (GlobalFlags, []string) {
	var flags GlobalFlags
	var rest []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--json":
			flags.JSON = true
		case arg == "--help" || arg == "-h":
			flags.Help = true
		case arg == "--version":
			flags.Version = true
		case strings.HasPrefix(arg, "--config="):
			flags.ConfigPath = strings.TrimPrefix(arg, "--config=")
		case arg == "--config":
			if i+1 < len(args) {
				i++
				flags.ConfigPath = args[i]
			}
		default:
			rest = append(rest, arg)
		}
	}
	return flags, rest
}

func flagValue(args []string, name string) (string, bool) {
	for i := 0; i < len(args); i++ {
		if args[i] == name && i+1 < len(args) {
			return args[i+1], true
		}
		if strings.HasPrefix(args[i], name+"=") {
			return strings.TrimPrefix(args[i], name+"="), true
		}
	}
	return "", false
}

func hasFlag(args []string, name string) bool {
	for _, arg := range args {
		if arg == name {
			return true
		}
	}
	return false
}
