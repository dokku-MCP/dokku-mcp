package services

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"

	dokkuApi "github.com/dokku-mcp/dokku-mcp/internal/dokku-api"
)

// SupportedTypes lists the official Dokku datastore plugins. They all share
// the same command interface (<type>:create, <type>:link, ...).
var SupportedTypes = []string{
	"postgres", "mysql", "mariadb", "redis", "mongo",
	"rabbitmq", "memcached", "clickhouse", "elasticsearch",
}

var (
	serviceNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,62}$`)
	aliasPattern       = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)
	imageVersionRegex  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)
	// credentialsInURL matches the password part of scheme://user:password@host.
	credentialsInURL = regexp.MustCompile(`(://[^:/@\s]*:)[^@\s]+@`)
)

// ValidateServiceType checks that a type is one of the supported datastores.
func ValidateServiceType(serviceType string) error {
	if !slices.Contains(SupportedTypes, serviceType) {
		return fmt.Errorf("unsupported service type %q (supported: %s)", serviceType, strings.Join(SupportedTypes, ", "))
	}
	return nil
}

// ValidateServiceName checks a datastore service name.
func ValidateServiceName(name string) error {
	if !serviceNamePattern.MatchString(name) {
		return fmt.Errorf("invalid service name %q: use lowercase letters, digits, '-' and '_' (max 63 characters)", name)
	}
	return nil
}

// RedactCredentials masks passwords embedded in connection URLs.
func RedactCredentials(value string) string {
	return credentialsInURL.ReplaceAllString(value, "${1}***@")
}

// PluginLister reports which Dokku plugins are enabled.
type PluginLister interface {
	GetEnabledDokkuPlugins(ctx context.Context) ([]string, error)
}

// Adapter runs datastore commands through the Dokku client.
type Adapter struct {
	client  dokkuApi.DokkuClient
	plugins PluginLister
}

// NewAdapter creates a datastore adapter.
func NewAdapter(client dokkuApi.DokkuClient, plugins PluginLister) *Adapter {
	return &Adapter{client: client, plugins: plugins}
}

// InstalledTypes returns the supported datastore types whose Dokku plugin
// is enabled on the server.
func (a *Adapter) InstalledTypes(ctx context.Context) ([]string, error) {
	enabled, err := a.plugins.GetEnabledDokkuPlugins(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list Dokku plugins: %w", err)
	}
	var installed []string
	for _, t := range SupportedTypes {
		if slices.Contains(enabled, t) {
			installed = append(installed, t)
		}
	}
	return installed, nil
}

// requireInstalled returns an actionable error when a datastore plugin is
// missing.
func (a *Adapter) requireInstalled(ctx context.Context, serviceType string) error {
	if err := ValidateServiceType(serviceType); err != nil {
		return err
	}
	installed, err := a.InstalledTypes(ctx)
	if err != nil {
		return err
	}
	if !slices.Contains(installed, serviceType) {
		return fmt.Errorf("the Dokku %s plugin is not installed; an administrator can install it with "+
			"`sudo dokku plugin:install https://github.com/dokku/dokku-%s.git %s`", serviceType, serviceType, serviceType)
	}
	return nil
}

func (a *Adapter) run(ctx context.Context, serviceType, subcommand string, args ...string) (string, error) {
	if err := a.requireInstalled(ctx, serviceType); err != nil {
		return "", err
	}
	out, err := a.client.ExecuteCommand(ctx, serviceType+":"+subcommand, args)
	if err != nil {
		return "", fmt.Errorf("%s:%s failed: %w", serviceType, subcommand, err)
	}
	return string(out), nil
}

// List returns the names of the services of one type.
func (a *Adapter) List(ctx context.Context, serviceType string) ([]string, error) {
	out, err := a.run(ctx, serviceType, "list")
	if err != nil {
		return nil, err
	}
	return parseServiceList(out), nil
}

