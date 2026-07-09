#!/usr/bin/env bash
# ==============================================================================
# LingMirror AIOS - Documentation Sync Linter
# Checks that all active Gin endpoints in internal packages are documented
# in docs/reference-module-catalog.md.
# ==============================================================================
set -euo pipefail

CATALOG="docs/reference-module-catalog.md"
if [ ! -f "$CATALOG" ]; then
  echo "Error: Catalog file $CATALOG not found"
  exit 1
fi

echo "Verifying API endpoints in docs/reference-module-catalog.md..."

python3 - "$CATALOG" << 'EOF'
import os
import re
import sys

catalog_path = sys.argv[1]
backend_dir = "backend-go/internal"

if not os.path.exists(catalog_path):
    print(f"Error: Catalog file {catalog_path} not found")
    sys.exit(1)

# Step 1: Parse all documented routes from catalog
with open(catalog_path, "r", encoding="utf-8") as f:
    catalog_content = f.read()

# Extract backticked paths
catalog_paths = set()
for match in re.findall(r'`(/api/v1/[^`]+)`', catalog_content):
    for p in [x.strip() for x in match.split(",")]:
        catalog_paths.add(p)

for match in re.findall(r'`(/ws[^`]*)`', catalog_content):
    catalog_paths.add(match)
for match in re.findall(r'`(/metrics[^`]*)`', catalog_content):
    catalog_paths.add(match)
for match in re.findall(r'`(/api/health[^`]*)`', catalog_content):
    catalog_paths.add(match)

# Expand brackets in catalog paths
expanded_catalog_paths = set()
def expand_brackets(path):
    if '[' not in path:
        return [path]
    start = path.find('[')
    end = path.find(']')
    if start != -1 and end != -1:
        inner = path[start+1:end]
        before = path[:start]
        after = path[end+1:]
        return expand_brackets(before + after) + expand_brackets(before + inner + after)
    return [path]

for p in catalog_paths:
    for expanded in expand_brackets(p):
        expanded_catalog_paths.add(expanded)

def normalize_path(path):
    path = re.sub(r':\w+', '{PARAM}', path)
    path = re.sub(r'\*\w+', '{WILDCARD}', path)
    path = re.sub(r'/+', '/', path)
    return path.rstrip('/')

normalized_catalog = {normalize_path(p) for p in expanded_catalog_paths}

# Known exclusions (e.g. mock endpoints, comments, or in-development modules)
EXCLUDED_PREFIXES = {
    "/api/v1/reliability",
    "/api/v1/user",
    "/api/v1/domain/path",
    "/api/v1/mock/unregistered-test",
    "/ws/extension",
    "/swagger",
}

# Alias mappings to bridge naming style differences
aliases = {
    "/api/v1/product-master": "/api/v1/products",
    "/api/v1/sourcing-1688": "/api/v1/sourcing1688",
    "/api/v1/operation-log": "/api/v1/operationlog",
    "/api/v1/image-gen": "/api/v1/imagegen",
}

# Step 2: Scan Go files for route registrations
go_files = []
for root, dirs, files in os.walk(backend_dir):
    # Skip test files and mock directories
    if any(x in root for x in ["dbtest", "mock"]):
        continue
    for file in files:
        if file.endswith(".go") and not file.endswith("_test.go") and not file == "testserver.go":
            go_files.append(os.path.join(root, file))

all_extracted_routes = []

for filepath in go_files:
    with open(filepath, 'r', encoding='utf-8') as f:
        content = f.read()

    # Strip comments to avoid matching commented routes
    # Multi-line
    content = re.sub(r'/\*.*?\*/', '', content, flags=re.DOTALL)
    # Single-line
    lines = content.split('\n')
    clean_lines = []
    for line in lines:
        if '//' in line:
            idx = line.find('//')
            before = line[:idx]
            if before.count('"') % 2 == 0:
                line = before
        clean_lines.append(line)
    clean_content = '\n'.join(clean_lines)

    # Detect RegisterRoutes parameter name
    root_vars = {"rg", "r", "protected", "api"}
    func_match = re.search(r'func\s+(?:RegisterRoutes|RegisterWebhookRoutes|RegisterWebhookAdminRoutes|RegisterPublicRoutes)\(\s*(\w+)', clean_content)
    if func_match:
        root_vars.add(func_match.group(1))

    is_router_go = os.path.basename(filepath) == "router.go"
    prefixes = {}
    for var_name in root_vars:
        if is_router_go and var_name == "r":
            prefixes[var_name] = ""
        else:
            prefixes[var_name] = "/api/v1"

    # Process group registrations
    group_regex = re.compile(r'(\w+)\s*:=\s*(\w+)\.Group\("([^"]+)"\)')
    for match in group_regex.finditer(clean_content):
        child_var, parent_var, group_path = match.groups()
        if parent_var in prefixes:
            parent_prefix = prefixes[parent_var]
            prefixes[child_var] = (parent_prefix + "/" + group_path).replace("//", "/")
        else:
            prefixes[child_var] = ("/api/v1/" + group_path).replace("//", "/")

    # Process route registrations
    route_regex = re.compile(r'(\w+)\.(GET|POST|PUT|DELETE|PATCH|OPTIONS|HEAD|Any)\("([^"]*)"')
    for match in route_regex.finditer(clean_content):
        var_name, method, route_path = match.groups()
        if var_name in prefixes:
            prefix = prefixes[var_name]
            full_path = (prefix + "/" + route_path).replace("//", "/")
            all_extracted_routes.append((method, full_path, filepath))
        else:
            full_path = ("/api/v1/" + var_name + "/" + route_path).replace("//", "/")
            all_extracted_routes.append((method, full_path, filepath))

# Step 3: Validate extracted routes against normalized catalog
errors = []
verified_count = 0

for method, path, filepath in all_extracted_routes:
    # Check exclusions
    is_excluded = False
    for excl in EXCLUDED_PREFIXES:
        if path.startswith(excl):
            is_excluded = True
            break
    if is_excluded:
        continue

    # Apply aliases
    mapped_path = path
    for alias_src, alias_dst in aliases.items():
        if path.startswith(alias_src):
            mapped_path = path.replace(alias_src, alias_dst, 1)
            break

    norm_path = normalize_path(mapped_path)

    # Check if norm_path is in catalog
    if norm_path not in normalized_catalog:
        # Also check if base path is in catalog as a fallback (e.g. /api/v1/something)
        # to handle cases where nested sub-routes are not individually documented
        parts = norm_path.split('/')
        base_path = "/".join(parts[:4]) # e.g. /api/v1/something
        if base_path in normalized_catalog:
            verified_count += 1
            continue

        errors.append(f"Error: Extracted route {method} {path} (normalized: {norm_path}) in {filepath} is not documented in {catalog_path}")
    else:
        verified_count += 1

if errors:
    for err in errors:
        print(err)
    print(f"\nVerification failed with {len(errors)} errors.")
    sys.exit(1)

print(f"Verification complete. {verified_count} active Gin routes verified successfully against {catalog_path}.")
sys.exit(0)
EOF
