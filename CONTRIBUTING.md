# Contributing to Dokku MCP Server

First off, thank you for considering contributing! Your help is greatly appreciated. This document provides guidelines for contributing to this project.

## How Can I Contribute?

There are many ways to contribute, from writing tutorials or blog posts, improving the documentation, submitting bug reports and feature requests or writing code which can be incorporated into the main project.

### Reporting Bugs

- **Ensure the bug was not already reported** by searching on GitHub under [Issues](https://github.com/dokku-mcp/dokku-mcp/issues).
- If you're unable to find an open issue addressing the problem, [open a new one](https://github.com/dokku-mcp/dokku-mcp/issues/new). Be sure to include a **title and clear description**, as much relevant information as possible, and a **code sample** or an **executable test case** demonstrating the expected behavior that is not occurring.

### Suggesting Enhancements

- Open a new issue to discuss your enhancement.
- Clearly describe the enhancement, its motivation, and its use case.

## Development Workflow

Here is the recommended workflow for contributing code.

### 1. Setup Your Environment

Follow the [Getting Started](./README.md#getting-started) guide in the README to set up your local development environment. The key steps are:

```bash
git clone https://github.com/dokku-mcp/dokku-mcp.git
cd dokku-mcp
make install-tools
make setup-dev
```

### 2. Create a Feature Branch

Start your work from a feature branch created from the `main` branch.

```bash
git checkout -b feature/your-amazing-feature
```

### 3. Write Code and Tests

- Write your code, following the project's [Architecture](#architecture) and [Code Style](#code-style) guidelines.
- **Add tests!** Your patch won't be accepted if it doesn't have tests. Run tests with `make test`.

### 4. Code Quality Checks

Before committing, ensure your code passes all quality checks. The pre-commit hook you installed with `make setup-dev` will run most of these automatically.

You can also run them manually:
```bash
make check
```

### 5. Submit a Pull Request

1.  Push your feature branch to your fork on GitHub.
2.  Open a pull request against the `main` branch of the `dokku-mcp` repository.
3.  Provide a clear description of your changes.
4.  Ensure all status checks are passing.

## Git Hooks

We use Git hooks to enforce code style and run checks before you commit. The `make setup-dev` command configures this for you automatically by setting `core.hooksPath` to `.githooks/`.

The `pre-commit` hook currently performs these actions:
- ✅ **Auto-formats** Go code with `goimports`/`gofmt`.
- ✅ **Runs static analysis** with `go vet`.
- ✅ **Enforces strong typing** rules (e.g., blocks `interface{}`).
- ✅ **Runs quick tests** on modified packages.

## Code Review Checklist

When reviewing pull requests, we use the following checklist:

- [ ] Does the code have unit tests?
- [ ] Is the documentation updated to reflect the changes?
- [ ] Is there proper error handling for all new paths?
- [ ] Is there proper input validation and sanitization?
- [ ] Are logs added for important operations?
- [ ] Has performance been considered (e.g., avoiding N+1 queries)?

## Architecture & Code Style

- **Domain-Driven Design**: Follow the existing DDD patterns. See `docs/playbooks/development.md` for detailed guides on adding new features.
- **Strong Typing**: Avoid `interface{}` and `any` unless absolutely necessary and justified.
- **Error Handling**: Use `fmt.Errorf` with `%w` to wrap errors with context.
- **Go Conventions**: Follow standard Go formatting and style (`gofmt`, `go vet`).

Thank you for your contribution!