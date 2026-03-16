# Documentation

Overview of project documentation. Use this folder for long-lived reference material (ADRs, guides, product docs, reports). Work items and specs live in .work instead.

## Planning and roadmap

- **PLAN.md** – The free-form planning document lives under the docs folder. With the default `docs_folder` (`.docs`), the path is `.docs/PLAN.md`. Configure `docs_folder` in `kira.yml` to use a different path; the absolute path is `config_dir/docs_folder/PLAN.md`.
- **Plan vs roadmap** – PLAN.md is prose-first intent (goals, sequencing, workstreams). ROADMAP.yml is the structured derivative (YAML tree of work item refs, ad-hoc items, groups). Workflow: extract from product/use-case docs → PLAN.md → generate/update ROADMAP.yml (e.g. via LLM or a future plan-to-roadmap tool). Use `kira roadmap lint`, `kira roadmap apply`, and `kira roadmap draft` / `kira roadmap promote` to validate, promote ad-hoc items, and manage drafts. Plan-to-roadmap extraction is not automated in v1; use manual editing or an external tool to produce or update ROADMAP.yml from PLAN.md.

## Sections

- [Agents](agents/) – Agent-specific documentation (e.g. using kira)
- [Architecture](architecture/) – Architecture Decision Records and diagrams
- [Product](product/) – Product vision, roadmap, personas, glossary, feature briefs
- [Reports](reports/) – Release reports, metrics, audits, retrospectives
- [API](api/) – API reference
- [Guides](guides/) – Development and usage guides (including security)
