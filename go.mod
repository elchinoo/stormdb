module github.com/elchinoo/stormdb

go 1.21

replace github.com/elchinoo/stormdb/core => ./core

require github.com/elchinoo/stormdb/core v0.0.0-00010101000000-000000000000

require (
	github.com/gorilla/mux v1.8.1 // indirect
	github.com/lib/pq v1.10.9 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
