#!/usr/bin/env python3
"""Strip all markdown links from the input document."""

import re
from pathlib import Path

p = Path(__file__).parent.parent / "input" / "heroes" / "Draw Steel Heroes.md"
content = p.read_text()

original_count = len(re.findall(r'\[([^\]]+)\]\([^)]+\)', content))

# Strip markdown links: [text](url) -> text
stripped = re.sub(r'\[([^\]]+)\]\([^)]+\)', r'\\1', content)

remaining_count = len(re.findall(r'\[([^\]]+)\]\([^)]+\)', stripped))
p.write_text(stripped)

print(f"Stripped {original_count - remaining_count} links ({remaining_count} remaining)")
