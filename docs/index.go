package docs

import (
	_ "embed"
)

//go:embed docs.html
var DocsHTML []byte
