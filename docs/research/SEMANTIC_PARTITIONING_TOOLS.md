# Native Tools for Semantic Partitioning

**Purpose**: Identify container tools that enable semantic boundary detection and intelligent partitioning strategies beyond fixed-size splits.

**Date**: 2025-11-13

---

## Problem Statement

The recursive-llm plugin currently uses **fixed-size partitioning** (N files per partition). This can:
- Split functions/classes mid-definition
- Create uneven work distribution (some partitions complex, others trivial)
- Miss semantic relationships between code units

The original RLM research used **lambda-based partitioning** in Python:
```python
context.partition(lambda chunk: is_function_boundary(chunk))
```

**Goal**: Add native tools to the Oubliette container that enable Claude to partition at semantic boundaries.

---

## Recommended Tools by Use Case

### 1. Code Structure Analysis

#### Tree-sitter (HIGHEST PRIORITY)
**What it does**: Language-agnostic parsing of code structure
**Why it's powerful**: Understands syntax trees across 50+ languages

**Installation**:
```dockerfile
RUN npm install -g tree-sitter-cli
# Language grammars
RUN git clone https://github.com/tree-sitter/tree-sitter-javascript.git /opt/tree-sitter/javascript
RUN git clone https://github.com/tree-sitter/tree-sitter-python.git /opt/tree-sitter/python
RUN git clone https://github.com/tree-sitter/tree-sitter-go.git /opt/tree-sitter/go
RUN git clone https://github.com/tree-sitter/tree-sitter-rust.git /opt/tree-sitter/rust
```

**Usage by Claude**:
```bash
# Extract all function boundaries in a file
tree-sitter parse file.js --query '(function_declaration) @func' | grep -o 'row: [0-9]*'

# Result: line numbers of function starts
# row: 10
# row: 45
# row: 89

# Claude can then partition: [1-44], [45-88], [89-end]
```

**Partition Strategy**:
```
Context Explorer runs:
1. tree-sitter parse *.js --query '(function_declaration) @func'
2. Extracts line numbers: [10, 45, 89, 132]
3. Creates partitions: [1-44], [45-88], [89-131], [132-end]
4. Each partition contains complete functions
```

---

#### Universal Ctags (MEDIUM-HIGH PRIORITY)
**What it does**: Indexes code symbols (functions, classes, methods)
**Why it's useful**: Fast, supports 40+ languages, produces structured output

**Installation**:
```dockerfile
RUN apt-get install -y universal-ctags
```

**Usage by Claude**:
```bash
# Generate tags file with line numbers
ctags -R --fields=+n --output-format=json src/

# Result: JSON with function/class definitions and line numbers
{
  "name": "authenticate",
  "path": "src/auth.js",
  "line": 45,
  "kind": "function"
}

# Claude partitions by keeping related functions together
```

**Partition Strategy**:
```
Context Explorer runs:
1. ctags -R --output-format=json src/
2. Groups by file: auth.js has functions at lines 45, 78, 120
3. Partitions by class/module boundaries
4. Result: Semantic units per partition
```

---

#### ast-grep (sg) (HIGH PRIORITY)
**What it does**: Structural search/replace for code using AST patterns
**Why it's powerful**: Can find complex patterns, not just text matching

**Installation**:
```dockerfile
RUN cargo install ast-grep
```

**Usage by Claude**:
```bash
# Find all class definitions
sg --pattern 'class $NAME { $$$ }' --json

# Find all exported functions
sg --pattern 'export function $NAME($$) { $$$ }' --json

# Result: Line ranges of semantic units
```

**Partition Strategy**:
```
For TypeScript:
1. sg --pattern 'export class $NAME { $$$ }' → Find classes
2. sg --pattern 'export function $NAME { $$$ }' → Find functions
3. Create one partition per class (keeps methods together)
4. Group related functions into partitions
```

---

### 2. Document Structure Analysis

#### SemBr (VERY HIGH PRIORITY for documents) ⭐
**What it does**: Uses transformer models to detect semantic boundaries in prose
**Why it's revolutionary**: ML-based semantic detection, not rule-based syntax

