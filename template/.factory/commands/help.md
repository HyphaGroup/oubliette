---
description: Show available custom droids, skills, and commands
---

# Oubliette Help

Here's what you can do with this system:

## Custom Droids (Subagents)

You can spawn specialized subagents for focused tasks:

- **code-reviewer** - Comprehensive code review (correctness, security, maintainability)
- **security-auditor** - Security-focused analysis (OWASP Top 10, CVEs)
- **test-coordinator** - Test execution and coverage analysis

To spawn a droid:
```
"Please spawn the code-reviewer subagent to review my changes"
```

Or I can invoke them directly using the Task tool.

## Skills (Automatic)

These workflows are automatically applied when relevant:

- **recursive-task-planning** - Breaks complex tasks into subtasks with child gogols

Skills are invoked automatically based on context - you don't need to request them explicitly.

## Custom Commands

Quick shortcuts for common operations:

- **/review <branch>** - Start comprehensive code review workflow
- **/help** - Show this help (you're reading it now!)

## MCP Tools

Standard Oubliette tools available:

- **gogol_spawn** - Create a new child gogol
- **gogol_continue_session** - Continue an existing session
- **gogol_get_session** - Get session details
- **gogol_list_sessions** - List all sessions in project
- **gogol_end_session** - End a session
- **gogol_get_streaming_output** - View streaming events
- **gogol_get_project** - Get project configuration
- **gogol_list_projects** - List all projects

## Configuration Levels

Configuration is layered:

1. **Global** - Base droids/skills for all projects (read-only)
2. **Project** - Project-specific customizations (shared with team)
3. **Workspace** - Workspace-specific overrides (your experiments)

## Need More Help?

- Read `/workspace/.rlm-context/` for results from child sessions
- Check project documentation in `/workspace/docs/`
- Ask me to explain any concept in more detail!
