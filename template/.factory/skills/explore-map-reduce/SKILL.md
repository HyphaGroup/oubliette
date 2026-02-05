---
name: explore-map-reduce
description: Systematically explore and analyze large contexts using parallel partition processing
---

# Explore-Map-Reduce Skill

Automatically apply the explore-map-reduce pattern when facing large contexts that exceed single-session capabilities.

## When to Use

This skill activates when:
- Analyzing >50 files or >30k tokens of context
- User requests analysis of "large codebase" or "many files"
- Task involves systematic processing of independent units
- Context window would be overwhelmed by direct loading

## Pattern Overview

```
1. EXPLORE: Survey context → identify partitions
2. MAP: Process each partition in parallel
3. REDUCE: Aggregate results into final output
```

## Execution Steps

### Step 1: Exploration Phase

Spawn **context-explorer** droid:
```
Objective: Survey the large context
Tasks:
- Understand structure and organization
- Identify natural partition boundaries  
- Estimate scope and complexity
- Recommend partition strategy

Output: Exploration report with partition recommendations
```

Wait for exploration report before proceeding.

### Step 2: Map Phase

Based on exploration recommendations, spawn **partition-mapper** droids in parallel:

```
For each partition P:
  Spawn partition-mapper:
    - partition_id: P
    - scope: [files/data in partition P]
    - objective: [what to analyze]
    - output_schema: [consistent structure]
    - output_file: .rlm-context/gogol_<id>_partition_<P>.json
```

**Key principles:**
- Partitions should be **independent** (no shared mutable state)
- All mappers use the **same output schema** (enables aggregation)
- Each mapper writes to **`.rlm-context/`** for collection
- Partitions can run **in parallel** (spawn all at once)

### Step 3: Reduce Phase

After all partition-mappers complete, spawn **result-aggregator** droid:

```
Objective: Synthesize partition results
Tasks:
- Read all partition JSON from .rlm-context/
- Verify all partitions accounted for
- Merge findings using appropriate strategy
- Identify cross-partition patterns
- Generate final comprehensive report

Output: Aggregated results addressing original objective
```

### Step 4: Report Results

Present the aggregated findings to the user or parent session.

## Output Schema Definition

Critical for successful aggregation: **All partition-mappers must use identical schema**.

Define schema based on analysis type:

### Code Analysis Schema
```json
{
  "partition_id": "string",
  "scope": { "files": [...], "description": "..." },
  "findings": {
    "functions": [{
      "name": "string",
      "file": "path",
      "complexity": "low|medium|high",
      "issues": [...]
    }],
    "classes": [...],
    "dependencies": [...],
    "test_coverage": "percentage"
  },
  "metrics": {
    "files_analyzed": 0,
    "lines_of_code": 0,
    "issues_found": 0
  }
}
```

### Document Analysis Schema
```json
{
  "partition_id": "string",
  "scope": { "documents": [...] },
  "findings": {
    "key_themes": [...],
    "entities_mentioned": [...],
    "facts_extracted": [...],
    "citations": [...]
  },
  "metadata": {
    "documents_analyzed": 0,
    "total_words": 0
  }
}
```

### Data Analysis Schema
```json
{
  "partition_id": "string",
  "scope": { "records": "range" },
  "findings": {
    "statistics": { "count": 0, "mean": 0, "median": 0 },
    "outliers": [...],
    "patterns": [...],
    "validation_errors": [...]
  }
}
```

## Example Usage

### Example 1: Large Codebase Security Audit

```
User: "Audit our 120-file Go backend for security issues"

Step 1: Spawn context-explorer
→ Report: 5 modules identified (auth, api, database, worker, utils)

Step 2: Spawn 5 partition-mappers (parallel)
→ Partition 1: auth/ (20 files)
→ Partition 2: api/ (35 files)
→ Partition 3: database/ (25 files)
→ Partition 4: worker/ (30 files)
→ Partition 5: utils/ (10 files)

Each outputs: .rlm-context/gogol_<id>_partition_<N>.json

Step 3: Spawn result-aggregator
→ Reads all 5 partition JSONs
→ Synthesizes: 12 critical issues, 45 warnings
→ Produces: Prioritized security report

Step 4: Present final report to user
```

### Example 2: Research Paper Summarization

```
User: "Summarize 50 ML research papers in docs/"

Step 1: Spawn context-explorer
→ Report: 50 papers, no natural grouping, recommend 10 partitions of 5 papers each

Step 2: Spawn 10 partition-mappers (parallel)
→ Each processes 5 papers
→ Extracts: key contributions, methods, results, citations

Output schema:
{
  "partition_id": "batch-1",
  "papers": [
    { "title": "...", "summary": "...", "key_insights": [...] }
  ]
}

Step 3: Spawn result-aggregator
→ Synthesizes: Common themes across papers
→ Identifies: Research trends, key authors, influential works
→ Produces: Meta-analysis report

Step 4: Present synthesized summary
```

## Best Practices

✅ **Do:**
- Let context-explorer guide partition strategy
- Define schema before spawning mappers
- Ensure partitions are independent
- Write all partition results to `.rlm-context/`
- Verify aggregator reads all partitions
- Check for partition failures

❌ **Don't:**
- Skip exploration phase (leads to poor partitions)
- Use inconsistent schemas across mappers
- Create overlapping partitions
- Forget to spawn the aggregator
- Ignore failed partitions in final results

## Performance Optimization

**Parallel Execution:**
- Spawn all partition-mappers at once (not sequentially)
- Each operates independently
- Reduces total wall-clock time significantly

**Token Efficiency:**
- Partitions should be sized to fit comfortably in single sessions
- Exploration prevents wasted token loading
- Aggregation is cheaper than full context loading

**Depth Management:**
- This pattern typically uses 2-3 recursion levels
- Explorer → Mappers → Aggregator = 2 levels deep
- Mappers may further decompose if needed (level 3)

## Error Handling

If a partition-mapper fails:
1. Aggregator should note the missing partition
2. Proceed with available partitions
3. Flag incomplete coverage in final report
4. Optionally retry failed partition

If aggregator fails:
1. Parent can read partition JSONs directly
2. Perform manual aggregation
3. Or retry with different aggregation strategy

## Integration with TodoWrite

Track progress:
```
[In Progress] Explore context
[Pending] Map partition 1
[Pending] Map partition 2
...
[Pending] Aggregate results
```

Update as each phase completes.

## Success Criteria

Successful explore-map-reduce achieves:
1. ✅ Complete coverage of original context
2. ✅ All partition results collected
3. ✅ Coherent aggregated output
4. ✅ Answers original objective
5. ✅ Token efficiency vs. direct loading
