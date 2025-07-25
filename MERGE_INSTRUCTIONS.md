# Quick Reference: v0.2.0 Merge Instructions

## Current Branch Status
```bash
Current branch: feature/v0.2.0-document-migration
Base branch: main (v0.1.0)
Target: main (v0.2.0)
```

## When Ready to Merge Back to Main

### 1. Final Pre-Merge Steps
```bash
# Sync with main branch
git checkout feature/v0.2.0-document-migration
git fetch origin
git merge origin/main  # Resolve conflicts if any

# Run complete test suite
make test
make test-integration
make build

# Verify version bumps
grep -r "0\.1\.0" . --exclude-dir=.git
# Update any remaining 0.1.0 references to 0.2.0
```

### 2. Create Pull Request
```bash
# Push final changes
git push origin feature/v0.2.0-document-migration

# Create PR (via GitHub web interface or CLI)
# Title: "Release v0.2.0: Add Document Migration Support"
# Include:
# - Summary of new features
# - Breaking changes (if any)
# - Testing performed
# - Documentation updates
```

### 3. Merge Process
```bash
# Once PR approved, merge to main
git checkout main
git pull origin main
git merge feature/v0.2.0-document-migration
git push origin main
```

### 4. Release Tagging
```bash
# Create release tag
git tag -a v0.2.0 -m "Release v0.2.0: Document migration support

Features:
- Multi-media type support (images + documents)
- Separate bucket configuration per media type
- Document pattern detection in Markdown, HTML, CSS
- Backward compatibility with v0.1.0 configurations
- Enhanced CLI with media-type filtering

Breaking Changes: None
Migration Notes: Existing configurations continue to work unchanged"

git push origin v0.2.0
```

### 5. Cleanup
```bash
# Optional: Delete feature branch after successful merge
git branch -d feature/v0.2.0-document-migration
git push origin --delete feature/v0.2.0-document-migration
```

## Emergency Rollback
If critical issues found after merge:
```bash
# Find the merge commit
git log --oneline --merges

# Revert the merge
git revert -m 1 <merge-commit-hash>
git push origin main

# Create hotfix branch from last stable
git checkout -b hotfix/v0.2.1 v0.1.0
```

## Key Files to Update Before Merge
- [ ] `README.md` - Document new features
- [ ] `CHANGELOG.md` - Add v0.2.0 entry
- [ ] Version references in code
- [ ] Configuration examples
- [ ] CLI help text
- [ ] Integration tests
