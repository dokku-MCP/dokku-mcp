package app

import (
	"encoding/base64"
	"errors"
	"io"
	"log/slog"
	"slices"
	"strings"
	"testing"
	"time"

	dokkuApi "github.com/dokku-mcp/dokku-mcp/internal/dokku-api"
	"github.com/dokku-mcp/dokku-mcp/internal/dokku-api/dokkutest"
	"github.com/dokku-mcp/dokku-mcp/internal/server-plugin/domain"
	"github.com/dokku-mcp/dokku-mcp/internal/server-plugin/plugintest"
	"github.com/dokku-mcp/dokku-mcp/internal/server-plugins/app/infrastructure"
	"github.com/dokku-mcp/dokku-mcp/internal/server-plugins/deployment"
	deploymentAdapter "github.com/dokku-mcp/dokku-mcp/internal/server-plugins/deployment/adapter"
	deploymentDomain "github.com/dokku-mcp/dokku-mcp/internal/server-plugins/deployment/domain"
	deploymentInfra "github.com/dokku-mcp/dokku-mcp/internal/server-plugins/deployment/infrastructure"
	"github.com/dokku-mcp/dokku-mcp/pkg/config"
)

type fixture struct {
	client      *dokkutest.FakeClient
	apps        domain.ToolProvider
	deployments domain.ToolProvider
}

// newFixture wires the apps and deployment plugins to a fake Dokku that
// knows a single deployed application, "myapp".
func newFixture(t *testing.T) *fixture {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	client := dokkutest.NewFakeClient().
		On("apps:exists", func(args []string) ([]byte, error) {
			if args[0] == "myapp" {
				return nil, nil
			}
			return nil, &dokkuApi.NotFoundError{Command: "apps:exists", Err: dokkuApi.ErrAppNotFound}
		}).
		Respond("ps:report", "Deployed: true\nRunning: true\nPs scale: web=1\n").
		Respond("config:show", "=====> myapp env vars\nEXISTING:  keep me\n").
		Respond("logs", "2026-10-08T00:00:00 app[web.1]: hello\n")

	tracker := deploymentDomain.NewDeploymentTracker()
	deploymentSvc := deploymentAdapter.NewDeploymentServiceAdapter(
		deploymentDomain.NewApplicationDeploymentService(
			deploymentInfra.NewDeploymentRepository(logger),
			deploymentInfra.NewDeploymentInfrastructure(client, logger, tracker, nil),
			tracker,
			logger,
		),
	)

	logsConfig := config.LogsConfig{Runtime: config.RuntimeLogsConfig{DefaultLines: 100, MaxLines: 1000}}
	apps := NewAppsServerPlugin(infrastructure.NewDokkuApplicationRepository(client, logger), deploymentSvc, logger, logsConfig)

	return &fixture{
		client:      client,
		apps:        apps.(domain.ToolProvider),
		deployments: deployment.NewDeploymentServerPlugin(tracker, logger).(domain.ToolProvider),
	}
}

func TestToolsAreAnnotated(t *testing.T) {
	f := newFixture(t)
	plugintest.RequireAnnotated(t, f.apps)
	plugintest.RequireAnnotated(t, f.deployments)
}

func TestConfigureAppEncodesValues(t *testing.T) {
	f := newFixture(t)

	value := `postgres://u:p@db:5432/app?sslmode=require&x=$(rm -rf /) "quoted" spaced value`
	result := plugintest.CallTool(t, f.apps, "configure_app", plugintest.Args{
		"app_name": "myapp",
		"config":   plugintest.Args{"DATABASE_URL": value, "A_FLAG": ""},
	})
	plugintest.RequireSuccess(t, result)

	calls := f.client.CallsTo("config:set")
	if len(calls) != 1 {
		t.Fatalf("expected one config:set call, got %v", calls)
	}
	want := []string{
		"--encoded", "myapp",
		"A_FLAG=",
		"DATABASE_URL=" + base64.StdEncoding.EncodeToString([]byte(value)),
	}
	if !slices.Equal(calls[0].Args, want) {
		t.Fatalf("config:set args = %v, want %v", calls[0].Args, want)
	}
}

