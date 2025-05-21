## SPDX-FileCopyrightText: 2025 M. Shulhan <ms@kilabit.info>
## SPDX-License-Identifier: GPL-3.0-only

.PHONY: all
all: lint test

.PHONY: lint
lint:
	go run ./internal/cmd/gocheck ./...
	go vet ./...

.PHONY: test
test:
	go test ./...