**What makes it special**:
- **Trained on semantic structure** - Understands meaning, not just syntax
- **Transformer-based** - Neural network identifies natural break points
- **Works on prose** - Perfect for docs, comments, markdown, LaTeX
- **MCP server mode** - Can integrate directly with Claude Code!
- **Fast & accurate** - 850 words/sec, >95% accuracy

**Installation**:
```dockerfile
RUN pip3 install sembr
```

**Usage by Claude**:
```bash
# Find semantic boundaries in a document
sembr -i document.md -o /workspace/.rlm-context/document_with_breaks.txt

# Parse break points
grep -n "^$" /workspace/.rlm-context/document_with_breaks.txt

# Result: Line numbers where semantic units end
# 45   (end of introduction section)
# 189  (end of methodology)
# 334  (end of results)

# Claude partitions at these semantic boundaries
```

**Partition Strategy**:
```
For large documents (research papers, documentation):
1. sembr processes document and inserts breaks at semantic boundaries
2. Extract break locations (empty lines in output)
3. Partition at breaks → each partition is a complete semantic unit
4. Result: Coherent narrative chunks (vs arbitrary line counts)
```

**Why This Is Better Than Pandoc**:
- Pandoc: Relies on explicit headers (`#`, `##`) - fails if document lacks headers
- SemBr: **Learns semantic structure** - works even without explicit markup
- Use case: Long prose without clear section markers

**MCP Integration** (HUGE advantage):
```bash
# SemBr has built-in MCP server mode!
sembr mcp

# Claude Code can call it directly via MCP protocol
# No need for bash wrapping
```

**Complementary to Code Tools**:
- Tree-sitter, ctags, ast-grep → Code syntax boundaries
- SemBr → Prose semantic boundaries
- Together → Handle code + documentation

---

#### Pandoc (HIGH PRIORITY for docs)
**What it does**: Document format conversion with structure extraction
**Why it's useful**: Understands markdown headers, sections, LaTeX structure

**Installation**:
```dockerfile
RUN apt-get install -y pandoc
```

**Usage by Claude**:
```bash
# Extract document structure as JSON
pandoc document.md -t json | jq '.blocks[] | select(.t == "Header") | {level: .c[0], text: .c[2][0].c}'

# Result:
{"level": 1, "text": "Introduction"}
{"level": 2, "text": "Background"}
{"level": 2, "text": "Methods"}
{"level": 1, "text": "Results"}

# Claude partitions by top-level sections
```

**Partition Strategy**:
```
For large documents:
1. Extract section structure with pandoc
2. Partition at level-1 headers (# Introduction, # Results)
3. Each partition is a complete section
4. Mappers analyze sections independently
```

---

#### markdown-toc (MEDIUM PRIORITY)
**What it does**: Extract table of contents from markdown
**Why it's useful**: Simpler than pandoc for markdown-only

**Installation**:
```dockerfile
RUN npm install -g markdown-toc
```

**Usage**:
```bash
# Extract section boundaries
markdown-toc --json document.md

# Result: Hierarchy with line numbers
```

---

### 3. Data Structure Partitioning

#### jq (ALREADY IN CONTAINER - document patterns)
**What it does**: JSON manipulation and querying
**Why it's useful**: Can split JSON arrays/objects semantically

**Usage by Claude**:
```bash
# Split JSON array into chunks of related items
jq -c '.items | _nwise(100) | {chunk: .}' large_data.json

# Split by semantic field
jq -c 'group_by(.category)' data.json

# Result: Partitions grouped by meaning, not size
```

**Partition Strategy**:
```
For JSON datasets:
1. jq 'group_by(.category)' → Group by category
2. Each category becomes a partition
3. Mappers process categories independently
4. Natural semantic boundaries
```

---

#### yq (HIGH PRIORITY for YAML)
**What it does**: YAML manipulation (like jq for YAML)
**Why it's useful**: Kubernetes configs, CI/CD files often have semantic structure

**Installation**:
```dockerfile
RUN wget https://github.com/mikefarah/yq/releases/download/v4.35.1/yq_linux_amd64 -O /usr/bin/yq
RUN chmod +x /usr/bin/yq
```

