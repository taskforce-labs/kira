---
name: domain-discovery
description: Discover and articulate the domain and context of the product. Use domain modeling techniques and context mapping approaches. Use when defining bounded contexts or aligning with business capabilities.
disable-model-invocation: false
---

# Domain Discovery

Guide the user through domain discovery and context mapping.

## When to Use

- Defining bounded contexts
- Aligning technical design with business capabilities
- Understanding domain language and aggregates
- Identifying context boundaries and relationships

## Instructions

1. **Read Project Configuration**
   - Check `kira.yaml` and artifact locations
   - Use project conventions for domain artifacts

2. **Discover Domain Language**
   - Identify entities, aggregates, and value objects
   - Document ubiquitous language
   - Map domain events

3. **Context Mapping**
   - Identify bounded contexts
   - Map relationships (partnership, shared kernel, customer-supplier, conformist, ACL, OHS)
   - Document context map

4. **Create Artifacts**
   - Domain model diagrams or descriptions
   - Context map
   - Store in configured artifact location

## Intervention Points

- When bounded contexts are ambiguous
- When context boundaries need stakeholder validation
- When upstream/downstream relationships need clarification

At each intervention point, present options and guide the user to make informed decisions.
