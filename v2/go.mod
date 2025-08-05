module github.com/elchinoo/stormdb/v2

go 1.21

require github.com/elchinoo/stormdb/v2/core v0.0.0

require (
	github.com/gorilla/mux v1.8.1 // indirect
	github.com/kr/pretty v0.3.0 // indirect
	github.com/lib/pq v1.10.9 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/elchinoo/stormdb/v2/core => ./core
