module kira

// Note: Go 1.25.2+ is required to avoid GO-2025-4010 vulnerability in net/url.
// The vulnerability affects URL validation functions used in field configuration.
// See: https://pkg.go.dev/vuln/GO-2025-4010
go 1.25.2

require (
	github.com/spf13/cobra v1.8.0
	github.com/stretchr/testify v1.8.4
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
)
