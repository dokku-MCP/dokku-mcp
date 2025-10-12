package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	cfg "github.com/dokku-mcp/dokku-mcp/pkg/config"
	"github.com/spf13/viper"
)

type mcpJSON struct {
	MCpServers map[string]serverDef `json:"mcpServers"`
}

type serverDef struct {
	Command string     `json:"command,omitempty"`
	Args    []string   `json:"args,omitempty"`
	Env     orderedEnv `json:"env,omitempty"`
}

func findModuleRoot(start string) (string, error) {
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("go.mod not found in any parent directory")
		}
		dir = parent
	}
}

func main() {
	// Load server config using existing loader
	config, err := cfg.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Build env map from Viper settings
	_ = config // ensure config is used to initialize viper
	settings := viper.AllSettings()
	flat := make(map[string]any)
	flattenMap("", settings, flat)
	env := make(map[string]string, len(flat))
	for k, v := range flat {
		env[toEnvKey(k)] = anyToString(v)
	}

	// Allow override for the binary path via env; default to build output
	command := os.Getenv("DOKKU_MCP_GEN_COMMAND")
	if command == "" {
		buildDir := os.Getenv("BUILD_DIR")
		binName := os.Getenv("BINARY_NAME")
		if buildDir != "" && binName != "" {
			command = filepath.ToSlash(filepath.Join(buildDir, binName))
		} else {
			command = filepath.ToSlash(filepath.Join("./build", "dokku-mcp"))
		}
	}

	m := mcpJSON{
		MCpServers: map[string]serverDef{
			"dokku": {
				Command: command,
				Args:    []string{},
				Env:     orderedEnv(env),
			},
		},
	}

	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to marshal .mcp.json: %v\n", err)
		os.Exit(1)
	}

	wd, _ := os.Getwd()
	root, err := findModuleRoot(wd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to locate module root: %v\n", err)
		os.Exit(1)
	}
	outPath := filepath.Join(root, ".mcp.json")
	if err := os.WriteFile(outPath, append(data, '\n'), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write .mcp.json: %v\n", err)
		os.Exit(1)
	}
}

func toEnvKey(dotKey string) string {
	return "DOKKU_MCP_" + strings.ToUpper(strings.ReplaceAll(dotKey, ".", "_"))
}

// orderedEnv marshals map[string]string with deterministic key order (alphabetical).
type orderedEnv map[string]string

func (o orderedEnv) MarshalJSON() ([]byte, error) {
	keys := make([]string, 0, len(o))
	for k := range o {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var buf bytes.Buffer
	buf.WriteByte('{')
	first := true
	for _, k := range keys {
		if !first {
			buf.WriteByte(',')
		}
		first = false
		kb, err := json.Marshal(k)
		if err != nil {
			return nil, err
		}
		vb, err := json.Marshal(o[k])
		if err != nil {
			return nil, err
		}
		buf.Write(kb)
		buf.WriteByte(':')
		buf.Write(vb)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// flattenMap flattens nested maps into dot-separated keys
func flattenMap(prefix string, in map[string]any, out map[string]any) {
	for k, v := range in {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		switch t := v.(type) {
		case map[string]any:
			flattenMap(key, t, out)
		case map[any]any:
			m := make(map[string]any)
			for kk, vv := range t {
				m[fmt.Sprint(kk)] = vv
			}
			flattenMap(key, m, out)
		default:
			out[key] = v
		}
	}
}

func anyToString(v any) string {
	switch vv := v.(type) {
	case []string:
		return strings.Join(vv, ",")
	case []any:
		parts := make([]string, 0, len(vv))
		for _, e := range vv {
			parts = append(parts, fmt.Sprint(e))
		}
		return strings.Join(parts, ",")
	case bool:
		return strconv.FormatBool(vv)
	case int:
		return strconv.Itoa(vv)
	case int64:
		return strconv.FormatInt(vv, 10)
	case float64:
		return strconv.FormatFloat(vv, 'f', -1, 64)
	case string:
		return vv
	default:
		return fmt.Sprint(v)
	}
}
