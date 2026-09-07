# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

### Added

- Added the initial Go application scaffold for the `simply-cli` executable in
	the `go-simply-cli` repository.
- Added credential resolution from flags, environment variables, and
	terminal-aware prompts without persistent secret storage.
- Added `auth check` with Bearer and Basic authentication support, stable table
	and JSON success output, rate-limit-aware errors, and redacted failures.
- Added `products list`, `dns records list`, and `dns zone show` with table and
	JSON output, deterministic ordering, and empty-result handling.