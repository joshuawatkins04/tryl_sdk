# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2026-01-30

### Added

- Initial release of Tryl Go SDK
- `Client` with `Log`, `LogAsync`, `LogBatch`, and `List` methods
- Event batching support with configurable batch size and flush interval
- Automatic retry with exponential backoff for transient errors
- Typed errors (`APIError`, `NetworkError`) with helper functions
- Configuration options: `WithBaseURL`, `WithTimeout`, `WithRetry`, `WithBatching`, `WithHTTPClient`, `WithUserAgent`
- Examples for basic usage, async logging, and batching
