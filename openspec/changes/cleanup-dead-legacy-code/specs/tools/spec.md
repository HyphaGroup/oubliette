## MODIFIED Requirements

### Requirement: Tool Registration
The system SHALL register tools using `ToolDef` with `Name`, `Description`, `Target` (ToolTarget), `Access` (ToolAccess), and `InputSchema` fields. The legacy `Scope` field SHALL NOT be used.

#### Scenario: Tool registered with Target/Access
- **WHEN** a tool is registered with `Target: TargetProject` and `Access: AccessWrite`
- **THEN** it is accessible to admin tokens and project-scoped write tokens for the matching project

#### Scenario: Legacy scope field rejected
- **WHEN** a tool is registered without Target and Access fields
- **THEN** it SHALL be denied access regardless of token scope

## REMOVED Requirements

### Requirement: Legacy Scope Fallback
**Reason**: All tools now use Target/Access model. The legacy `Scope` string field ("admin", "write", "read") and `isToolAllowedForTokenScope()` fallback are dead code.
**Migration**: None required — all registered tools already use Target/Access.
