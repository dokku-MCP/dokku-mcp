# Development Setup - Git Hooks

## Quick Setup (Recommended)

After cloning the repository, run:

```bash
make setup-dev
```

This will:
- Configure Git to use tracked hooks from `.githooks/`
- Install required Go development tools
- Make all hooks executable
- Verify the setup

## How It Works

### The Problem
Git hooks in `.git/hooks/` are **not tracked by Git**, so they don't get cloned with the repository. Each developer would need to manually set them up.

### The Solution
We use **tracked hooks** in `.githooks/` directory:

1. **`.githooks/`** - Contains all hooks (tracked by Git)
2. **`git config core.hooksPath .githooks`** - Tells Git to use our tracked directory
3. **`scripts/setup-dev.sh`** - One-time setup script for new developers

## Manual Setup

If you prefer manual setup:

```bash
# Configure Git to use tracked hooks
git config core.hooksPath .githooks

# Make hooks executable
chmod +x .githooks/*

# Install development tools
make install-tools
```

## What the Pre-commit Hook Does

✅ **Auto-formats** Go code with `goimports`/`gofmt`
✅ **Validates syntax** to prevent broken commits  
✅ **Runs static analysis** with `go vet`
✅ **Enforces strong typing** - blocks `interface{}`, `any`, etc.
✅ **Runs quick tests** on modified packages

## Onboarding as New Developer

```bash
git clone <repository-url>
cd dokku-mcp
make setup-dev
```

## Available Configurations

| Command | Description |
|---------|-------------|
| `make setup-dev` | Full development environment setup |
| `make setup-hooks` | Legacy: copies to `.git/hooks/` (not recommended) |
| `make install-tools` | Install Go development tools only |

## Troubleshooting

### Hook Not Running
```bash
# Check if hooksPath is configured
git config core.hooksPath

# Should output: .githooks
```

### Hook Permission Error
```bash
# Make hooks executable
chmod +x .githooks/*
```

### Missing Tools
```bash
# Install development tools
make install-tools
```
