---
name: recursive-task-planning
description: Automatically decompose complex tasks into manageable subtasks and coordinate execution
---

# Recursive Task Planning

When faced with a complex task that cannot be completed in a single session, you should break it down into subtasks and coordinate their execution through recursive spawning.

## When to Use This Skill

Apply recursive task planning when:
- Task complexity exceeds single-session scope
- Multiple independent subtasks can be parallelized
- Task requires specialized expertise areas
- Token/time limits make sequential execution impractical

## Task Decomposition Process

### 1. Analyze Complexity
- Identify major components of the task
- Map dependencies between components
- Estimate effort per component
- Assess parallelization opportunities

### 2. Define Subtasks
Each subtask should be:
- **Independent:** Minimal shared mutable state
- **Focused:** Single responsibility, clear scope
- **Testable:** Concrete success criteria
- **Bounded:** Completable in one session

### 3. Create Task Hierarchy
Use `TodoWrite` to structure tasks:
```
[High Priority] Main Goal
├─ [In Progress] Subtask 1 (spawned as gogol_001)
├─ [Pending] Subtask 2 (blocked by Subtask 1)
└─ [Pending] Subtask 3 (can run in parallel)
```

### 4. Spawn Child Sessions
For each independent subtask:
```
gogol_spawn({
  project_name: "current-project",
  prompt: "Clear, focused objective for child",
  workspace_id: "current-workspace",  // Inherit workspace
  // Specify success criteria and output format
})
```

### 5. Monitor Progress
- Track child session status via `gogol_get_session`
- Update TodoWrite as children complete
- Check for errors or blockers

### 6. Aggregate Results
Children write results to `.rlm-context/<session_id>_<descriptor>.json`:
```json
{
  "session_id": "gogol_child_001",
  "task": "Implement authentication module",
  "status": "completed",
  "artifacts": [
    "src/auth/login.go",
    "tests/auth_test.go"
  ],
  "summary": "Auth module completed with JWT support"
}
```

Parent reads and synthesizes:
```bash
# In parent session
ls /workspace/.rlm-context/
cat /workspace/.rlm-context/gogol_child_001_auth.json
```

### 7. Synthesize Final Result
- Combine child outputs
- Verify all success criteria met
- Report to parent (if recursive) or user
- Write own result to `.rlm-context/` if needed

## Best Practices

### Subtask Prompts
Write clear, specific prompts for children:

✅ **Good:**
```
Implement user authentication module:
1. JWT-based auth with RS256
2. Login/logout endpoints
3. Password hashing with bcrypt
4. Unit tests with 80% coverage

Output: Write summary to .rlm-context/gogol_<id>_auth.json
```

❌ **Bad:**
```
Do the auth stuff
```

### Dependency Management
- Spawn dependent tasks **sequentially**
- Spawn independent tasks **in parallel**
- Use TodoWrite to track dependencies

### Error Handling
If a child fails:
1. Inspect error in `.rlm-context/gogol_<id>_error.json`
2. Decide: retry, adjust approach, or escalate
3. Update TodoWrite with blocker status

### Resource Limits
Respect project recursion limits:
- Check `max_recursion_depth` before spawning
- Monitor `max_agents_per_session`
- Track `max_cost_usd` budget

## Example: Full-Stack Feature

**Task:** Implement user profile feature

**Decomposition:**
1. **Backend API** (child 1)
   - Database schema
   - CRUD endpoints
   - Validation logic

2. **Frontend UI** (child 2, parallel with backend)
   - Profile form component
   - API integration
   - State management

3. **Integration Tests** (child 3, after 1 & 2)
   - E2E test scenarios
   - API contract tests

**Execution:**
```
Parent spawns:
  - gogol_backend (immediate)
  - gogol_frontend (immediate, parallel)
  
Parent waits, then spawns:
  - gogol_integration (after both complete)
  
Parent aggregates:
  - Read .rlm-context/gogol_backend_*.json
  - Read .rlm-context/gogol_frontend_*.json
  - Read .rlm-context/gogol_integration_*.json
  
Parent synthesizes:
  - Final report with all artifacts
  - Verification checklist
```

## Result Format

Always write results to `.rlm-context/`:
```json
{
  "session_id": "<your-session-id>",
  "parent_session_id": "<parent-if-any>",
  "task_summary": "Brief description",
  "status": "completed | failed | blocked",
  "artifacts_created": ["file1", "file2"],
  "tests_added": ["test1", "test2"],
  "child_sessions": ["gogol_001", "gogol_002"],
  "recommendations": ["Next step 1", "Next step 2"],
  "metadata": {
    "duration_seconds": 120,
    "tokens_used": 5000
  }
}
```

## Anti-Patterns to Avoid

❌ **Over-decomposition:** Too many trivial subtasks  
❌ **Under-decomposition:** Subtasks still too complex  
❌ **Tight coupling:** Subtasks with heavy interdependencies  
❌ **Missing coordination:** No clear result aggregation plan  
❌ **Unbounded recursion:** No depth limit or stopping condition  

## Success Criteria

Effective recursive task planning achieves:
- ✅ Task completed within resource limits
- ✅ All subtasks have clear results
- ✅ Results properly aggregated
- ✅ No redundant or wasted work
- ✅ Proper error handling and recovery
