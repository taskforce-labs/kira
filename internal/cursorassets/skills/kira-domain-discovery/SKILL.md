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
   - Domain model diagrams (mermaid markdown) and text descriptions
   - Context map (mermaid)
   - Store in configured artifact location
   - Map


## Intervention Points

- When bounded contexts are ambiguous
- When context boundaries need stakeholder validation
- When upstream/downstream relationships need clarification

At each intervention point, present options and guide the user to make informed decisions. When presenting clarifying questions with selectable options, follow the clarifying-questions-format skill.

---
name: domain-discovery
description: Systematically discover, validate, and document the domain model and context boundaries of a product. Use pragmatic domain modeling and context mapping to support architecture, team topology, and decision-making.
disable-model-invocation: false
---

# Domain Discovery & Modeling

Guide the user through **practical domain discovery**, from understanding the business problem to producing **clear domain models and a validated context map** that can be used for architecture, team design, and planning.

This skill favors **decision clarity over completeness**. The goal is not a perfect model, but a *useful* one.

---

## When to Use

Use this skill when you need to:

- Define or validate **bounded contexts**
- Align system design with **business capabilities**
- Clarify **domain language and meaning**
- Identify **ownership boundaries** between teams or systems
- Prepare for modernisation, decomposition, or platform work

---

## Outcomes (What Good Looks Like)

By the end of this skill, you should have:

- A shared **ubiquitous language** with clear definitions
- One or more **domain models** showing key concepts and relationships
- A **context map** showing bounded contexts and their relationships
- Explicit **assumptions and open questions**
- Clear **next decisions** (e.g. where to split, integrate, or protect models)

---

## Instructions

### 1. Read Project Configuration & Constraints

- Review `kira.yaml` and configured artifact locations
- Identify:
  - Product goals and success criteria
  - Known constraints (regulatory, organisational, technical)
  - Existing systems or domains (if any)
- Respect project conventions for naming and artifacts

> **Intervention point:** If goals or constraints are unclear, pause and surface assumptions.

---

### 2. Discover the Domain (Language First)

Focus on **how the business talks**, not how systems work.

- Identify:
  - Core entities (things with identity)
  - Value objects (descriptive, immutable concepts)
  - Aggregates and consistency boundaries
- Capture:
  - Ubiquitous language (terms + definitions)
  - Synonyms and overloaded terms
  - Domain events (“something meaningful happened”)

Prefer real examples and scenarios over abstract definitions.

> **Intervention point:** Highlight terms that appear to mean different things in different places.

---

### 3. Shape the Domain Model

- Create a **conceptual domain model** (not a database schema)
- Show:
  - Key concepts and their relationships
  - Aggregate boundaries where relevant
- Keep models:
  - Simple
  - Focused on behaviour and meaning
  - Free of technical implementation detail

Use Mermaid diagrams where appropriate.

---

### 4. Identify Bounded Contexts

Using the domain model and language:

- Propose candidate bounded contexts
- For each context, document:
  - Purpose and responsibility
  - Key concepts it owns
  - What it explicitly does *not* own

Bounded contexts should align with:
- Differences in meaning
- Change frequency
- Ownership or team boundaries

> **Intervention point:** If boundaries are fuzzy, present multiple plausible options with trade-offs.

---

### 5. Create the Context Map

Map how bounded contexts relate to each other.

For each relationship, identify:
- Direction (upstream / downstream)
- Relationship type:
  - Partnership
  - Shared Kernel
  - Customer–Supplier
  - Conformist
  - Anti-Corruption Layer (ACL)
  - Open Host Service (OHS)
  - Separate Ways
- Key risks or constraints at the boundary

This is a **semantic map**, not a deployment diagram.

---

### 6. Validate With Scenarios

Stress-test the model using real workflows:

- Walk through common business scenarios
- Trace how concepts move across contexts
- Look for:
  - Leaky abstractions
  - Accidental coupling
  - Ambiguous ownership

Refine models and boundaries based on findings.

---

### 7. Create and Store Artifacts

Produce the following artifacts:

- Ubiquitous language glossary (Markdown)
- Domain model diagrams (Mermaid + explanation)
- Context map (Mermaid + relationship notes)
- Open questions and assumptions

Store artifacts in the configured location.

---

## Guiding Principles

- Language beats diagrams
- Boundaries matter more than entities
- If everything is shared, nothing is owned
- Translation is a feature, not a failure
- “Separate ways” is often the healthiest choice

---

## Intervention Points Summary

Pause and involve the user when:

- Domain language conflicts or overlaps
- Context boundaries are unclear
- Relationship types imply strong coupling
- Organisational reality contradicts the model

At each intervention point:
- Present options
- Explain trade-offs
- Guide the user toward an explicit decision

Use the `clarifying-questions-format` skill when offering choices.