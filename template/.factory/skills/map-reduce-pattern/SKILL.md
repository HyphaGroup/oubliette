---
name: map-reduce-pattern
description: Apply map-reduce pattern over large file or data sets with parallel processing
---

# Map-Reduce Pattern Skill

Apply the classic map-reduce pattern when you need to process many independent items systematically.

## When to Use

This skill activates when:
- Processing a list of independent items (files, records, documents)
- Each item can be analyzed independently
- Results need to be combined/aggregated
- Direct sequential processing would be too slow or token-intensive

## Pattern Overview

```
INPUT: List of N items
MAP: Process each item → individual result
REDUCE: Combine all results → final output
```

## Execution Steps

### Step 1: Identify Items

Determine what needs to be processed:
- Files in a directory
- Records in a dataset
- Documents in a collection
- Modules in a codebase

Use tools:
- `Glob` for file patterns
- `LS` for directory contents
- `Read` for data files with lists
- `Grep` for finding items matching criteria

### Step 2: Define Processing

Specify what to do with each item:
- **Analysis type**: What to extract/compute
- **Output format**: Consistent schema across all items
- **Dependencies**: Ensure items are truly independent

### Step 3: Partition Items

Group items into partitions for parallel processing:

**Strategies:**
- **Equal distribution**: N items / M mappers
- **By category**: Group similar items
- **By size**: Balance large and small items
- **Batch size**: Optimize for token usage

**Example:**
```
100 files total
→ 10 partitions of 10 files each
→ Spawn 10 partition-mappers in parallel
```

### Step 4: Map Phase

Spawn **partition-mapper** droid for each partition:

```json
{
  "partition_id": "partition-1",
  "items": ["item1", "item2", "item3", ...],
  "objective": "What to analyze per item",
  "output_schema": {
    // Define consistent structure
  },
  "output_file": ".rlm-context/gogol_<id>_partition_1.json"
}
```

All mappers run **in parallel** and write to `.rlm-context/`.

### Step 5: Reduce Phase

Spawn **result-aggregator** droid:

```
Tasks:
1. Read all partition results from .rlm-context/
2. Apply reduction strategy:
   - Union: Combine lists (deduplicate if needed)
   - Sum: Add up counts/metrics
   - Merge: Combine objects/maps
   - Aggregate: Compute statistics
   - Filter: Extract top N, threshold
3. Generate final unified result
```

### Step 6: Output

Present reduced results to user or parent session.

## Map Functions

Common mapping operations:

### File Analysis Map
```
For each file:
  → Read content
  → Extract: imports, functions, classes, TODOs
  → Analyze: complexity, issues, patterns
  → Output: Structured file metadata
```

### Document Processing Map
```
For each document:
  → Parse content
  → Extract: key points, entities, facts
  → Compute: word count, sentiment, topics
  → Output: Document summary
```

### Data Validation Map
```
For each record:
  → Validate schema
  → Check constraints
  → Identify anomalies
  → Output: Validation results
```

## Reduce Functions

Common reduction operations:

### Aggregation Reduce
```python
# Combine metrics across partitions
total_files = sum(p.files_analyzed for p in partitions)
all_issues = [i for p in partitions for i in p.issues]
avg_complexity = mean(p.avg_complexity for p in partitions)
```

### Collection Reduce
```python
# Union of findings
all_dependencies = set()
for p in partitions:
    all_dependencies.update(p.dependencies)

# Ranked list
top_issues = sorted(
    [i for p in partitions for i in p.issues],
    key=lambda x: x.severity,
    reverse=True
)[:10]
```

### Synthesis Reduce
```python
# Cross-partition analysis
common_patterns = find_patterns_across_partitions(partitions)
outliers = find_outliers(partitions)
recommendations = generate_recommendations(partitions)
```

## Example Usage

### Example 1: Test Coverage Analysis

```
Task: "Analyze test coverage across 80 test files"

Step 1: Identify items
→ Glob: tests/**/*.test.js
→ Found: 80 test files

Step 2: Define processing
→ Per file: count tests, identify coverage gaps, check assertions
→ Schema: { file, test_count, coverage_gaps, assertions }

Step 3: Partition
→ 8 partitions × 10 files each

Step 4: Map
→ Spawn 8 partition-mappers (parallel)
→ Each processes 10 test files
→ Outputs to .rlm-context/

Step 5: Reduce
→ Spawn result-aggregator
→ Total tests: SUM
→ Coverage gaps: UNION + PRIORITIZE
→ Statistics: COMPUTE averages

Step 6: Output
→ Report: "640 tests across 80 files, 15 coverage gaps identified"
```

### Example 2: Dependency Audit

```
Task: "List all npm dependencies across 50 package.json files"

Step 1: Identify items
→ Glob: **/package.json
→ Found: 50 files

Step 2: Define processing
→ Per file: extract dependencies, devDependencies, versions
→ Schema: { file, dependencies: { name, version, type } }

Step 3: Partition
→ 5 partitions × 10 files each

Step 4: Map
→ Spawn 5 partition-mappers
→ Each reads 10 package.json files
→ Extracts dependency data

Step 5: Reduce
→ Aggregator creates dependency graph
→ Identifies: version conflicts, duplicates, vulnerabilities
→ Generates: consolidated dependency list

Step 6: Output
→ "215 unique dependencies, 8 version conflicts, 3 outdated packages"
```

## Optimization Strategies

### Partition Sizing

**Too small:**
- More overhead from spawning agents
- Aggregation complexity increases
- May not be worth parallelization

**Too large:**
- Risk hitting token limits
- Longer individual processing time
- Less parallelization benefit

**Sweet spot:**
- 5-15 items per partition
- 5-20 partitions total
- Balance parallelism with overhead

### Output Schema Design

Keep schemas:
- **Consistent**: Same structure across all mappers
- **Flat**: Avoid deep nesting (easier to aggregate)
- **Quantifiable**: Include counts and metrics
- **Deduplicated**: Each mapper removes internal duplicates

### Token Management

- Mappers should output summaries, not full content
- Include only actionable data in results
- Aggregator should synthesize, not concatenate
- Use `.rlm-context/` for result storage (atomic writes)

## Error Handling

### Mapper Failure
```
If partition-mapper fails:
1. Log failure in .rlm-context/
2. Other mappers continue
3. Aggregator notes missing partition
4. Partial results still useful
5. Optionally retry failed partition
```

### Aggregator Failure
```
If result-aggregator fails:
1. Partition results still in .rlm-context/
2. Parent can read directly
3. Retry with simpler aggregation
4. Or manual inspection
```

## Best Practices

✅ **Do:**
- Verify items are independent before mapping
- Use consistent schemas across mappers
- Write all results to `.rlm-context/`
- Spawn mappers in parallel (not sequential)
- Include metadata (counts, durations) in outputs
- Handle partial results gracefully

❌ **Don't:**
- Map over dependent items (will fail)
- Use different schemas per mapper (breaks reduce)
- Read all items first (defeats purpose)
- Spawn mappers sequentially (loses parallelism)
- Forget to aggregate (mappers alone insufficient)
- Ignore mapper failures completely

## Integration

Works well with:
- **context-explorer**: Can identify items to map over
- **recursive-planner**: Can decide when to apply map-reduce
- **recursive-task-planning**: Use as decomposition strategy

## Success Criteria

Successful map-reduce achieves:
1. ✅ All items processed
2. ✅ Results properly aggregated
3. ✅ Significant speedup vs sequential
4. ✅ Token efficiency gains
5. ✅ Actionable final output
