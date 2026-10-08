// Package buildinfo holds build metadata injected into the server binary.
package buildinfo

// Version is the server version reported to MCP clients. cmd/server sets it
// from the value injected via -ldflags at build time.
var Version = "dev"
