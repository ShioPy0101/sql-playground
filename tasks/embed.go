package taskdata

import "embed"

// FS contains bundled task definitions for serverless deployments.
//
//go:embed *.json
var FS embed.FS