func TestConfigureAppWithoutRestart(t *testing.T) {
	f := newFixture(t)

	result := plugintest.CallTool(t, f.apps, "configure_app", plugintest.Args{
		"app_name": "myapp",
		"config":   plugintest.Args{"KEY": "v"},
		"restart":  false,
	})
	plugintest.RequireSuccess(t, result)

	calls := f.client.CallsTo("config:set")
	if len(calls) != 1 || !slices.Contains(calls[0].Args, "--no-restart") {
		t.Fatalf("expected config:set with --no-restart, got %v", calls)
	}
}

func TestConfigureAppRejectsInvalidKey(t *testing.T) {
	f := newFixture(t)

	result := plugintest.CallTool(t, f.apps, "configure_app", plugintest.Args{
		"app_name": "myapp",
		"config":   plugintest.Args{"BAD KEY": "v"},
	})
	plugintest.RequireError(t, result, "BAD KEY")
	if calls := f.client.CallsTo("config:set"); len(calls) != 0 {
		t.Fatalf("expected no config:set call, got %v", calls)
	}
}

func TestConfigureAppUnknownApp(t *testing.T) {
	f := newFixture(t)

	result := plugintest.CallTool(t, f.apps, "configure_app", plugintest.Args{
		"app_name": "ghost",
		"config":   plugintest.Args{"KEY": "v"},
	})
	plugintest.RequireError(t, result, "not found")
}

func TestScaleAppDoesNotRewriteConfig(t *testing.T) {
	f := newFixture(t)

	result := plugintest.CallTool(t, f.apps, "scale_app", plugintest.Args{
		"app_name":     "myapp",
		"process_type": "web",
		"instances":    3,
	})
	plugintest.RequireSuccess(t, result)

	if calls := f.client.CallsTo("ps:scale"); len(calls) != 1 || calls[0].String() != "ps:scale myapp web=3" {
		t.Fatalf("unexpected ps:scale calls: %v", calls)
	}
	if calls := f.client.CallsTo("config:set"); len(calls) != 0 {
		t.Fatalf("scaling must not rewrite config, got %v", calls)
	}
}

func TestDeployAppReturnsTrackableDeployment(t *testing.T) {
	f := newFixture(t)

	result := plugintest.CallTool(t, f.apps, "deploy_app", plugintest.Args{
		"app_name": "myapp",
		"repo_url": "https://github.com/dokku/smoke-test-app.git",
		"git_ref":  "v1.0.0",
	})
	started := plugintest.Structured[DeployStarted](t, result)
	if started.DeploymentID == "" {
		t.Fatalf("deploy_app must return a deployment_id, got %+v", started)
	}

	sync := f.client.CallsTo("git:sync")
	if len(sync) != 1 || sync[0].String() != "git:sync myapp https://github.com/dokku/smoke-test-app.git v1.0.0" {
		t.Fatalf("unexpected git:sync calls: %v", sync)
	}

	status := plugintest.Structured[deployment.DeploymentView](t, plugintest.CallTool(t, f.deployments, "get_deployment_status", plugintest.Args{
		"deployment_id": started.DeploymentID,
	}))
	if status.ID != started.DeploymentID || status.AppName != "myapp" || status.GitRef != "v1.0.0" {
		t.Fatalf("unexpected deployment status: %+v", status)
	}

	list := plugintest.Structured[deployment.DeploymentList](t, plugintest.CallTool(t, f.deployments, "list_deployments", plugintest.Args{
		"app_name": "myapp",
	}))
	if len(list.Deployments) != 1 || list.Deployments[0].ID != started.DeploymentID {
		t.Fatalf("unexpected deployment list: %+v", list)
	}
}

func TestDeployAppGitSyncFailure(t *testing.T) {
	f := newFixture(t)
	f.client.Fail("git:sync", errors.New("repository not found"))

	result := plugintest.CallTool(t, f.apps, "deploy_app", plugintest.Args{
		"app_name": "myapp",
		"repo_url": "https://github.com/dokku/missing.git",
	})
	plugintest.RequireError(t, result, "repository not found")
}

func TestGetDeploymentStatusUnknownID(t *testing.T) {
	f := newFixture(t)
	result := plugintest.CallTool(t, f.deployments, "get_deployment_status", plugintest.Args{"deployment_id": "nope"})
	plugintest.RequireError(t, result, "not found")
}

