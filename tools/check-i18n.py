import json, re, sys, pathlib
root = pathlib.Path("ui/App")
loc = root / "locales"
def flat(d, p=""):
    out = {}
    for k, v in d.items():
        key = f"{p}.{k}" if p else k
        if isinstance(v, dict): out.update(flat(v, key))
        else: out[key] = v
    return out
langs = {f.stem: flat(json.load(open(f, encoding="utf-8"))) for f in sorted(loc.glob("*.json"))}
en = langs["en"]
print("== Ключи из en, которых нет в других локалях")
for lang, d in langs.items():
    if lang == "en": continue
    miss = sorted(set(en) - set(d))
    print(f"[{lang}] {len(miss)}")
    for k in miss: print("   ", k)
print("\n== Ключи из t(...) в коде, которых нет в локали")
used = {}
for f in root.rglob("*.jsx"):
    for i, line in enumerate(f.read_text(encoding="utf-8").splitlines(), 1):
        for m in re.finditer(r"\bt\(\s*[\"']([A-Za-z0-9_.\-]+)[\"']", line):
            used.setdefault(m.group(1), []).append(f"{f}:{i}")
for lang, d in langs.items():
    miss = sorted(k for k in used if k not in d)
    print(f"[{lang}] {len(miss)}")
    for k in miss: print("   ", k, "<-", used[k][0])
print("\n== Английский текст прямо в JSX (шумно, смотреть глазами)")
for f in root.rglob("*.jsx"):
    for i, line in enumerate(f.read_text(encoding="utf-8").splitlines(), 1):
        s = line.strip()
        if "t(" in s or s.startswith(("//", "import", "*")): continue
        for m in re.finditer(r">\s*([A-Z][A-Za-z][A-Za-z ,.'\-—:!?]{2,})\s*<", s):
            print(f"{f}:{i}: {m.group(1)}")
        for m in re.finditer(r"(?:placeholder|title|label)=\"([A-Za-z][^\"{]{2,})\"", s):
            print(f"{f}:{i}: attr {m.group(1)}")
