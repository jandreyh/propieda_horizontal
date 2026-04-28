// Package version contiene la version del binario, escrita en build time.
package version

// Version es la version semver del binario. Se sobrescribe en build via
// `-ldflags "-X github.com/saas-ph/api/internal/version.Version=..."`.
var Version = "0.0.0-dev"
