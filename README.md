# Zigflow Schema

This repository contains the versioned JSON Schema definitions for the Zigflow DSL.

The schema acts as the contract between Zigflow tooling, including:

* Zigflow CLI
* Zigflow Studio
* Documentation
* Validation tooling
* External integrations

<!-- toc -->

* [Purpose](#purpose)
* [Consuming the schema](#consuming-the-schema)
* [Relationship to Zigflow](#relationship-to-zigflow)
* [Consumers](#consumers)
* [Goals](#goals)
* [Contributing](#contributing)
  * [Open in a container](#open-in-a-container)
  * [Commit style](#commit-style)
* [License](#license)

<!-- Regenerate with "pre-commit run -a markdown-toc" -->

<!-- tocstop -->

## Purpose

The schema defines the structure of valid Zigflow workflows.

It is the authoritative description of the Zigflow DSL and is intended to evolve
independently from individual tooling implementations.

The schema version should therefore be considered distinct from any specific CLI
or Studio version.

## Consuming the schema

Most users should not consume this repository directly.

The recommended way to access the Zigflow schema is through the
[https://github.com/zigflow/zigflow](Zigflow CLI).

The CLI provides a stable interface for retrieving the schema and ensures the
schema version is compatible with the installed Zigflow tooling.

This repository primarily exists as the source of truth for schema definitions
and for consumers that have a specific need to work with schema artefacts directly.

If you are building tooling on top of Zigflow, consider whether consuming the
schema via the Zigflow CLI is sufficient before depending directly on this
repository.

## Relationship to Zigflow

This repository defines the schema only.

It does not:

* Execute workflows
* Validate runtime behaviour
* Provide a workflow engine
* Generate code

Those responsibilities belong to Zigflow tooling such as the CLI and Studio.

## Consumers

Typical consumers include:

* Zigflow CLI
* Zigflow Studio
* Documentation generators
* IDE integrations
* Validation tools
* AI assistants and MCP servers

## Goals

* Stable and versioned
* Language agnostic
* Tooling independent
* Suitable for validation and code generation
* Easy to consume from multiple ecosystems

## Contributing

### Open in a container

* [Open in a container](https://code.visualstudio.com/docs/devcontainers/containers)

### Commit style

All commits must be done in the [Conventional Commit](https://www.conventionalcommits.org)
format.

```git
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

## License

[Apache License 2.0](./LICENSE)