func TestGetRuntimeLogsRequestsLineCount(t *testing.T) {
	f := newFixture(t)

	result := plugintest.CallTool(t, f.apps, "get_runtime_logs", plugintest.Args{
		"app_name": "myapp",
		"lines":    50,
	})
	plugintest.RequireSuccess(t, result)
	if !strings.Contains(plugintest.Text(result), "hello") {
		t.Fatalf("expected log output, got %s", plugintest.Text(result))
	}

	calls := f.client.CallsTo("logs")
	if len(calls) != 1 || slices.Contains(calls[0].Args, "--tail") || slices.Contains(calls[0].Args, "-t") {
		t.Fatalf("logs must not follow the stream, got %v", calls)
	}
}

func TestCreateApp(t *testing.T) {
	f := newFixture(t)
	plugintest.RequireSuccess(t, plugintest.CallTool(t, f.apps, "create_app", plugintest.Args{"name": "new-app"}))
	if calls := f.client.CallsTo("apps:create"); len(calls) != 1 || calls[0].String() != "apps:create new-app" {
		t.Fatalf("unexpected calls: %v", calls)
	}
}

func TestCreateAppRejectsInvalidName(t *testing.T) {
	f := newFixture(t)
	result := plugintest.CallTool(t, f.apps, "create_app", plugintest.Args{"name": "Bad_Name!"})
	plugintest.RequireError(t, result, "invalid application name")
	if calls := f.client.CallsTo("apps:create"); len(calls) != 0 {
		t.Fatalf("expected no apps:create call, got %v", calls)
	}
}

func TestProcessStateTools(t *testing.T) {
	f := newFixture(t)
	for tool, command := range map[string]string{"restart_app": "ps:restart", "stop_app": "ps:stop", "start_app": "ps:start"} {
		plugintest.RequireSuccess(t, plugintest.CallTool(t, f.apps, tool, plugintest.Args{"app_name": "myapp"}))
		if calls := f.client.CallsTo(command); len(calls) != 1 || calls[0].String() != command+" myapp" {
			t.Fatalf("%s: unexpected calls %v", tool, calls)
		}
	}
	plugintest.RequireError(t, plugintest.CallTool(t, f.apps, "restart_app", plugintest.Args{"app_name": "ghost"}), "not found")
}

func TestGetFailedDeployLogs(t *testing.T) {
	f := newFixture(t)
	f.client.Respond("logs:failed", "web.1 | Error: Cannot find module 'express'\n")

	result := plugintest.CallTool(t, f.apps, "get_failed_deploy_logs", plugintest.Args{"app_name": "myapp"})
	plugintest.RequireSuccess(t, result)
	if !strings.Contains(plugintest.Text(result), "Cannot find module") {
		t.Fatalf("unexpected output: %s", plugintest.Text(result))
	}
}

func TestRollbackInfersRepositoryFromLastDeploy(t *testing.T) {
	f := newFixture(t)
	f.client.Respond("apps:report", "=====> myapp app information\n       App deploy source:             git-sync\n       App deploy source metadata:    https://github.com/acme/web.git#0a1b2c3d\n")

	result := plugintest.CallTool(t, f.apps, "rollback_app", plugintest.Args{"app_name": "myapp", "git_ref": "v1.2.0"})
	started := plugintest.Structured[DeployStarted](t, result)
	if started.DeploymentID == "" {
		t.Fatalf("expected a deployment id, got %+v", started)
	}
	sync := f.client.CallsTo("git:sync")
	if len(sync) != 1 || sync[0].String() != "git:sync myapp https://github.com/acme/web.git v1.2.0" {
		t.Fatalf("unexpected git:sync calls: %v", sync)
	}
}

func TestRollbackNeedsRepositoryForNonGitSyncDeploys(t *testing.T) {
	f := newFixture(t)
	f.client.Respond("apps:report", "App deploy source:   git-push\nApp deploy source metadata:  0a1b2c3d\n")

	result := plugintest.CallTool(t, f.apps, "rollback_app", plugintest.Args{"app_name": "myapp", "git_ref": "v1.2.0"})
	plugintest.RequireError(t, result, "repo_url")

	result = plugintest.CallTool(t, f.apps, "rollback_app", plugintest.Args{
		"app_name": "myapp", "git_ref": "v1.2.0", "repo_url": "https://github.com/acme/web.git",
	})
	plugintest.RequireSuccess(t, result)
}

