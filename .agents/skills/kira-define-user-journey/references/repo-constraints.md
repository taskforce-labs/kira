## Repo constraints: `## Questions` headings (`kira questions`)

When authoring markdown intended to be scanned by `kira questions`:

- **Preferred authoring form**: use `## Questions: <theme>` with a **colon**.
- **Compatibility (parsing)**: the parser should also accept:
  - `## Questions — <theme>`
  - `## Questions - <theme>`
- **Untitled is valid**: plain `## Questions` is allowed for an untitled section.
- **Per-question headings**: within a questions section, each question should be `### N. Title` with numeric `N`.

Agent output should **not teach or emit** the dash forms; keep those as compatibility for existing human-written headings.
