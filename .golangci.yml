# yaml-language-server: $schema=https://json.schemastore.org/golangci-lint

version: "2"

# Linter settings
linters:
  settings:
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

  # Inverted configuration with enable-all and disable is not scalable
  # during updates of golangci-lint.
  default: none
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
    - goheader
    - gosec
    - govet
    - ineffassign
    - lll
    - misspell
    - nakedret
    - staticcheck
    - revive
    - unconvert
    - unparam
    - unused
    - whitespace

  # Excluding configuration per-path, per-linter, per-text and per-source
  exclusions:
    rules:
      # We allow error shadowing
      - path: '(.+)\.go$'
        text: 'declaration of "err" shadows declaration at'
      # Exclude some linters from running on tests files.
      - path: _test\.go
        linters:
          - errcheck
          - funlen
          - gochecknoglobals # Globals in test files are tolerated.
          - gocyclo
          - goheader # Don't require license headers in test files.
          - gosec

# formatter settings
formatters:
  enable:
    - gofmt
    - goimports
