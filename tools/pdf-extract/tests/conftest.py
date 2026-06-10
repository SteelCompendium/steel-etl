"""Make the pdf-extract/ root importable when pytest runs from any cwd."""
import sys
from pathlib import Path

# Insert the tools/pdf-extract directory so `import normalize` resolves.
sys.path.insert(0, str(Path(__file__).parent.parent))
