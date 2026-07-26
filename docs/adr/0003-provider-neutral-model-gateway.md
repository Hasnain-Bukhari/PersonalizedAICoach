# ADR 0003: Provider-neutral model gateway

Status: accepted

Agents call a task alias through a model gateway rather than a provider SDK. The primary route is an OpenAI-compatible local endpoint; optional Bedrock, OpenAI, and Anthropic adapters are fallback routes. JSON-schema validation, timeouts, concurrency limits, circuit breaking, telemetry, citations, and repair attempts apply consistently.

Model aliases and prompt versions are configuration, not API contracts. Application pods never contain model weights; production inference uses isolated GPU nodes.