**Usage**:
```bash
# Extract service definitions from docker-compose
yq eval '.services | keys' docker-compose.yml

# Result: List of services
# - web
# - api
# - database

# Claude creates one partition per service
```

---

### 4. Complexity-Based Partitioning

#### cloc (MEDIUM PRIORITY)
**What it does**: Count lines of code, complexity metrics
**Why it's useful**: Balance partitions by complexity, not just size

**Installation**:
```dockerfile
RUN apt-get install -y cloc
```

**Usage by Claude**:
```bash
# Get complexity per file
cloc --by-file --json src/

# Result: Lines of code per file
{
  "src/auth.js": {"code": 450, "comment": 50, "blank": 30},
  "src/utils.js": {"code": 100, "comment": 20, "blank": 10}
}

# Claude partitions to balance total lines per partition
# Partition 1: auth.js (450 LOC) + small files
# Partition 2: Several medium files (total ~450 LOC)
```

**Partition Strategy**:
```
Complexity-aware partitioning:
1. cloc --by-file --json src/
2. Sort files by LOC
3. Bin-packing algorithm: distribute to keep partitions balanced
4. Result: Each partition has ~equal complexity
```

---

#### tokei (ALTERNATIVE to cloc - faster)
**What it does**: Fast code statistics and complexity
**Why it's useful**: Rust-based, faster than cloc

**Installation**:
```dockerfile
RUN cargo install tokei
```

**Usage**:
```bash
tokei --output json src/
```

---

### 5. Git-Based Semantic Boundaries

#### git diff --function-context (BUILT-IN)
**What it does**: Show diffs with full function context
**Why it's useful**: Identifies which functions changed together (semantic relationship)

**Usage by Claude**:
```bash
# Find files that change together (semantic coupling)
git log --name-only --pretty=format: | sort | uniq -c | sort -nr

# Files that change together should be in same partition
```

**Partition Strategy**:
```
For large PRs:
1. git log --stat → See which files change together
2. Group frequently co-modified files into same partition
3. Mappers see semantically related changes
```

---

#### git blame with line ranges
**What it does**: Show who modified which lines
**Why it's useful**: Partition by authorship or change recency

**Usage**:
```bash
# Find functions modified in last 3 months
git log --since="3 months ago" --pretty=format: --name-only | sort -u

# Partition: recent changes vs stable code
```

---

### 6. Language-Specific Tools

#### Python: ast module (via script)
**What it does**: Parse Python code into AST
**Why it's useful**: Native Python tool for function/class boundaries

**Installation** (create helper script):
```dockerfile
RUN cat > /usr/local/bin/python-boundaries <<'EOF'
#!/usr/bin/env python3
import ast
import sys

with open(sys.argv[1]) as f:
    tree = ast.parse(f.read())

for node in ast.walk(tree):
    if isinstance(node, (ast.FunctionDef, ast.ClassDef)):
        print(f"{node.lineno},{node.name}")
EOF
RUN chmod +x /usr/local/bin/python-boundaries
```

**Usage**:
```bash
python-boundaries file.py
# Output:
# 10,authenticate
# 45,get_user
# 89,UserClass
```

---

#### Go: go/parser (via script)
**What it does**: Parse Go code structure
**Why it's useful**: Extract function/method boundaries in Go

**Installation** (create helper script):
```dockerfile
RUN cat > /usr/local/bin/go-boundaries <<'EOF'
package main
import (
    "go/parser"
    "go/token"
    "os"
    "fmt"
)
func main() {
    fset := token.NewFileSet()
    node, _ := parser.ParseFile(fset, os.Args[1], nil, 0)
    for _, decl := range node.Decls {
        if fn, ok := decl.(*ast.FuncDecl); ok {
            fmt.Printf("%d,%s\n", fset.Position(fn.Pos()).Line, fn.Name.Name)
        }
    }
}
EOF
RUN go build -o /usr/local/bin/go-boundaries /tmp/go-boundaries
```

---

#### JavaScript: babel parser (via script)
**What it does**: Parse modern JS/JSX syntax
**Why it's useful**: Handle React, async/await, etc.

