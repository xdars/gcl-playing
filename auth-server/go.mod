module auth-server

go 1.23.0

toolchain go1.23.10

require golang.org/x/oauth2 v0.30.0

require shared/jwt v0.0.0

require (
	github.com/matoous/go-nanoid/v2 v2.1.0
	shared/database v0.0.0
)

replace shared/jwt => ../shared/jwt

replace shared/database => ../shared/database

require (
	cloud.google.com/go/compute/metadata v0.3.0 // indirect
	github.com/golang-jwt/jwt/v5 v5.3.0 // indirect
	github.com/mattn/go-sqlite3 v1.14.30 // indirect
)
