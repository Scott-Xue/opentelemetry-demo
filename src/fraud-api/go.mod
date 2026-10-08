// Simulated third-party fraud screening vendor.
//
// Deliberately has NO dependencies. This service stands in for a SaaS vendor the
// application cannot instrument, so it must not carry an OpenTelemetry SDK -- the
// absence of one is the first layer of the observability boundary described in
// docs/superpowers/specs/2026-10-08-believable-fraud-detection-design.md.
module github.com/open-telemetry/opentelemetry-demo/src/fraud-api

go 1.26