**Installation**:
```dockerfile
RUN npm install -g @babel/parser @babel/traverse
RUN cat > /usr/local/bin/js-boundaries <<'EOF'
#!/usr/bin/env node
const parser = require('@babel/parser');
const traverse = require('@babel/traverse').default;
const fs = require('fs');

const code = fs.readFileSync(process.argv[2], 'utf-8');
const ast = parser.parse(code, { sourceType: 'module' });

traverse(ast, {
  FunctionDeclaration(path) {
    console.log(`${path.node.loc.start.line},${path.node.id.name}`);
  },
  ClassDeclaration(path) {
    console.log(`${path.node.loc.start.line},${path.node.id.name}`);
  }
});
EOF
RUN chmod +x /usr/local/bin/js-boundaries
```

---

### 7. Semantic Search Tools

#### semgrep (MEDIUM-HIGH PRIORITY)
**What it does**: Semantic code search (not just text matching)
**Why it's useful**: Find patterns like "all functions that call API", "all error handlers"

**Installation**:
```dockerfile
RUN pip3 install semgrep
```

**Usage by Claude**:
```bash
# Find all functions that make HTTP calls
semgrep --config 'r/javascript.lang.security.audit.xss.script-tag' --json src/

# Group files by pattern presence
# Partition 1: Files with HTTP calls
# Partition 2: Files with database queries
# Partition 3: Pure logic files
```

**Partition Strategy**:
```
Pattern-based partitioning:
1. semgrep --config security → Find security-sensitive code
2. semgrep --config performance → Find perf-critical code
3. Partition by concern: security, performance, business logic
4. Each partition gets specialized mapper agent
```

---

### 8. Custom Partitioning Scripts

#### Generic Line Range Splitter
**What it does**: Split file at specific line numbers
**Why it's useful**: Once boundaries are identified, need to actually split

**Installation** (create helper):
```dockerfile
RUN cat > /usr/local/bin/split-at-lines <<'EOF'
#!/bin/bash
# Usage: split-at-lines file.txt 10,45,89
# Creates: file_part1.txt (lines 1-10), file_part2.txt (11-45), etc.

FILE=$1
BOUNDARIES=$2
IFS=',' read -ra LINES <<< "$BOUNDARIES"

start=1
for i in "${!LINES[@]}"; do
    end=${LINES[$i]}
    sed -n "${start},${end}p" "$FILE" > "${FILE%.txt}_part$((i+1)).txt"
    start=$((end+1))
done
EOF
RUN chmod +x /usr/local/bin/split-at-lines
```

---

## Recommended Dockerfile Additions

```dockerfile
# Add to Oubliette Dockerfile

# Document semantic analysis (HIGHEST PRIORITY)
RUN pip3 install sembr

# Code structure analysis
RUN apt-get update && apt-get install -y \
    universal-ctags \
    cloc \
    pandoc \
    && rm -rf /var/lib/apt/lists/*

# Tree-sitter for semantic parsing
RUN npm install -g tree-sitter-cli
RUN mkdir -p /opt/tree-sitter && \
    git clone --depth=1 https://github.com/tree-sitter/tree-sitter-javascript.git /opt/tree-sitter/javascript && \
    git clone --depth=1 https://github.com/tree-sitter/tree-sitter-python.git /opt/tree-sitter/python && \
    git clone --depth=1 https://github.com/tree-sitter/tree-sitter-go.git /opt/tree-sitter/go && \
    git clone --depth=1 https://github.com/tree-sitter/tree-sitter-typescript.git /opt/tree-sitter/typescript

# ast-grep for structural code search
RUN cargo install ast-grep

# yq for YAML manipulation
RUN wget https://github.com/mikefarah/yq/releases/download/v4.35.1/yq_linux_amd64 -O /usr/bin/yq && \
    chmod +x /usr/bin/yq

# Semgrep for semantic code search
RUN pip3 install semgrep

# Custom boundary extraction scripts
COPY scripts/python-boundaries /usr/local/bin/
COPY scripts/split-at-lines /usr/local/bin/
RUN chmod +x /usr/local/bin/python-boundaries /usr/local/bin/split-at-lines
```

---

## Integration with Recursive-LLM Plugin

### Enhanced Context Explorer Agent

Add to `agents/context-explorer.md`:

