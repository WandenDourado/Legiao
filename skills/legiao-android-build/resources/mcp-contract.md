# MCP Contract

This skill does not require a dedicated MCP server.

If MCP tools are available, use generic tools with these contracts.

## Filesystem

Input:

- Repository-relative file path.
- Read, write, or patch operation.
- New content or patch when editing.

Output:

- Success or failure.
- File metadata when available.
- Error message when unavailable.

Security:

- Do not read files outside the repository unless the user explicitly authorizes it.
- Do not read or print secret values.

## Shell

Input:

- Command.
- Working directory.
- Timeout.

Output:

- Exit code.
- Standard output.
- Standard error.
- Duration when available.

Security:

- Do not execute destructive cleanup commands without explicit confirmation.
- Do not assume administrative privileges.

## Git

Input:

- Repository root.
- Status or diff request.

Output:

- Changed files.
- Diff summary.

Security:

- Do not commit, push, reset, checkout, or clean unless explicitly requested.
