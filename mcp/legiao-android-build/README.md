# legiao-android-build MCP Notes

`legiao-android-build` does not require a dedicated MCP server.

Agents may use generic MCP tools if their host provides them:

- Filesystem MCP for repository file reads and patches.
- Shell/terminal MCP for build commands.
- Git MCP for status and diff inspection.

Do not create or depend on proprietary MCP integrations for this skill. The authoritative skill instructions are:

- `AGENTS.md`
- `skills/legiao-android-build/SKILL.md`
- `skills/legiao-android-build/resources/mcp-contract.md`