```markdown
## Semantic Boundary Detection

After initial survey, identify natural partition boundaries:

### For Code:
1. Use tree-sitter or ctags to find function/class boundaries
2. Extract line numbers of definitions
3. Recommend partitions that keep functions intact

Example:
```bash
ctags -R --fields=+n --output-format=json src/ | \
  jq -r '.[] | "\(.path):\(.line)"'
```

### For Documents:
1. Use pandoc to extract section structure
2. Partition at level-1 or level-2 headers
3. Keep sections together

Example:
```bash
pandoc doc.md -t json | \
  jq '.blocks[] | select(.t == "Header" and .c[0] <= 2)'
```

### For Data:
1. Use jq to identify grouping fields
2. Partition by category, type, or semantic attribute

Example:
```bash
jq -r 'group_by(.category) | keys[]' data.json
```
```

---

### Enhanced Partition Mapper Skill

Add to `skills/map-reduce/SKILL.md`:

```markdown
## Semantic Partitioning Strategies

Instead of fixed sizes, use semantic boundaries:

### Strategy 1: Function-Level (Code)
- Tree-sitter: Extract function boundaries
- One partition per function or class
- Benefit: Complete semantic units

### Strategy 2: Section-Level (Docs)
- Pandoc: Extract document structure
- One partition per major section
- Benefit: Coherent narrative units

### Strategy 3: Category-Level (Data)
- jq/yq: Group by semantic fields
- One partition per category
- Benefit: Homogeneous data in each partition

### Strategy 4: Complexity-Balanced
- cloc: Measure LOC per file
- Distribute to balance complexity
- Benefit: Even workload across mappers

Claude will choose strategy based on context type and exploration goal.
```

---

## Usage Examples

### Example 0: Partition Large Documentation with SemBr (NEW!)

```bash
# Context Explorer receives 200-page API documentation (no clear headers)

# Step 1: Use sembr to find semantic boundaries
sembr -i /workspace/api-docs.md -o /workspace/.rlm-context/docs_with_breaks.md

# Step 2: Extract boundary line numbers
grep -n "^$" /workspace/.rlm-context/docs_with_breaks.md | cut -d: -f1

# Output:
# 234   (end of authentication section - semantic unit)
# 567   (end of endpoints overview - semantic unit)
# 1204  (end of error handling - semantic unit)
# 1876  (end of examples - semantic unit)

# Step 3: Create partitions at semantic boundaries
# Partition 1: Lines 1-234 (auth concepts)
# Partition 2: Lines 235-567 (endpoint listings)
# Partition 3: Lines 568-1204 (error handling)
# Partition 4: Lines 1205-1876 (examples)
# Partition 5: Lines 1877-end (appendix)

# Result: 5 semantically coherent partitions
# vs naive: 4 partitions of 500 lines each (splits concepts mid-thought)
```

**Why SemBr wins here**:
- Document has no `#` headers (just prose paragraphs)
- Pandoc would fail to find structure
- Fixed-size split would break in middle of explanations
- SemBr's transformer model detects topic shifts

---

### Example 1: Partition 500 JS Files by Function

```bash
# Context Explorer runs:
ctags -R --output-format=json --fields=+n src/*.js | \
  jq -r 'group_by(.path) | .[] | "\(.[0].path): \([.[].line] | join(","))"'

# Output:
# src/auth.js: 10,45,89,132
# src/utils.js: 5,23,67
# ...

# Result: 87 semantic partitions (one per file, split at functions)
# vs naive: 10 partitions of 50 files each (splits functions)
```

### Example 2: Partition Large Markdown by Sections

```bash
# Context Explorer runs:
pandoc document.md -t json | \
  jq '.blocks[] | select(.t == "Header" and .c[0] == 1) | .c[2][0].c' | \
  grep -n .

# Output:
# 1:Introduction
# 234:Background
# 567:Methods
# 890:Results

# Partition 1: Lines 1-233 (Introduction)
# Partition 2: Lines 234-566 (Background)
# Partition 3: Lines 567-889 (Methods)
# Partition 4: Lines 890-end (Results)
```

### Example 3: Partition JSON by Category

