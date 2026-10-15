# Ḍād (ضاد) installer for Windows.
#
#   irm https://raw.githubusercontent.com/mtwalli/dad-lang/main/install.ps1 | iex
#
# Options (environment variables):
#   DAD_VERSION      release tag to install, e.g. v0.1.0 (default: latest)
#   DAD_INSTALL_DIR  where to put dad.exe (default: %LOCALAPPDATA%\dad\bin)

$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'

$Repo = 'mtwalli/dad-lang'
$Version = if ($env:DAD_VERSION) { $env:DAD_VERSION } else { 'latest' }
$InstallDir = if ($env:DAD_INSTALL_DIR) { $env:DAD_INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA 'dad\bin' }

$arch = switch ($env:PROCESSOR_ARCHITECTURE) {
  'AMD64' { 'amd64' }
  'ARM64' { 'arm64' }
  default { throw "Unsupported processor: $($env:PROCESSOR_ARCHITECTURE)" }
}

$name = "dad_windows_$arch"
$base = if ($Version -eq 'latest') {
  "https://github.com/$Repo/releases/latest/download"
} else {
  "https://github.com/$Repo/releases/download/$Version"
}

$tmp = Join-Path ([IO.Path]::GetTempPath()) ("dad-" + [guid]::NewGuid())
New-Item -ItemType Directory -Path $tmp | Out-Null
try {
  Write-Host "Downloading Ḍād ($Version) for windows/$arch..."
  $zip = Join-Path $tmp "$name.zip"
  $sums = Join-Path $tmp 'checksums.txt'
  Invoke-WebRequest -UseBasicParsing "$base/$name.zip" -OutFile $zip
  Invoke-WebRequest -UseBasicParsing "$base/checksums.txt" -OutFile $sums

  $line = Get-Content $sums | Where-Object { $_ -match " $([regex]::Escape("$name.zip"))$" }
  if (-not $line) { throw "No checksum listed for $name.zip" }
  $expected = ($line -split '\s+')[0].ToLower()
  $actual = (Get-FileHash -Algorithm SHA256 $zip).Hash.ToLower()
  if ($expected -ne $actual) { throw "Checksum mismatch for $name.zip" }

  Expand-Archive -Path $zip -DestinationPath $tmp -Force
  New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
  Copy-Item (Join-Path $tmp "$name\dad.exe") (Join-Path $InstallDir 'dad.exe') -Force
} finally {
  Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue
}

$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
if (-not ($userPath -split ';' | Where-Object { $_ -eq $InstallDir })) {
  $newPath = if ($userPath) { "$userPath;$InstallDir" } else { $InstallDir }
  [Environment]::SetEnvironmentVariable('Path', $newPath, 'User')
  Write-Host "Added $InstallDir to your PATH. Open a new terminal to use it."
}
$env:Path = "$env:Path;$InstallDir"

Write-Host "Installed: $(Join-Path $InstallDir 'dad.exe')"
& (Join-Path $InstallDir 'dad.exe') version
