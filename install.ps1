# pzmod installer for Windows (PowerShell).
#
#   irm https://pzmod.dev/install.ps1 | iex
#
# Environment:
#   PZMOD_VERSION   install a specific release tag (e.g. v3.0.0); defaults to latest.
#
# On Linux/macOS, use install.sh instead:
#   curl -fsSL https://pzmod.dev/install.sh | bash

#Requires -Version 5
$ErrorActionPreference = 'Stop'
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

$Repo = 'kldzj/pzmod'

$arch = switch ($env:PROCESSOR_ARCHITECTURE) {
  'AMD64' { 'x86_64' }
  'ARM64' { 'arm64' }
  'x86'   { 'i386' }
  default { throw "pzmod install: unsupported architecture '$($env:PROCESSOR_ARCHITECTURE)'" }
}

$tag = $env:PZMOD_VERSION
if (-not $tag) {
  $rel = Invoke-RestMethod -UseBasicParsing -Headers @{ 'User-Agent' = 'pzmod-install' } `
    "https://api.github.com/repos/$Repo/releases/latest"
  $tag = $rel.tag_name
}
if (-not $tag) { throw 'pzmod install: could not determine the latest release' }

$asset = "pzmod_windows_$arch.zip"
$base  = "https://github.com/$Repo/releases/download/$tag"
Write-Host "Installing pzmod $tag (windows/$arch)..."

$tmp = Join-Path ([IO.Path]::GetTempPath()) ("pzmod-" + [Guid]::NewGuid())
New-Item -ItemType Directory -Path $tmp | Out-Null
try {
  $zip = Join-Path $tmp $asset
  Invoke-WebRequest -UseBasicParsing "$base/$asset" -OutFile $zip

  # Verify against the release checksums when possible.
  try {
    $sums = Join-Path $tmp 'checksums.txt'
    Invoke-WebRequest -UseBasicParsing "$base/checksums.txt" -OutFile $sums
    $line = Select-String -Path $sums -SimpleMatch $asset | Select-Object -First 1
    if ($line) {
      $expected = ($line.Line -split '\s+')[0].ToLower()
      $actual   = (Get-FileHash -Algorithm SHA256 $zip).Hash.ToLower()
      if ($actual -ne $expected) { throw "checksum mismatch for $asset (expected $expected, got $actual)" }
      Write-Host 'Checksum verified.'
    } else {
      Write-Warning 'no checksum entry found; continuing.'
    }
  } catch {
    Write-Warning "could not verify checksum: $_"
  }

  $dest = Join-Path $env:LOCALAPPDATA 'pzmod'
  New-Item -ItemType Directory -Force -Path $dest | Out-Null
  Expand-Archive -Path $zip -DestinationPath $dest -Force
  Write-Host "pzmod installed to $dest\pzmod.exe"

  # Add to the user PATH if it isn't already there.
  $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
  if (($userPath -split ';') -notcontains $dest) {
    [Environment]::SetEnvironmentVariable('Path', "$userPath;$dest".Trim(';'), 'User')
    Write-Host "Added $dest to your user PATH. Restart your terminal, then run 'pzmod'."
  } else {
    Write-Host "Run 'pzmod' to get started."
  }
} finally {
  Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue
}
