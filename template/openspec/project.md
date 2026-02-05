# Project Context

## Purpose

This project runs inside an Oubliette container - a sandboxed execution environment for AI agents. Use OpenSpec for any significant changes to the codebase.

## Tech Stack

[Fill in your project's tech stack]

## Project Conventions

### Code Style

[Describe your code style preferences]

### Architecture Patterns

[Document your architectural decisions]

### Testing Strategy

[Explain your testing approach]

### Git Workflow

[Describe your branching strategy and commit conventions]

## Domain Context

[Add domain-specific knowledge]

## Important Constraints

- This project runs in an isolated container environment
- External network access may be limited
- File system changes persist within the workspace

## External Dependencies

[Document key external services, APIs, or systems]

## OpenSpec Workflow

This project uses OpenSpec for spec-driven development:

1. **Plan Mode**: Use `/openspec-proposal` to create formal specifications
2. **Build Mode**: Use `/openspec-apply <change-id>` to implement approved specs
3. **Archive**: Use `/openspec-archive <change-id>` when complete

See `openspec/AGENTS.md` for detailed workflow instructions.
