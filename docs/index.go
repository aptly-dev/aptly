package docs

import (
	_ "embed" // embed html below

	_ "github.com/swaggo/swag" // make sure swag is in go.mod
)

//go:embed docs.html
var DocsHTML []byte