// waitForDeployment polls get_deployment_status until the deployment is done.
func waitForDeployment(t *testing.T, f *fixture, id string) deployment.DeploymentView {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		view := plugintest.Structured[deployment.DeploymentView](t, plugintest.CallTool(t, f.deployments, "get_deployment_status", plugintest.Args{"deployment_id": id}))
		if view.Done {
			return view
		}
		if time.Now().After(deadline) {
			t.Fatalf("deployment %s did not finish: %+v", id, view)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestDeploymentSucceedsWhenRebuildSucceeds(t *testing.T) {
	f := newFixture(t)
	f.client.Respond("ps:rebuild", "-----> Building myapp\n-----> Build complete\n")

	started := plugintest.Structured[DeployStarted](t, plugintest.CallTool(t, f.apps, "deploy_app", plugintest.Args{
		"app_name": "myapp", "repo_url": "https://github.com/acme/web.git",
	}))
	view := waitForDeployment(t, f, started.DeploymentID)
	if view.Status != "succeeded" || !strings.Contains(view.BuildLogTail, "Build complete") {
		t.Fatalf("unexpected deployment: %+v", view)
	}
}

func TestDeploymentFailsWithBuildOutput(t *testing.T) {
	f := newFixture(t)
	f.client.On("ps:rebuild", func([]string) ([]byte, error) {
		return []byte("-----> Building myapp\nnpm ERR! missing script: build\n"), errors.New("exit status 1")
	})

	started := plugintest.Structured[DeployStarted](t, plugintest.CallTool(t, f.apps, "deploy_app", plugintest.Args{
		"app_name": "myapp", "repo_url": "https://github.com/acme/web.git",
	}))
	view := waitForDeployment(t, f, started.DeploymentID)
	if view.Status != "failed" || !strings.Contains(view.BuildLogTail, "missing script") || view.Error == "" {
		t.Fatalf("unexpected deployment: %+v", view)
	}
}

func TestFollowRuntimeLogs(t *testing.T) {
	f := newFixture(t)
	f.client.Respond("logs", "line one\nline two\nline three\n")

	result := plugintest.Structured[FollowedLogs](t, plugintest.CallTool(t, f.apps, "follow_runtime_logs", plugintest.Args{
		"app_name": "myapp", "seconds": 5,
	}))
	if !slices.Equal(result.Lines, []string{"line one", "line two", "line three"}) || result.Truncated {
		t.Fatalf("unexpected result: %+v", result)
	}
	if calls := f.client.CallsTo("logs"); len(calls) != 1 || calls[0].String() != "logs myapp -t" {
		t.Fatalf("unexpected calls: %v", calls)
	}
}

func TestFollowRuntimeLogsStopsAtMaxLines(t *testing.T) {
	f := newFixture(t)
	f.client.Respond("logs", "1\n2\n3\n4\n")

	result := plugintest.Structured[FollowedLogs](t, plugintest.CallTool(t, f.apps, "follow_runtime_logs", plugintest.Args{
		"app_name": "myapp", "max_lines": 2,
	}))
	if !slices.Equal(result.Lines, []string{"1", "2"}) || !result.Truncated {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestFollowRuntimeLogsUnknownApp(t *testing.T) {
	f := newFixture(t)
	plugintest.RequireError(t, plugintest.CallTool(t, f.apps, "follow_runtime_logs", plugintest.Args{"app_name": "ghost"}), "not found")
}

func TestExistsFailureIsNotReportedAsMissing(t *testing.T) {
	f := newFixture(t)
	f.client.Fail("apps:exists", errors.New("exit status 255: ssh: connect to host dokku port 22: Connection refused"))

	result := plugintest.CallTool(t, f.apps, "restart_app", plugintest.Args{"app_name": "myapp"})
	if !result.IsError || strings.Contains(plugintest.Text(result), "not found") {
		t.Fatalf("expected a connectivity error, got: %s", plugintest.Text(result))
	}
	if calls := f.client.CallsTo("ps:restart"); len(calls) != 0 {
		t.Fatalf("must not restart when existence is unknown: %v", calls)
	}
}
