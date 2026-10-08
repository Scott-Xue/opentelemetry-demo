# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0

FROM docker.io/library/golang:1.26.5-bookworm AS builder

WORKDIR /usr/src/app/

COPY ./src/fraud-api/go.mod go.mod
COPY ./src/fraud-api/screen.go screen.go
COPY ./src/fraud-api/main.go main.go

# No go.sum and no `go mod download`: this service has no dependencies, by design.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w" -o fraud-api .

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /usr/src/app/
COPY --from=builder /usr/src/app/fraud-api ./

EXPOSE 8080
ENTRYPOINT [ "./fraud-api" ]
