# Cursor Rules for Dokku MCP Server - Enhanced with Go Documentation

This directory contains modern Cursor rules using the MDC format, enhanced with **official Go documentation best practices** for optimal human-LLM collaboration on the Dokku MCP Server project.

## Rules Structure

### Always Applied
- **project-workflow.mdc** - Main project context and development guidelines (always active)

### Auto-Attached Rules (by file pattern)
- **golang-standards.mdc** - Applied to all `*.go` files *(Enhanced with Effective Go principles)*
- **ddd-architecture.mdc** - Applied to files in `internal/**/*.go`
- **mcp-development.mdc** - Applied to MCP-specific handler files
- **testing-patterns.mdc** - Applied to all `*_test.go` files *(Enhanced with fuzzing and benchmarking)*
- **security-validation.mdc** - Applied to internal and cmd Go files *(Enhanced with crypto patterns)*
- **go-performance.mdc** - Applied to all `*.go` files *(New: Performance optimization patterns)*

## Enhanced Features

### From Official Go Documentation
✅ **Effective Go Principles** - Idiomatic code patterns and best practices  
✅ **Concurrency Patterns** - Worker pools, context usage, goroutine management  
✅ **Memory Management** - Buffer reuse, slice preallocation, sync.Pool usage  
✅ **Error Handling** - Proper error wrapping, custom error types, safe messages  
✅ **Database Best Practices** - Connection pooling, prepared statements, transactions  
✅ **Security Patterns** - SQL injection prevention, command sanitization, crypto usage  

### New Testing Capabilities
✅ **Fuzzing Support** - Go 1.18+ fuzzing patterns for robust testing  
✅ **Benchmarking** - Performance testing with memory allocation tracking  
✅ **Table-Driven Tests** - Official Go testing patterns  
✅ **Integration Testing** - Build tags and proper test isolation  

### Performance Optimization
✅ **Profiling Integration** - Built-in pprof support for performance analysis  
✅ **HTTP Client Optimization** - Connection pooling and timeout management  
✅ **Caching Strategies** - TTL cache implementation with cleanup  
✅ **Metrics Collection** - Performance monitoring and statistics  

## Migration from Legacy

The legacy `.cursorrules` file has been replaced with these focused, composable rules that follow [Cursor's best practices](https://docs.cursor.com/context/rules) and incorporate **official Go documentation patterns**:

✅ **Focused and actionable** - Each rule covers a specific domain  
✅ **Under 500 lines** - Concise and targeted guidance  
✅ **Concrete examples** - Real code patterns from Go documentation  
✅ **File references** - Links to project documentation  
✅ **Auto-attached** - Rules apply automatically based on file context  
✅ **Official Go Standards** - Aligned with Go team recommendations  

## Rule Types Used

- **alwaysApply: true** - Project workflow (core context)
- **globs: ["pattern"]** - Auto-attached based on file patterns
- **description** - Available for agent-requested inclusion

## Benefits

### For Development
- **Context-aware guidance** - Rules activate based on what you're working on
- **Official Go patterns** - Aligned with Go team recommendations
- **Performance-focused** - Built-in optimization patterns
- **Security-first** - Comprehensive security validation patterns
- **Comprehensive examples** - Real implementations from the project

### For LLM Collaboration
- **Automatic context** - No need to manually specify rules
- **Focused guidance** - Only relevant rules included in context
- **Rich examples** - Concrete patterns for code generation
- **Enhanced patterns** - Based on official Go documentation

## Usage

Rules are automatically applied by Cursor based on:
1. **File patterns** - Working on Go files triggers enhanced golang-standards.mdc
2. **Directory context** - Files in internal/ get DDD architecture guidance
3. **Always active** - Project workflow provides consistent context
4. **Performance rules** - go-performance.mdc applies to all Go files
5. **Manual selection** - Use @ruleName to include specific rules

The AI will automatically have access to the most relevant rules for your current work context, enhanced with official Go best practices.

## Code Quality Standards

### Strong Typing (No `any`)
- Use specific types instead of `interface{}`
- Create domain-specific types
- Proper type assertions with comma ok idiom

### Domain Driven Development
- Clear layer separation
- Business logic in domain layer
- Repository pattern for data access

### Performance Optimization
- Memory-efficient patterns from Go documentation
- Proper concurrency using official patterns
- Built-in profiling support

### Security First
- SQL injection prevention using prepared statements
- Command injection protection
- Cryptographic best practices

### Code Comments and Documentation
- Clear, descriptive comments for complex functions
- JSDoc-style comments for public APIs
- Self-documenting code with meaningful names

## Quick Start

When working on Go files, the enhanced rules will automatically provide:
- **Effective Go patterns** for idiomatic code
- **Performance optimization** suggestions
- **Security validation** patterns
- **Testing strategies** including fuzzing
- **Concurrency patterns** with proper context usage

The AI assistant will guide you through implementing these patterns while maintaining code simplicity and readability.

## Resources

### Official Go Documentation Applied
- [Effective Go](https://go.dev/doc/effective_go) - Core principles integrated
- [Go Testing](https://pkg.go.dev/testing) - Testing patterns enhanced
- [Go Security](https://go.dev/security/) - Security patterns included
- [Go Performance](https://go.dev/doc/faq#Performance) - Optimization patterns

### Project Documentation
- **Architecture**: @docs/architecture.md
- **Development Playbook**: @docs/playbooks/development.md
- **Dokku Analysis**: @docs/dokku-analysis.md
- **Project Summary**: @docs/project-summary.md 