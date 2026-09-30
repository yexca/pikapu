// Package buildinfo carries values injected at link time.
package buildinfo

// Version is set with -ldflags "-X pikapu/internal/buildinfo.Version=vX.Y.Z"
// from the repository VERSION file.
var Version = "dev"
