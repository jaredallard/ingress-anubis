# yaml-language-server: $schema=https://json.schemastore.org/golangci-lint

# Linter settings
linters-settings:
  errcheck:
    check-blank: true
  gocyclo:
    min-complexity: 25
  gocritic:
    enabled-tags:
      - diagnostic
      - experimental
      - opinionated
      - performance
      - style
  lll:
    line-length: 140

linters:
  # Inverted configuration with enable-all and disable is not scalable
  # during updates of golangci-lint.
  disable-all: true
  enable:
    - bodyclose
    - dogsled
    - errcheck
    - errorlint
    - exhaustive
    - copyloopvar
    - gochecknoinits
    - gocritic
    - gocyclo
    - gofmt
    - goheader
    - goimports
    - gosec
    - gosimple
    - govet
    - ineffassign
    - lll
    - misspell
    - nakedret
    - staticcheck
    - revive
    - typecheck
    - unconvert
    - unparam
    - unused
    - whitespace

issues:
  exclude:
    # We allow error shadowing
    - 'declaration of "err" shadows declaration at'

  # Excluding configuration per-path, per-linter, per-text and per-source
  exclude-rules:
    # Exclude some linters from running on tests files.
    - path: _test\.go
      linters:
        - errcheck
        - funlen
        - gochecknoglobals # Globals in test files are tolerated.
        - gocyclo
        - goheader # Don't require license headers in test files.
        - gosec

output:
  sort-results: true
