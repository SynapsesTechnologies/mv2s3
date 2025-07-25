# mv2s3 v0.2.0 Development & Merge Guide

## Overview
This guide outlines the development workflow for implementing document migration support in mv2s3 v0.2.0, including branch management, testing strategy, and merge procedures.

## Branch Structure

### Current Branch Setup
- **Main Branch**: `main` - Production-ready code (v0.1.0)
- **Feature Branch**: `feature/v0.2.0-document-migration` - Development branch for v0.2.0

### Branch Protection Strategy
```bash
# Current branch
git branch
# * feature/v0.2.0-document-migration

# Remote tracking
git branch -vv
# * feature/v0.2.0-document-migration e6c766f [origin/feature/v0.2.0-document-migration] Added README.md
```

## Development Workflow

### 1. Daily Development Process
```bash
# Start each dev session with latest changes
git checkout feature/v0.2.0-document-migration
git pull origin feature/v0.2.0-document-migration

# Create focused commits for each component
git add .
git commit -m "feat: add MediaType enum and classification system"
git push origin feature/v0.2.0-document-migration
```

### 2. Commit Message Convention
Follow conventional commits for clear history:
```
feat: add document pattern detection to LinkParser
fix: resolve media type detection edge case
docs: update configuration examples for document support
test: add integration tests for multi-media migration
refactor: extract media type logic to separate package
```

### 3. Feature Implementation Phases

#### Phase 1: Core Infrastructure
```bash
# Commits in this phase:
git commit -m "feat: add MediaType enum and constants"
git commit -m "feat: extend MigrationConfig with media type support"
git commit -m "feat: add MediaTypeDetector component"
git commit -m "feat: implement backward compatibility layer"
```

#### Phase 2: Scanner Enhancements
```bash
git commit -m "feat: add document patterns to LinkParser"
git commit -m "feat: extend FileScanner with media filtering"
git commit -m "feat: implement document reference detection"
git commit -m "test: add scanner tests for document patterns"
```

#### Phase 3: Storage and Processing
```bash
git commit -m "feat: implement multi-bucket support in S3Client"
git commit -m "feat: add document processing capabilities"
git commit -m "feat: update URL replacement for media types"
git commit -m "test: add storage integration tests"
```

#### Phase 4: CLI and Configuration
```bash
git commit -m "feat: add media-type CLI flags"
git commit -m "feat: update configuration templates"
git commit -m "feat: implement configuration migration"
git commit -m "docs: update README with document support"
```

## Testing Strategy

### 1. Continuous Testing During Development
```bash
# Run tests after each significant change
make test

# Run specific test suites
go test ./internal/scanner/...
go test ./internal/storage/...
go test ./pkg/types/...

# Run integration tests
go test ./test/integration/...
```

### 2. Pre-Merge Testing Checklist
Before merging to main, ensure all tests pass:

```bash
# Full test suite
make test

# Test coverage check
make test-coverage

# Integration tests with real S3 (if applicable)
make test-integration

# Build verification
make build

# Cross-platform build test
make build-all
```

### 3. Manual Testing Scenarios
```bash
# Test backward compatibility
mv2s3 migrate --source ./test/fixtures --bucket test-images

# Test document-only migration
mv2s3 migrate --source ./test/fixtures --media-types documents --bucket test-docs

# Test mixed media migration
mv2s3 migrate --source ./test/fixtures --media-types images,documents \
  --image-bucket test-images --document-bucket test-docs

# Test configuration migration
mv2s3 config show --format yaml
```

## Merge Process

### Step 1: Pre-Merge Preparation
```bash
# Ensure feature branch is up to date with main
git checkout feature/v0.2.0-document-migration
git fetch origin
git merge origin/main

# Resolve any conflicts if they exist
# Run full test suite after merge
make test
```

### Step 2: Final Validation
```bash
# Version bump check
grep -r "0\.1\.0" . --exclude-dir=.git
# Update any remaining version references to 0.2.0

# Documentation updates
# Ensure README.md reflects new capabilities
# Update CHANGELOG.md with v0.2.0 changes
# Verify configuration examples are current
```

### Step 3: Create Pull Request
```bash
# Push final changes
git push origin feature/v0.2.0-document-migration

# Create PR via GitHub CLI (if available) or web interface
gh pr create --title "Release v0.2.0: Document Migration Support" \
  --body-file .github/pull_request_template.md \
  --base main --head feature/v0.2.0-document-migration
```

### Step 4: Merge to Main
Once PR is approved and all checks pass:

```bash
# Option A: Merge via GitHub (recommended for review trail)
# Use "Squash and merge" or "Create a merge commit" based on preference

# Option B: Local merge (if doing locally)
git checkout main
git pull origin main
git merge --no-ff feature/v0.2.0-document-migration -m "Merge feature/v0.2.0-document-migration into main"
git push origin main
```

### Step 5: Post-Merge Tasks
```bash
# Tag the release
git tag -a v0.2.0 -m "Release v0.2.0: Document migration support"
git push origin v0.2.0

# Clean up feature branch (optional)
git branch -d feature/v0.2.0-document-migration
git push origin --delete feature/v0.2.0-document-migration

# Update any deployment/release automation
```

## Release Checklist

### Before Merging to Main
- [ ] All tests pass (`make test`)
- [ ] Integration tests pass (`make test-integration`)
- [ ] Documentation updated (README.md, CHANGELOG.md)
- [ ] Version numbers updated to 0.2.0
- [ ] Backward compatibility verified
- [ ] Configuration migration tested
- [ ] CLI help text updated
- [ ] Example configurations tested

### After Merging to Main
- [ ] Create and push v0.2.0 tag
- [ ] Update GitHub release notes
- [ ] Update package documentation
- [ ] Clean up feature branch
- [ ] Notify team/users of new version

## Rollback Strategy

If issues are discovered after merge:

```bash
# Option 1: Revert the merge commit
git revert -m 1 <merge-commit-hash>

# Option 2: Create hotfix branch from previous stable tag
git checkout -b hotfix/v0.1.1 v0.1.0
# Apply minimal fixes
# Merge hotfix to main
# Tag as v0.1.1
```

## Development Best Practices

### 1. Keep Commits Atomic
- Each commit should represent a single logical change
- Commits should compile and pass tests
- Use descriptive commit messages

### 2. Test Early and Often
- Write tests before implementing features (TDD)
- Run tests locally before pushing
- Use CI/CD feedback to catch issues

### 3. Document as You Go
- Update documentation with each feature
- Include usage examples
- Keep configuration references current

### 4. Maintain Backward Compatibility
- Existing configurations should continue to work
- Deprecated features should have migration paths
- API changes should be additive when possible

## Contact and Support

For questions during development:
- Create issues in the GitHub repository
- Use descriptive issue titles and include steps to reproduce
- Tag issues with appropriate labels (bug, enhancement, documentation)

---

**Current Status**: Development branch created and ready for v0.2.0 implementation
**Next Steps**: Begin Phase 1 implementation with MediaType infrastructure
