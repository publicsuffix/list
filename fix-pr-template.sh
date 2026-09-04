#!/bin/bash
# Manual fix script for PR #3227 template issues
# Run this locally or via GitHub Actions

set -e

PR_NUMBER="3227"
REPO="publicsuffix/list"
GH_TOKEN="${GH_TOKEN:?GH_TOKEN environment variable required}"

echo "🔄 Fetching PR #$PR_NUMBER body..."
PR_BODY=$(gh pr view "$PR_NUMBER" --repo "$REPO" --json body --jq .body --raw)

echo "🔧 Running auto-fix script..."
export PR_NUMBER
export REPO
python3 .github/scripts/auto_fix_pr_template.py

echo "✅ PR #$PR_NUMBER has been auto-fixed!"
echo "📝 Visit: https://github.com/$REPO/pull/$PR_NUMBER"
