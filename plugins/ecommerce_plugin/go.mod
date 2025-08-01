module github.com/elchinoo/stormdb/plugins/ecommerce_plugin

go 1.24.4

require (
	github.com/elchinoo/stormdb v0.0.0
	github.com/jackc/pgx/v5 v5.7.5
)

replace github.com/elchinoo/stormdb => ../../

require (
	github.com/Masterminds/semver/v3 v3.4.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/stretchr/testify v1.10.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/crypto v0.37.0 // indirect
	golang.org/x/sync v0.13.0 // indirect
	golang.org/x/text v0.24.0 // indirect
)
