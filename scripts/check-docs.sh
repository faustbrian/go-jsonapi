#!/usr/bin/env bash
set -euo pipefail

required=(
  LICENSE
  README.md
  CHANGELOG.md
  CONTRIBUTING.md
  SECURITY.md
  docs/README.md
  docs/quickstart.md
  docs/architecture.md
  docs/api-reference.md
  docs/features.md
  docs/conformance.md
  docs/extensions-and-profiles.md
  docs/recommendations.md
  docs/adoption.md
  docs/examples.md
  docs/cookbook.md
  docs/faq.md
  docs/troubleshooting.md
  docs/migration.md
  docs/compatibility.md
  docs/performance.md
  docs/releasing.md
)
for file in "${required[@]}"; do
  if [[ ! -s "$file" ]]; then
    echo "required documentation is missing or empty: $file" >&2
    exit 1
  fi
done

python3 - <<'PY'
from pathlib import Path
import re

for document in Path(".").rglob("*.md"):
    content = document.read_text(encoding="utf-8")
    for target in re.findall(r"\[[^\]]*\]\(([^)]+)\)", content):
        if target.startswith(("http://", "https://", "mailto:", "#")):
            continue
        relative = target.split("#", 1)[0]
        if relative.startswith("<") and relative.endswith(">"):
            relative = relative[1:-1]
        resolved = (document.parent / relative).resolve()
        if not resolved.exists():
            raise SystemExit(f"broken relative link in {document}: {target}")

print("all relative Markdown links resolve")
PY

go test ./... -run '^Example'
