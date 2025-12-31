#!/bin/bash
# migrate-to-v2.sh - Migrate schema files from v1.x to v2.0.0 naming convention
#
# This script updates all JSON files in the current directory and subdirectories
# to use the new v2.0.0 strategy naming convention:
#   - mergeRight  -> mergeRequest
#   - keepLeft    -> keepBase
#   - keepRight   -> keepRequest
#
# Usage:
#   ./scripts/migrate-to-v2.sh [directory]
#
# If no directory is specified, the current directory is used.
#
# IMPORTANT: This script modifies files in place. Make sure you have committed
# your changes or have a backup before running this script.

set -e

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Get target directory (default to current directory)
TARGET_DIR="${1:-.}"

echo -e "${YELLOW}KFS-Flow-Merge v2.0.0 Migration Script${NC}"
echo "=========================================="
echo ""
echo "This script will update all JSON files in:"
echo "  ${TARGET_DIR}"
echo ""
echo "The following replacements will be made:"
echo "  - \"mergeRight\"  -> \"mergeRequest\""
echo "  - \"keepLeft\"    -> \"keepBase\""
echo "  - \"keepRight\"   -> \"keepRequest\""
echo ""

# Check if directory exists
if [ ! -d "$TARGET_DIR" ]; then
    echo -e "${RED}Error: Directory '$TARGET_DIR' does not exist${NC}"
    exit 1
fi

# Confirm before proceeding
read -p "Do you want to proceed? (y/N) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Migration cancelled."
    exit 0
fi

echo ""
echo "Searching for JSON files..."

# Find all JSON files
JSON_FILES=$(find "$TARGET_DIR" -name "*.json" -type f)
FILE_COUNT=$(echo "$JSON_FILES" | grep -c . || echo "0")

if [ "$FILE_COUNT" -eq 0 ]; then
    echo -e "${YELLOW}No JSON files found in $TARGET_DIR${NC}"
    exit 0
fi

echo "Found $FILE_COUNT JSON file(s)"
echo ""

# Process each file
UPDATED_COUNT=0
for file in $JSON_FILES; do
    # Check if file contains any of the old strategy names
    if grep -q -E '"(mergeRight|keepLeft|keepRight)"' "$file"; then
        echo "Updating: $file"
        
        # Create backup
        cp "$file" "$file.bak"
        
        # Perform replacements
        sed -i '' \
            -e 's/"mergeRight"/"mergeRequest"/g' \
            -e 's/"keepLeft"/"keepBase"/g' \
            -e 's/"keepRight"/"keepRequest"/g' \
            "$file"
        
        # Verify the file is still valid JSON
        if ! python3 -m json.tool "$file" > /dev/null 2>&1; then
            echo -e "${RED}  Error: File is no longer valid JSON. Restoring backup.${NC}"
            mv "$file.bak" "$file"
        else
            echo -e "${GREEN}  ✓ Updated successfully${NC}"
            rm "$file.bak"
            ((UPDATED_COUNT++))
        fi
    fi
done

echo ""
echo "=========================================="
echo -e "${GREEN}Migration complete!${NC}"
echo "  Files processed: $FILE_COUNT"
echo "  Files updated: $UPDATED_COUNT"
echo ""

if [ "$UPDATED_COUNT" -gt 0 ]; then
    echo "Next steps:"
    echo "  1. Review the changes: git diff"
    echo "  2. Run your tests to verify everything works"
    echo "  3. Commit the changes: git commit -am 'Migrate to kfs-flow-merge v2.0.0'"
    echo ""
    echo "If you need to revert the changes:"
    echo "  git checkout -- ."
fi

