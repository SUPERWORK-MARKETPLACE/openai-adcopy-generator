# Cross-compiles adcopy for all target platforms into ../bin.
$ErrorActionPreference = 'Stop'
Set-Location $PSScriptRoot
$targets = @(
    @{ GOOS = 'windows'; GOARCH = 'amd64'; Out = 'adcopy-windows-amd64.exe' },
    @{ GOOS = 'darwin';  GOARCH = 'arm64'; Out = 'adcopy-darwin-arm64' },
    @{ GOOS = 'darwin';  GOARCH = 'amd64'; Out = 'adcopy-darwin-amd64' }
)
New-Item -ItemType Directory -Force ..\bin | Out-Null
foreach ($t in $targets) {
    $env:GOOS = $t.GOOS; $env:GOARCH = $t.GOARCH; $env:CGO_ENABLED = '0'
    go build -trimpath -ldflags '-s -w' -o "..\bin\$($t.Out)" .\cmd\adcopy
    if ($LASTEXITCODE -ne 0) { throw "go build failed: $($t.GOOS)/$($t.GOARCH)" }
    Write-Host "built bin/$($t.Out)"
}
Remove-Item Env:GOOS, Env:GOARCH, Env:CGO_ENABLED -ErrorAction SilentlyContinue
