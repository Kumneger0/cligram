// nolint
package assets

import _ "embed"

// The //go:embed directive is a compiler instruction.
// It finds the file at the specified path (relative to this Go file)
// and embeds its raw binary content into the JSBackendBinary variable.
//
//go:embed resources/cligram-js-backend
var JSBackendBinary []byte
