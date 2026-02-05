#!/bin/bash
# Oubliette Pattern Enforcement Checks
#
# This script checks for common pattern violations in the codebase.
# Run before committing or in CI to catch issues early.
#
# Usage: ./tools/check-patterns.sh [--strict] [--pattern PATTERN_NAME]

set -e

# Colors for output
RED='\033[0;31m'
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Counters
ERRORS=0
WARNINGS=0

# Flags
STRICT_MODE=false
SPECIFIC_PATTERN=""

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --strict)
            STRICT_MODE=true
            shift
            ;;
        --pattern)
            SPECIFIC_PATTERN="$2"
            shift 2
            ;;
        *)
            echo "Unknown option: $1"
            echo "Usage: $0 [--strict] [--pattern PATTERN_NAME]"
            exit 1
            ;;
    esac
done

echo -e "${BLUE}🔍 Oubliette Pattern Enforcement Checks${NC}"
echo "================================================"
echo ""

# Helper functions
error() {
    echo -e "${RED}❌ ERROR: $1${NC}"
    ((ERRORS++))
}

warning() {
    echo -e "${YELLOW}⚠️  WARNING: $1${NC}"
    ((WARNINGS++))
}

info() {
    echo -e "${BLUE}ℹ️  INFO: $1${NC}"
}

success() {
    echo -e "${GREEN}✅ $1${NC}"
}

# Check 1: context.Background() in non-main code
check_context_background() {
    if [[ -n "$SPECIFIC_PATTERN" && "$SPECIFIC_PATTERN" != "context-background" ]]; then
        return
    fi

    echo "Checking for context.Background() violations..."

    # Find context.Background() in internal/ excluding comments with "OK:"
    if git grep -n "context.Background()" -- 'internal/*.go' | grep -v "// OK:"; then
        error "Found context.Background() in internal/ (see docs/PATTERNS.md #8)"
        info "Add context parameter to these methods instead"
        echo ""
    else
        success "No context.Background() violations"
    fi
}

# Check 2: Deprecated os.IsNotExist
check_deprecated_error_inspection() {
    if [[ -n "$SPECIFIC_PATTERN" && "$SPECIFIC_PATTERN" != "error-inspection" ]]; then
        return
    fi

    echo "Checking for deprecated error inspection..."

    local count
    count=$(git grep -n "os.IsNotExist" -- '*.go' 2>/dev/null | wc -l | tr -d ' ')

    if [ "$count" -gt 0 ]; then
        warning "Found $count uses of deprecated os.IsNotExist (see docs/PATTERNS.md #7)"
        info "Migrate to: errors.Is(err, fs.ErrNotExist)"

        if [ "$STRICT_MODE" = true ]; then
            ((ERRORS++))
        fi
        echo ""
    else
        success "No deprecated error inspection found"
    fi
}

