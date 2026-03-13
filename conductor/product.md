# Product Guide — gotham

## Overview

`gotham` (`github.com/TouchBistro/gotham`) is a Go utility library providing reusable types and functions for TouchBistro's Go microservices. It reduces boilerplate and standardizes patterns around authentication, session handling, caching, and HTTP middleware.

## Target Users

**Primary:** TouchBistro backend engineers building Go microservices who need consistent, well-tested building blocks for auth, JWT, cache, and HTTP handling.

## Problems Solved

1. **Reduce boilerplate** — Eliminates the need to re-implement auth, JWT validation, cache adapters, and middleware in every service.
2. **Standardize patterns** — Enforces consistent auth/session/caching approaches across TouchBistro's Go service fleet.

## Key Features

- **HTTP utilities** (`http/`): Auth policy enforcement, JWT configuration, principal loading, role/permission sets, middleware, and handlers for both `gin` and `net/http`.
- **Cache abstractions** (`cache/`): Pluggable cache interface with memory, Redis, and nil (no-op) implementations, plus serialization helpers.
- **Utilities** (`util/`): JWT helpers and other shared utility functions.

## Package Structure

| Package | Purpose |
|---------|---------|
| `github.com/TouchBistro/gotham/http` | Auth, JWT, middleware, gin & net/http handlers |
| `github.com/TouchBistro/gotham/cache` | Cache interface + memory/Redis/nil implementations |
| `github.com/TouchBistro/gotham/util` | JWT and general utility helpers |

## Success Metrics

- **Developer experience**: Library is easy to integrate, well-documented, and has a clear, stable API surface.
- Consuming engineers can add auth/cache/middleware to a new service with minimal configuration.
- New packages and features are discoverable through godoc and README examples.
