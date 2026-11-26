# pathdoc installer. Downloads the latest release for this machine, checks
# its SHA-256 against the release's SHA256SUMS.txt (nothing is installed on a
# mismatch), puts it in %LOCALAPPDATA%\Programs\pathdoc and adds that folder
# to your user Path. Works in Windows PowerShell 5.1 and PowerShell 7:
#
#   irm https://raw.githubusercontent.com/Fd3os/pathdoc/main/install.ps1 | iex
#
# This file is ASCII on purpose: PowerShell 5.1 reads scripts in the ANSI
# code page, the very thing pathdoc is about. It runs in its own script
# block, so `iex` leaves nothing behind in your session but the Path entry.

& {
    $ErrorActionPreference = 'Stop'
    $ProgressPreference = 'SilentlyContinue'
    [Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12

    $arch = 'amd64'
    if ($env:PROCESSOR_ARCHITECTURE -eq 'ARM64' -or $env:PROCESSOR_ARCHITEW6432 -eq 'ARM64') { $arch = 'arm64' }
    $name = "pathdoc-windows-$arch.exe"
    $base = 'https://github.com/Fd3os/pathdoc/releases/latest/download'
    $dir = Join-Path $env:LOCALAPPDATA 'Programs\pathdoc'
    New-Item -ItemType Directory -Force -Path $dir | Out-Null

    $exe = Join-Path $dir "$name.download"
    $sums = Join-Path $dir 'SHA256SUMS.txt'
    Invoke-WebRequest -UseBasicParsing -Uri "$base/$name" -OutFile $exe
    Invoke-WebRequest -UseBasicParsing -Uri "$base/SHA256SUMS.txt" -OutFile $sums

    $want = ''
    foreach ($line in Get-Content $sums) {
        $parts = $line -split '\s+'
        if ($parts.Count -ge 2 -and $parts[1] -eq $name) { $want = $parts[0].ToLower() }
    }
    $got = (Get-FileHash $exe -Algorithm SHA256).Hash.ToLower()
    Remove-Item $sums
    if (-not $want -or $got -ne $want) {
        Remove-Item $exe
        throw "pathdoc: the download's SHA-256 ($got) does not match the release ($want). Nothing was installed."
    }
    Move-Item -Force $exe (Join-Path $dir 'pathdoc.exe')

    # Add the folder to the user Path, keeping %VAR% entries exactly as they are.
    $key = Get-Item 'HKCU:\Environment'
    $path = $key.GetValue('Path', '', 'DoNotExpandEnvironmentNames')
    $entries = @($path -split ';' | Where-Object { $_ -ne '' })
    $expanded = $entries | ForEach-Object { [Environment]::ExpandEnvironmentVariables($_).TrimEnd('\') }
    if ($expanded -notcontains $dir.TrimEnd('\')) {
        $new = (@($entries) + $dir) -join ';'
        Set-ItemProperty -Path 'HKCU:\Environment' -Name Path -Value $new -Type ExpandString
        Add-Type -Namespace PathdocInstall -Name Native -MemberDefinition @'
[DllImport("user32.dll", SetLastError = true, CharSet = CharSet.Unicode)]
public static extern IntPtr SendMessageTimeout(IntPtr hWnd, uint Msg, UIntPtr wParam, string lParam, uint fuFlags, uint uTimeout, out UIntPtr lpdwResult);
'@
        $result = [UIntPtr]::Zero
        [PathdocInstall.Native]::SendMessageTimeout([IntPtr]0xffff, 0x1a, [UIntPtr]::Zero, 'Environment', 2, 5000, [ref]$result) | Out-Null
    }
    # This terminal too, not only new ones.
    if (($env:Path -split ';' | ForEach-Object { $_.TrimEnd('\') }) -notcontains $dir.TrimEnd('\')) { $env:Path = "$env:Path;$dir" }

    & (Join-Path $dir 'pathdoc.exe') --version
    Write-Host "Installed to $dir. Run: pathdoc"
}
