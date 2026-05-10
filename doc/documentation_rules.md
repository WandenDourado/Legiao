# Documentation Guidelines

## File Placement
* **Feature documentation** – Place in the `/doc` directory with a concise filename (`feature-name.md`).
* **Changelog** – Use `doc/changelog.md` to record chronological updates. Each entry should start with a date in `DD/MM/YYYY HH:MM` format followed by a brief description.

## Style
* Write in **third‑person** present tense.
* Begin each document with a one‑sentence **summary** of the feature or change.
* Use **markdown headings** (`#`, `##`, `###`) to structure content.
* Include **code snippets** in fenced blocks with language tags for clarity.
* When describing protocols or APIs, list message types and example JSON payloads.

## Updating Docs
1. Add a new markdown file for a new feature.
2. Append an entry to `doc/changelog.md` with a timestamp.
3. If the change affects existing documentation, update the relevant file and add a note in the changelog about the revision.

## Future LLM Contributors
* Follow the same structure and naming conventions.
* Do **not** modify existing documentation unless a clarification is required.
* Ensure any new files are referenced in the changelog.
