# ADR 0001: Go clean-architecture backend

Status: accepted

The prototype TypeScript backend is replaced by Go commands for API and worker processes. Domain and application packages cannot import infrastructure adapters. External systems are accessed behind narrow ports, allowing deterministic fake implementations in tests. OpenAPI is the cross-language source of truth; disconnected prototype interfaces are not preserved.

This raises initial replacement cost but prevents ORM/provider concerns from leaking into learning policies and gives both runtimes one versioned contract.