# Check 3: Manager size limit
check_manager_size() {
    if [[ -n "$SPECIFIC_PATTERN" && "$SPECIFIC_PATTERN" != "manager-size" ]]; then
        return
    fi

    echo "Checking manager file sizes..."

    local oversized=false
    for file in internal/*/manager.go; do
        if [ -f "$file" ]; then
            local lines
            lines=$(wc -l < "$file")
            if [ "$lines" -gt 500 ]; then
                warning "$file has $lines lines (limit: 500, see docs/PATTERNS.md #1)"
                info "Consider splitting into multiple files"
                oversized=true
            fi
        fi
    done

    if [ "$oversized" = false ]; then
        success "All manager files within size limit"
    fi
    echo ""
}

# Check 4: Error wrapping with %w
check_error_wrapping() {
    if [[ -n "$SPECIFIC_PATTERN" && "$SPECIFIC_PATTERN" != "error-wrapping" ]]; then
        return
    fi

    echo "Checking error wrapping patterns..."

    # Find fmt.Errorf with %s instead of %w for errors
    if git grep -n 'fmt\.Errorf.*%s.*err\b' -- 'internal/*.go' 'cmd/*.go'; then
        error "Found error wrapping with %s instead of %w (see docs/PATTERNS.md #7)"
        info "Use fmt.Errorf(\"context: %w\", err) for proper error chains"
        echo ""
    else
        success "Error wrapping correct"
    fi
}

# Check 5: Mutex naming
check_mutex_naming() {
    if [[ -n "$SPECIFIC_PATTERN" && "$SPECIFIC_PATTERN" != "mutex-naming" ]]; then
        return
    fi

    echo "Checking mutex naming conventions..."

    # Find sync.Mutex or sync.RWMutex without "Mu" or "mu" in name
    local violations
    violations=$(git grep -n 'sync\.\(RW\)\?Mutex' -- 'internal/*.go' | grep -v -E '(Mu|mu)\s*sync\.')

    if [ -n "$violations" ]; then
        warning "Found mutexes without 'Mu' suffix (see docs/PATTERNS.md #9)"
        info "Convention: use suffix like cacheMu, sessionsMu"
        echo "$violations"
        echo ""
    else
        success "Mutex naming correct"
    fi
}

# Check 6: Defer on resource cleanup
check_defer_usage() {
    if [[ -n "$SPECIFIC_PATTERN" && "$SPECIFIC_PATTERN" != "defer-cleanup" ]]; then
        return
    fi

    echo "Checking defer usage for resource cleanup..."

    # Find Close() calls without defer in same function
    # This is a heuristic check - may have false positives
    local files_to_check
    files_to_check=$(git grep -l '\.Close()' -- 'internal/*.go' 'cmd/*.go')

    local missing_defer=false
    for file in $files_to_check; do
        # Count Close() vs defer Close() in each function
        # This is simplified - full analysis would need AST parsing
        local close_count
        local defer_close_count
        close_count=$(grep -c '\.Close()' "$file" || true)
        defer_close_count=$(grep -c 'defer.*\.Close()' "$file" || true)

        if [ "$close_count" -gt "$defer_close_count" ]; then
            warning "$file may have Close() without defer"
            missing_defer=true
        fi
    done

    if [ "$missing_defer" = false ]; then
        success "Resource cleanup looks good"
    else
        info "Manual review recommended for Close() usage"
    fi
    echo ""
}

# Check 7: Public methods with context parameter
check_context_parameter() {
    if [[ -n "$SPECIFIC_PATTERN" && "$SPECIFIC_PATTERN" != "context-parameter" ]]; then
        return
    fi

    echo "Checking context parameter usage..."

    # Find exported functions in internal/ that don't take context
    # This is a heuristic - checks func signatures
    local violations
    violations=$(git grep -n '^func ([^)]*) [A-Z][^(]*([^)]*) ' -- 'internal/*.go' | grep -v 'ctx context\.Context' || true)

    if [ -n "$violations" ]; then
        warning "Found exported methods without context parameter"
        info "Convention: All public methods should accept context.Context as first parameter"
        if [ "$STRICT_MODE" = false ]; then
            info "Note: Some exceptions are acceptable (constructors, simple getters)"
        fi
        echo ""
    else
        success "Context parameter usage looks good"
    fi
}

# Check 8: Test coverage
check_test_coverage() {
    if [[ -n "$SPECIFIC_PATTERN" && "$SPECIFIC_PATTERN" != "test-coverage" ]]; then
        return
    fi

    echo "Checking test coverage..."

    if [ -d "test/cmd" ]; then
        cd test/cmd
        if go run . --coverage-report > /dev/null 2>&1; then
            success "Test coverage check passed"
        else
            warning "Test coverage check returned non-zero exit code"
            info "Run 'cd test/cmd && go run . --coverage-report' for details"
        fi
        cd - > /dev/null
        echo ""
    else
        info "Test coverage tool not found, skipping"
        echo ""
    fi
}

# Check 9: Magic numbers/strings
check_magic_values() {
    if [[ -n "$SPECIFIC_PATTERN" && "$SPECIFIC_PATTERN" != "magic-values" ]]; then
        return
    fi

    echo "Checking for magic numbers/strings..."

    # Look for repeated string literals (potential constants)
    # This is best done with goconst tool
    if command -v goconst &> /dev/null; then
        local violations
        violations=$(goconst -min 3 ./... 2>/dev/null | head -20)

        if [ -n "$violations" ]; then
            warning "Found repeated string literals that could be constants"
            info "Consider extracting to named constants"
            echo "$violations"
            echo ""
        else
            success "No obvious magic values found"
        fi
    else
        info "goconst not installed, skipping (install: go install github.com/jgautheron/goconst/cmd/goconst@latest)"
        echo ""
    fi
}

# Check 10: Goroutine leak prevention
check_goroutine_patterns() {
    if [[ -n "$SPECIFIC_PATTERN" && "$SPECIFIC_PATTERN" != "goroutines" ]]; then
        return
    fi

    echo "Checking goroutine patterns..."

    # Find go func() without buffered channel or done channel
    # This is a heuristic check
    local violations
    violations=$(git grep -A 5 'go func()' -- 'internal/*.go' | grep -v 'done.*chan' | grep -v 'make(chan.*1)' || true)

    if [ -n "$violations" ]; then
        warning "Found goroutines that may not have proper cleanup"
        info "Use buffered channels or done channels to prevent leaks (see docs/PATTERNS.md #10)"
        if [ "$STRICT_MODE" = false ]; then
            info "Note: Some goroutines are intentionally long-lived"
        fi
        echo ""
    else
        success "Goroutine patterns look good"
    fi
}

# Run all checks
echo "Running pattern checks..."
echo ""

check_context_background
check_deprecated_error_inspection
check_manager_size
check_error_wrapping
check_mutex_naming
check_defer_usage
check_context_parameter
check_test_coverage
check_magic_values
check_goroutine_patterns

# Summary
echo "================================================"
echo -e "${BLUE}📊 Summary${NC}"
echo "================================================"
echo ""

if [ $ERRORS -eq 0 ] && [ $WARNINGS -eq 0 ]; then
    echo -e "${GREEN}🎉 All pattern checks passed!${NC}"
    exit 0
elif [ $ERRORS -eq 0 ]; then
    echo -e "${YELLOW}⚠️  $WARNINGS warnings found${NC}"
    echo "   Review warnings and consider fixing"
    echo ""
    if [ "$STRICT_MODE" = true ]; then
        echo "   (Running in strict mode - warnings are treated as errors)"
        exit 1
    fi
    exit 0
else
    echo -e "${RED}❌ $ERRORS errors found${NC}"
    if [ $WARNINGS -gt 0 ]; then
        echo -e "${YELLOW}   $WARNINGS warnings found${NC}"
    fi
    echo ""
    echo "   Fix errors before committing"
    echo "   See docs/PATTERNS.md for guidance"
    exit 1
fi
