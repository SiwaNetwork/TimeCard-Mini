#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Реконструкция заглушек из бинарника timebeat: валидные идентификаторы Go.

Проходит по всем .go в extracted_source и исправляет невалидный синтаксис:
  - func 1() / func 2() ... func 9() → func Extracted_Init_1() ...
  - func go()  → func main() в package main, иначе func Extracted_Go()
  - func type() → func Extracted_Type()

После перегенерации заглушек (extract_full_source.py) обязательно запустить:
  python3 code_analysis/reconstruct_stubs.py
или через WSL из корня репозитория.

Использование:
  python3 reconstruct_stubs.py [--dry-run]
"""

import os
import re
import sys

EXTRACTED_ROOT = os.path.join(os.path.dirname(__file__), "..", "extracted_source")

GO_KEYWORDS = {"go", "type", "func", "var", "const", "package", "import", "return", "select", "default", "interface", "struct", "map", "chan", "range"}
INVALID_NAMES = {"1", "2", "3", "4", "5", "6", "7", "8", "9", "0"}


def is_valid_go_identifier(name):
    if not name or name in GO_KEYWORDS or name in INVALID_NAMES:
        return False
    if name.endswith(".go") or name.endswith("-fm") or ".go" in name:
        return False
    if not (name[0].isalpha() or name[0] == "_"):
        return False
    for c in name[1:]:
        if not (c.isalnum() or c == "_"):
            return False
    return True


def sanitize_func_name(name):
    if is_valid_go_identifier(name):
        return name
    if name in GO_KEYWORDS:
        return "Extracted_" + name.capitalize()
    if name.isdigit():
        return "Extracted_Init_" + name
    s = re.sub(r"[^a-zA-Z0-9_]", "_", name)
    if s and (s[0].isalpha() or s[0] == "_"):
        return s
    return "Extracted_" + (s or "Unknown")


def fix_file(filepath, dry_run=False):
    with open(filepath, "r", encoding="utf-8", errors="replace") as f:
        content = f.read()
    original = content

    # func 1() .. func 9() → func Extracted_Init_N()
    for digit in "123456789":
        content = re.sub(r"\bfunc\s+" + digit + r"\s*\(", "func Extracted_Init_" + digit + "(", content)

    # func go() → main в package main, иначе Extracted_Go
    if "package main" in content and "func go()" in content:
        content = content.replace("func go() {", "func main() {", 1)
    content = re.sub(r"\bfunc\s+go\s*\(", "func Extracted_Go(", content)

    # func type(
    content = re.sub(r"\bfunc\s+type\s*\(", "func Extracted_Type(", content)

    if content != original:
        if not dry_run:
            with open(filepath, "w", encoding="utf-8") as f:
                f.write(content)
        return True
    return False


def main():
    dry_run = "--dry-run" in sys.argv
    root = os.path.abspath(EXTRACTED_ROOT)
    if not os.path.isdir(root):
        print("Not found:", root)
        return 1
    fixed = 0
    for dirpath, _, filenames in os.walk(root):
        for f in filenames:
            if f.endswith(".go"):
                path = os.path.join(dirpath, f)
                if fix_file(path, dry_run=dry_run):
                    fixed += 1
                    print("Fixed:", path)
    print("Fixed", fixed, "files" + (" (dry-run)" if dry_run else ""))
    return 0


if __name__ == "__main__":
    sys.exit(main())
