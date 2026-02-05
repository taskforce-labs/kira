module kira

// Note: Go 1.25.7+ is required to avoid standard library vulnerabilities (net/url,
// crypto/tls, crypto/x509) reported by govulncheck. See GO-2025-4010, GO-2026-4341,
// GO-2026-4340, GO-2025-4175, GO-2025-4155, GO-2025-4007, GO-2026-4337.
go 1.25.7

require (
	github.com/google/go-github/v61 v61.0.0
	github.com/spf13/cobra v1.8.0
	github.com/stretchr/testify v1.8.4
	golang.org/x/oauth2 v0.28.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
)
