#!/usr/bin/env python3
"""Builds dad-lang-<version>.vsix without Node tools (the release workflow uses vsce instead).

Usage: python3 scripts/package.py [output-dir]
"""
import fnmatch
import json
import pathlib
import sys
import zipfile
from xml.sax.saxutils import escape

root = pathlib.Path(__file__).resolve().parent.parent
pkg = json.loads((root / "package.json").read_text(encoding="utf-8"))
out_dir = pathlib.Path(sys.argv[1]) if len(sys.argv) > 1 else root
vsix = out_dir / f"{pkg['name']}-{pkg['version']}.vsix"

ignore = [l.strip() for l in (root / ".vscodeignore").read_text().splitlines() if l.strip()]
skip_dirs = {"scripts", "node_modules", ".git"}


def included(rel: pathlib.PurePosixPath) -> bool:
    if rel.parts[0] in skip_dirs or rel.name == ".vscodeignore":
        return False
    path = rel.as_posix()
    for pat in ignore:
        if pat.endswith("/**") and path.startswith(pat[:-2]):
            return False
        if fnmatch.fnmatch(path, pat):
            return False
    return True


files = sorted(
    p.relative_to(root).as_posix()
    for p in root.rglob("*")
    if p.is_file() and included(pathlib.PurePosixPath(p.relative_to(root).as_posix()))
)

keywords = ";".join(pkg.get("keywords", []))
manifest = f"""<?xml version="1.0" encoding="utf-8"?>
<PackageManifest Version="2.0.0" xmlns="http://schemas.microsoft.com/developer/vsx-schema/2011" xmlns:d="http://schemas.microsoft.com/developer/vsx-schema-design/2011">
  <Metadata>
    <Identity Language="en-US" Id="{pkg['name']}" Version="{pkg['version']}" Publisher="{pkg['publisher']}" />
    <DisplayName>{escape(pkg['displayName'])}</DisplayName>
    <Description xml:space="preserve">{escape(pkg['description'])}</Description>
    <Tags>{escape(keywords)}</Tags>
    <Categories>{escape(",".join(pkg.get('categories', [])))}</Categories>
    <GalleryFlags>Public</GalleryFlags>
    <Properties>
      <Property Id="Microsoft.VisualStudio.Code.Engine" Value="{pkg['engines']['vscode']}" />
      <Property Id="Microsoft.VisualStudio.Code.ExtensionDependencies" Value="" />
      <Property Id="Microsoft.VisualStudio.Code.ExtensionPack" Value="" />
      <Property Id="Microsoft.VisualStudio.Code.ExtensionKind" Value="workspace" />
      <Property Id="Microsoft.VisualStudio.Services.Links.Source" Value="{pkg['repository']['url']}" />
      <Property Id="Microsoft.VisualStudio.Services.Content.Pricing" Value="Free"/>
    </Properties>
    <Icon>extension/{pkg['icon']}</Icon>
  </Metadata>
  <Installation>
    <InstallationTarget Id="Microsoft.VisualStudio.Code"/>
  </Installation>
  <Dependencies/>
  <Assets>
    <Asset Type="Microsoft.VisualStudio.Code.Manifest" Path="extension/package.json" Addressable="true" />
    <Asset Type="Microsoft.VisualStudio.Services.Content.Details" Path="extension/README.md" Addressable="true" />
    <Asset Type="Microsoft.VisualStudio.Services.Content.Changelog" Path="extension/CHANGELOG.md" Addressable="true" />
    <Asset Type="Microsoft.VisualStudio.Services.Icons.Default" Path="extension/{pkg['icon']}" Addressable="true" />
  </Assets>
</PackageManifest>
"""

types = {".json": "application/json", ".js": "application/javascript", ".md": "text/markdown",
         ".png": "image/png", ".svg": "image/svg+xml", ".vsixmanifest": "text/xml"}
exts = sorted({pathlib.PurePosixPath(f).suffix for f in files} | {".vsixmanifest"})
content_types = '<?xml version="1.0" encoding="utf-8"?>\n<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">' + "".join(
    f'<Default Extension="{e}" ContentType="{types.get(e, "application/octet-stream")}"/>' for e in exts) + "</Types>\n"

with zipfile.ZipFile(vsix, "w", zipfile.ZIP_DEFLATED) as z:
    z.writestr("extension.vsixmanifest", manifest)
    z.writestr("[Content_Types].xml", content_types)
    for f in files:
        z.write(root / f, "extension/" + f)
print(vsix)
for f in files:
    print("  extension/" + f)