// parseServiceList extracts service names from <type>:list output, which is
// either a bare list under a "=====>" header or, on older plugin versions, a
// table whose first column is NAME.
func parseServiceList(out string) []string {
	names := []string{}
	for line := range strings.Lines(out) {
		fields := strings.Fields(line)
		if len(fields) == 0 || strings.HasPrefix(fields[0], "=") || strings.HasPrefix(fields[0], "!") || fields[0] == "NAME" {
			continue
		}
		if !serviceNamePattern.MatchString(fields[0]) {
			// Messages such as "There are no Postgres services".
			continue
		}
		names = append(names, fields[0])
	}
	return names
}

// CreateOptions configures a new service.
type CreateOptions struct {
	ImageVersion string
}

// Create creates a service.
func (a *Adapter) Create(ctx context.Context, serviceType, name string, opts CreateOptions) (string, error) {
	if err := ValidateServiceName(name); err != nil {
		return "", err
	}
	args := []string{name}
	if opts.ImageVersion != "" {
		if !imageVersionRegex.MatchString(opts.ImageVersion) {
			return "", fmt.Errorf("invalid image version %q", opts.ImageVersion)
		}
		args = append(args, "--image-version", opts.ImageVersion)
	}
	return a.run(ctx, serviceType, "create", args...)
}

// Info returns the service report with credentials redacted.
func (a *Adapter) Info(ctx context.Context, serviceType, name string) (map[string]string, error) {
	if err := ValidateServiceName(name); err != nil {
		return nil, err
	}
	out, err := a.run(ctx, serviceType, "info", name)
	if err != nil {
		return nil, err
	}
	info := make(map[string]string)
	for line := range strings.Lines(out) {
		key, value, ok := dokkuApi.ParseColonKeyValueLine(line)
		if !ok || strings.HasPrefix(strings.TrimSpace(line), "=") {
			continue
		}
		if strings.Contains(strings.ToLower(key), "password") {
			value = "***"
		}
		info[key] = RedactCredentials(value)
	}
	return info, nil
}

// LinkOptions configures how a service is exposed to an app.
type LinkOptions struct {
	Alias     string
	NoRestart bool
}

// Link links a service to an application, which sets <ALIAS>_URL (by
// default DATABASE_URL, REDIS_URL, ...) on the app.
func (a *Adapter) Link(ctx context.Context, serviceType, name, appName string, opts LinkOptions) (string, error) {
	if err := ValidateServiceName(name); err != nil {
		return "", err
	}
	args := []string{name, appName}
	if opts.Alias != "" {
		if !aliasPattern.MatchString(opts.Alias) {
			return "", fmt.Errorf("invalid alias %q: use uppercase letters, digits and '_'", opts.Alias)
		}
		args = append(args, "--alias", opts.Alias)
	}
	if opts.NoRestart {
		args = append(args, "--no-restart")
	}
	return a.run(ctx, serviceType, "link", args...)
}

// Unlink removes the link between a service and an application.
func (a *Adapter) Unlink(ctx context.Context, serviceType, name, appName string, noRestart bool) (string, error) {
	if err := ValidateServiceName(name); err != nil {
		return "", err
	}
	args := []string{name, appName}
	if noRestart {
		args = append(args, "--no-restart")
	}
	return a.run(ctx, serviceType, "unlink", args...)
}

// Links returns the applications linked to a service.
func (a *Adapter) Links(ctx context.Context, serviceType, name string) ([]string, error) {
	if err := ValidateServiceName(name); err != nil {
		return nil, err
	}
	out, err := a.run(ctx, serviceType, "links", name)
	if err != nil {
		return nil, err
	}
	return dokkuApi.ParseLinesSkipHeaders(out), nil
}

// Logs returns the most recent log lines of a service container.
func (a *Adapter) Logs(ctx context.Context, serviceType, name string) (string, error) {
	if err := ValidateServiceName(name); err != nil {
		return "", err
	}
	// Without --tail the plugins print the last 100 lines; --tail follows.
	return a.run(ctx, serviceType, "logs", name)
}

// Destroy permanently deletes a service and its data.
func (a *Adapter) Destroy(ctx context.Context, serviceType, name string) (string, error) {
	if err := ValidateServiceName(name); err != nil {
		return "", err
	}
	// --force skips the interactive confirmation, which cannot be answered
	// over a non-interactive SSH session.
	return a.run(ctx, serviceType, "destroy", name, "--force")
}
