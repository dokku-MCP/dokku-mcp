# Dokku Compatibility Testing Strategy

## Overview

This project maintains compatibility with multiple Dokku versions using an automated testing matrix. To balance comprehensive coverage with CI efficiency, we follow a **"Last 4 Minors"** strategy.

## Current strategy: Last 4 Minors, Latest Patch

### Approach
- Keep only the **latest patch version** of the **last 4 minor releases**
- Plus `master` branch for bleeding-edge compatibility testing
- **Total: 5 test runs per commit** (down from 13+ previously)

### Current Test Matrix
As of last update:
- `v0.33.9` - Latest from 0.33.x line
- `v0.34.9` - Latest from 0.34.x line  
- `v0.35.20` - Latest from 0.35.x line
- `v0.36.7` - Latest from 0.36.x line (current stable)
- `master` - Bleeding edge (auto-added, allowed to fail)

## Automation

### File: `.tested-dokku-versions`
- Single source of truth for tested versions
- Maintained automatically by GitHub Actions
- Supports comments (lines starting with `#`)

### Workflow: `dokku-release-monitor.yml`
**Schedule:** Daily at 20:12 UTC

**Actions:**
1. Checks Dokku repository for new stable releases
2. If new version detected:
   - Adds to `.tested-dokku-versions`
   - Automatically prunes to keep only last 4 minors
   - Commits and pushes changes
   - Triggers compatibility tests for the new version
   - Creates GitHub issue to notify maintainers

**Pruning Logic:**
```python
# Groups versions by (major, minor)
# Keeps only the highest patch version in each group
# Sorts minor versions and keeps last 4
```

### Workflow: `compatibility.yml`
**Triggers:**
- Push to `main` branch (full matrix)
- New Dokku release detected (single version)
- Manual dispatch (configurable)

**Matrix Setup:**
- Reads versions from `.tested-dokku-versions`
- Filters comments and empty lines
- Adds `master` branch automatically
- Runs integration tests on each version

## Benefits

### Before
- **13 versions tested** on every commit
- Long CI times (13+ parallel jobs)
- Redundant coverage (multiple patches of same minor)

### After
- **5 versions tested** on every commit
- Faster CI times (5 parallel jobs = ~60% reduction)
- Focused coverage (critical minors only)
- Automatic maintenance (no manual pruning needed)

## Manual Operations

### Testing a Specific Version
Use workflow dispatch in GitHub Actions:
```
Test type: single-version
Dokku version: v0.35.20
```

### Adding a Version Manually
Edit `.tested-dokku-versions`:
```bash
# Add version
echo "v0.37.0" >> .tested-dokku-versions

# Run the monitor workflow manually to prune
# Or wait for next scheduled run
```

### Emergency: Test All Versions
Temporarily disable the pruning logic in `dokku-release-monitor.yml` or manually populate `.tested-dokku-versions` with all desired versions.

## Configuration

### Adjusting Strategy
To change from 4 to N minors, edit `dokku-release-monitor.yml`:

```python
# Line 84: Change [:4] to [:N]
sorted_minors = sorted(latest_by_minor.keys(), reverse=True)[:4]
```

### Excluding Versions
Comment out versions in `.tested-dokku-versions`:
```
v0.33.9
# v0.34.9  # Skip this version temporarily
v0.35.20
v0.36.7
```

## Maintenance Notes

- Monitor the `dokku-release` issues for new version notifications
- Review compatibility test results in Actions tab
- Adjust strategy if Dokku release cadence changes
- Consider testing older versions if supporting legacy deployments

