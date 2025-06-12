## SPDX-FileCopyrightText: 2025 M. Shulhan <ms@kilabit.info>
## SPDX-License-Identifier: GPL-3.0-only

COVER_OUT:=cover.out
COVER_HTML:=cover.html

.PHONY: all
all: lint test

.PHONY: lint
lint:
	go run ./internal/cmd/gocheck ./...
	go vet ./...

.PHONY: test
test:
	CGO_ENABLED=1 go test -failfast -timeout=1m -race \
		-coverprofile=$(COVER_OUT) ./...
	go tool cover -html=$(COVER_OUT) -o $(COVER_HTML)

.PHONY: doc.serve
doc.serve:
	ciigo serve .