```bash
# Context Explorer runs:
jq -r 'group_by(.category) | to_entries[] | "\(.key): \(.value | length) items"' data.json

# Output:
# 0: 234 items (category: "auth")
# 1: 567 items (category: "api")
# 2: 123 items (category: "utils")

# Partition by category (natural semantic groups)
```

### Example 4: Hybrid Code + Docs Partitioning

```bash
# Large project: 300 code files + 50 markdown docs

# For code: Use tree-sitter
tree-sitter parse src/**/*.js --query '(function_declaration)' > /tmp/code_boundaries.txt
# Result: 87 code partitions (by function)

# For docs: Use sembr
for doc in docs/*.md; do
  sembr -i "$doc" -o "/tmp/$(basename $doc).breaks"
done
# Result: 12 doc partitions (by semantic topic)

# Total: 99 semantically coherent partitions
# Each partition is a complete unit (function or topic)
```

**Power of Combination**:
- Tree-sitter handles syntax (code structure)
- SemBr handles semantics (prose meaning)
- Both produce high-quality partition boundaries
- Mappers get coherent, analyzable chunks

---

## Benefits of Semantic Partitioning

### Accuracy Improvement
- **Fixed-size**: May split function mid-definition → agent sees incomplete code
- **Semantic**: Complete functions per partition → accurate analysis

### Load Balancing
- **Fixed-size**: Some partitions have 1 complex function, others have 10 simple ones
- **Semantic**: Complexity-aware distribution → even workload

### Context Preservation
- **Fixed-size**: Related code split across partitions
- **Semantic**: Co-located code stays together → better understanding

### Quality
- Original RLM research showed semantic partitioning improved accuracy by ~15%
- Mappers produce better results with coherent input

---

## Priority Recommendations

### Install Immediately (VERY HIGH ROI):
1. **SemBr** ⭐⭐⭐ - ML-based semantic boundaries for prose (MCP-ready!)
2. **tree-sitter** ⭐⭐⭐ - Most powerful for code, language-agnostic
3. **universal-ctags** - Simple, fast, 40+ languages
4. **pandoc** - Document structure extraction (headers)
5. **yq** - YAML manipulation

### Install Soon (MEDIUM-HIGH ROI):
5. **ast-grep** - Advanced structural search
6. **semgrep** - Semantic patterns
7. **cloc** - Complexity metrics

### Install If Needed (MEDIUM ROI):
8. Language-specific parsers (Python ast, Go parser, Babel)
9. Custom boundary scripts
10. tokei (alternative to cloc)

---

## Next Steps

### Immediate (Phase 1):
1. **Update Dockerfile**: Add sembr, tree-sitter, universal-ctags
2. **Test sembr**: Verify semantic boundary detection on sample docs
3. **Update Context Explorer agent**: Add sembr usage for document partitioning
4. **Update Map-Reduce skill**: Document semantic partitioning strategies

### Near-term (Phase 2):
5. **MCP Integration** (ADVANCED): Configure sembr MCP server
   - Add to `projects/<name>/claude/mcp-servers.json`:
   ```json
   {
     "sembr": {
       "command": "sembr",
       "args": ["mcp"],
       "description": "Semantic boundary detection for documents"
     }
   }
   ```
   - Claude can call sembr directly via MCP (no bash wrapping needed!)
   - Benefit: More efficient, structured responses

6. **Create Examples**: Show before/after with fixed vs semantic partitioning
7. **Benchmark**: Measure accuracy improvement (expect ~15% from RLM research)

### Future (Phase 3):
8. **Hybrid Strategy**: Combine sembr (docs) + tree-sitter (code) automatically
9. **Auto-detection**: Context Explorer chooses tool based on file type
10. **Cost tracking**: Compare token usage with semantic vs fixed partitioning

---

## Related Files
- `/home/user/oubliette/Dockerfile` - Container tool installation
- `/home/user/oubliette/template/.claude/plugins/recursive-llm/agents/context-explorer.md` - Survey agent
- `/home/user/oubliette/template/.claude/plugins/recursive-llm/skills/map-reduce/SKILL.md` - Partitioning skill
- `/home/user/oubliette/RLM_COMPARISON_ANALYSIS.md` - Gap analysis (section 1.4)
