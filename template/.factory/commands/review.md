---
description: Start a comprehensive code review workflow
argument-hint: <branch-or-file-path>
---

Please perform a comprehensive code review of `$ARGUMENTS` following our standard review process:

## Review Checklist

1. **Correctness**
   - Logic errors and edge cases
   - Proper error handling
   - Algorithm correctness

2. **Security**
   - Input validation
   - Authentication/authorization
   - OWASP Top 10 violations
   - Secrets or credentials in code

3. **Testing**
   - Unit test coverage
   - Integration test coverage
   - Edge cases tested
   - Test quality and clarity

4. **Performance**
   - Algorithm efficiency
   - Database query optimization
   - Memory usage
   - Caching opportunities

5. **Maintainability**
   - Code clarity
   - Documentation
   - Naming conventions
   - Code duplication

## Review Process

You may want to:
- Spawn the `code-reviewer` droid for detailed analysis
- Spawn the `security-auditor` droid for security checks
- Spawn the `test-coordinator` droid to run tests and check coverage

## Output Format

Provide:
- **Summary:** One-line assessment
- **Critical Issues:** Must-fix items
- **Recommendations:** Nice-to-have improvements
- **Testing Gaps:** Missing test coverage
- **Approval Status:** LGTM / Needs Changes
