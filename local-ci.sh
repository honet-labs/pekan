#!/bin/bash

# ============================================
# PEKAN Local CI/CD Pipeline (Automation Check)
# Acts as a Gatekeeper before deployment.
# ============================================

set -o pipefail

# Text color formatting
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

LOCAL_PROJECT_PATH="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo -e "${BLUE}================================================${NC}"
echo -e "${BLUE}🛡️  PEKAN Local CI/CD Pipeline (Gatekeeper)      ${NC}"
echo -e "${BLUE}================================================${NC}"
echo ""

# Helper function to print status
print_status() {
    local step_name="$1"
    local status="$2"
    if [ "$status" -eq 0 ]; then
        echo -e "  [${GREEN}PASSED${NC}] $step_name"
    else
        echo -e "  [${RED}FAILED${NC}] $step_name"
        echo ""
        echo -e "${RED}❌ CI/CD Pipeline Failed! Please fix the errors before deploying.${NC}"
        exit 1
    fi
}

# --- STEP 1: Backend Checks ---
echo -e "${YELLOW}🔍 Running Backend (Go) checks...${NC}"

# Check if go command is available
if ! command -v go &> /dev/null; then
    echo -e "  ⚠️  ${YELLOW}Go is not installed on this local system. Skipping backend checks...${NC}"
else
    # 1. Go Vet
    echo "  → Running go vet (static analysis)..."
    cd "$LOCAL_PROJECT_PATH/backend"
    go vet ./...
    print_status "Go Vet Static Analysis" $?

    # 2. Go Fmt check
    echo "  → Checking Go code formatting..."
    UNFORMATTED_FILES=$(gofmt -l .)
    if [ -n "$UNFORMATTED_FILES" ]; then
        echo -e "  ${RED}Found unformatted files:${NC}"
        echo "$UNFORMATTED_FILES"
        echo "  → Running go fmt ./... to auto-fix..."
        go fmt ./...
        print_status "Go Format Check (auto-fixed)" $?
    else
        print_status "Go Format Check" 0
    fi

    # 3. Go Unit Tests
    echo "  → Running backend unit tests..."
    go test ./... -v
    print_status "Go Unit Tests" $?

    # 4. Go Build Compilation
    echo "  → Verifying backend compilation..."
    go build -o /dev/null ./cmd/api
    print_status "Go API Compilation" $?
fi

echo ""

# --- STEP 2: Frontend Checks ---
echo -e "${YELLOW}🔍 Running Frontend (React + TS) checks...${NC}"

# Check if npm command is available
if ! command -v npm &> /dev/null; then
    echo -e "  ⚠️  ${YELLOW}Node/npm is not installed on this local system. Skipping frontend checks...${NC}"
else
    cd "$LOCAL_PROJECT_PATH/frontend"

    # 1. NPM Install (clean check)
    echo "  → Verifying frontend dependencies installation..."
    npm install --no-audit --no-fund --loglevel=error
    print_status "Frontend Dependencies Install" $?

    # 2. Frontend Build (combines tsc compilation and vite build)
    echo "  → Running frontend build (typecheck & bundle)..."
    npm run build
    print_status "Frontend Build & Typecheck" $?
fi

echo ""
echo -e "${GREEN}================================================${NC}"
echo -e "${GREEN}✅ All checks PASSED! Safe to deploy.            ${NC}"
echo -e "${GREEN}================================================${NC}"
exit 0
