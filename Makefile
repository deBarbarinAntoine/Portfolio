# Include variables from the .envrc file
include .envrc

# =================================================================================== #
# HELPERS
# =================================================================================== #

## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

.PHONY: confirm
confirm:
	@echo -n 'Are you sure? [y/N] ' && read ans && [ $${ans:-N} = y ]

# =================================================================================== #
# DEVELOPMENT
# =================================================================================== #

## run: run the cmd/web application
.PHONY: run
run:
	@go run ./cmd/web -port=${PORT} -dsn=${DB_DSN} -smtp-sender="${SMTP_SENDER}" -smtp-username=${SMTP_USERNAME} -smtp-password=${SMTP_PASS} -smtp-host=${SMTP_HOST} -smtp-port=${SMTP_PORT}

## db/psql: connect to the database using mysql
.PHONY: db/psql
db/psql:
	@psql ${DB_DSN}

## db/migrations/new name=$1: create a new database migration
.PHONY: db/migrations/new
db/migrations/new:
	@echo 'Creating migration files for ${name}'
	migrate create -seq -ext .sql -dir ./migrations ${name}

## db/migrations/up: apply all up database migrations
.PHONY: db/migrations/up
db/migrations/up: confirm
	@echo 'Running up migrations...'
	@migrate -path ./migrations -database ${DB_DSN} up

## db/migrations/drop: drop database migrations
.PHONY: db/migrations/drop
db/migrations/drop: confirm
	@echo 'Dropping migrations...'
	@migrate -path ./migrations -database ${DB_DSN} drop

# =================================================================================== #
# QUALITY CONTROL
# =================================================================================== #

## audit: tidy dependencies and format, vet and test all code
.PHONY: audit
audit: vendor
	@echo 'Formatting code...'
	go fmt ./...
	@echo 'Vetting code...'
	go vet ./...
	go run honnef.co/go/tools/cmd/staticcheck@latest -checks=all,-ST1000,-U1000 ./...
	@echo 'Running tests...'
	go test -race -vet=off ./...

## vendor: tidy and vendor dependencies
.PHONY: vendor
vendor:
	@echo 'Tidying and verifying module dependencies...'
	go mod tidy
	go mod verify
	@echo 'Vendoring dependencies'
	go mod vendor

# =================================================================================== #
# BUILD
# =================================================================================== #

## build: build the cmd/web application
.PHONY: build
build:
	@echo 'Building cmd/web...'
	go build -ldflags='-s' -o=./bin/portfolio ./cmd/web

# =================================================================================== #
# PRODUCTION
# =================================================================================== #

## bin: execute the bin application in ./bin/linux_amd64
.PHONY: bin
bin:
	@echo 'Executing binary...'
	@./bin/linux_amd64 -port=${PORT} -dsn=${DB_DSN} -smtp-sender="${SMTP_SENDER}" -smtp-username=${SMTP_USERNAME} -smtp-password=${SMTP_PASS} -smtp-host=${SMTP_HOST} -smtp-port=${SMTP_PORT}