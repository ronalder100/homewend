# homewend — Copyright (C) 2026 Ron Alder
# SPDX-License-Identifier: AGPL-3.0-or-later

# Installs the latest homewend release on Windows:
#
#   irm https://homewend.app/install.ps1 | iex
#
# The binary goes to %LOCALAPPDATA%\Programs\homewend, or to
# HOMEWEND_INSTALL_DIR if set, and that folder is added to the user's PATH.
# Nothing needs an administrator, and the download is checked against the
# release's checksums before it is installed.
#
# The whole script is one block, run in a scope of its own: under iex it runs
# in the user's session, which would otherwise keep its variables and its
# $ErrorActionPreference afterwards.
& {
    $ErrorActionPreference = "Stop"

    $repo = "ronalder100/homewend"
    $dir = if ($env:HOMEWEND_INSTALL_DIR) { $env:HOMEWEND_INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA "Programs\homewend" }

    $arch = switch ($env:PROCESSOR_ARCHITECTURE) {
        "AMD64" { "amd64" }
        "ARM64" { "arm64" }
        default { throw "homewend: no build for $env:PROCESSOR_ARCHITECTURE yet: amd64 and arm64 only" }
    }
    $binary = "homewend_windows_$arch.exe"
    $url = "https://github.com/$repo/releases/latest/download"
    $tmp = Join-Path ([System.IO.Path]::GetTempPath()) ([System.IO.Path]::GetRandomFileName())
    New-Item -ItemType Directory -Path $tmp | Out-Null

    try {
        Write-Host "downloading $binary"
        Invoke-WebRequest -Uri "$url/$binary" -OutFile (Join-Path $tmp "homewend.exe")
        Invoke-WebRequest -Uri "$url/homewend_checksums.txt" -OutFile (Join-Path $tmp "checksums.txt")

        $want = (Get-Content (Join-Path $tmp "checksums.txt") | Where-Object { $_ -match " $binary$" }) -replace " .*", ""
        if (-not $want) { throw "homewend: $binary is not in the release's checksums" }
        $got = (Get-FileHash -Algorithm SHA256 (Join-Path $tmp "homewend.exe")).Hash.ToLower()
        if ($got -ne $want) { throw "homewend: checksum mismatch: the download is damaged, nothing was installed" }

        New-Item -ItemType Directory -Path $dir -Force | Out-Null
        Move-Item -Force (Join-Path $tmp "homewend.exe") (Join-Path $dir "homewend.exe")
        $version = & (Join-Path $dir "homewend.exe") version
        Write-Host "installed $version to $dir\homewend.exe"
    } finally {
        Remove-Item -Recurse -Force $tmp
    }

    $path = [Environment]::GetEnvironmentVariable("Path", "User")
    if (($path -split ";") -notcontains $dir) {
        [Environment]::SetEnvironmentVariable("Path", "$path;$dir", "User")
        $env:Path = "$env:Path;$dir"
        Write-Host "added $dir to your PATH: open a new terminal, then: homewend login"
    } else {
        Write-Host "next: homewend login"
    }
}
