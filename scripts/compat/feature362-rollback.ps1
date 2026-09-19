[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$GuardBinary,

    [Parameter(Mandatory = $true)]
    [string]$GuardCommit,

    [string]$GuardSourceRoot,

    [string]$FeatureBinary,

    [switch]$GuardOnly
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..\..")).Path
if ([string]::IsNullOrWhiteSpace($GuardSourceRoot)) {
    $GuardSourceRoot = $repoRoot
}
$guardApiRoot = Join-Path (Resolve-Path $GuardSourceRoot).Path "src\api"
$apiRoot = Join-Path $repoRoot "src\api"

function Invoke-CheckedNative {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Executable,

        [Parameter(Mandatory = $true)]
        [string[]]$Arguments
    )

    & $Executable @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "$Executable exited with code $LASTEXITCODE"
    }
}

function Get-BinaryCommit {
    param(
        [Parameter(Mandatory = $true)]
        [string]$BinaryPath
    )

    $resolvedBinary = (Resolve-Path $BinaryPath).Path
    $commitPath = "$resolvedBinary.commit"
    if (-not (Test-Path -LiteralPath $commitPath -PathType Leaf)) {
        throw "Missing binary commit sidecar: $commitPath"
    }
    $commit = (Get-Content -LiteralPath $commitPath -Raw).Trim()
    if ($commit -notmatch "^[0-9a-fA-F]{40}$") {
        throw "Invalid binary commit sidecar: $commitPath"
    }
    return $commit.ToLowerInvariant()
}

function Assert-ContainsGuard {
    param(
        [Parameter(Mandatory = $true)]
        [string]$BinaryPath
    )

    $binaryCommit = Get-BinaryCommit -BinaryPath $BinaryPath
    Invoke-CheckedNative -Executable git -Arguments @("-C", $repoRoot, "cat-file", "-e", "$GuardCommit`^{commit}")
    & git -C $repoRoot merge-base --is-ancestor $GuardCommit $binaryCommit
    if ($LASTEXITCODE -ne 0) {
        throw "Binary commit $binaryCommit predates required compatibility guard $GuardCommit"
    }
    return $binaryCommit
}

$guardPath = (Resolve-Path $GuardBinary).Path
$guardBinaryCommit = Assert-ContainsGuard -BinaryPath $guardPath
$guardDigest = (Get-FileHash -LiteralPath $guardPath -Algorithm SHA256).Hash.ToLowerInvariant()

Push-Location $guardApiRoot
try {
    $guardPattern = "UnknownSource|UnknownSources|CoinCopilotSettingsDefaultsAndIndependentFallbacks"
    Invoke-CheckedNative -Executable go -Arguments @("test", "./handlers", "./repository", "./services", "-run", $guardPattern, "-count=1")
}
finally {
    Pop-Location
}

Write-Output "guard_commit=$guardBinaryCommit"
Write-Output "guard_sha256=$guardDigest"

if ($GuardOnly) {
    Write-Output "feature362_compatibility=guard-only-pass"
    return
}

if ([string]::IsNullOrWhiteSpace($FeatureBinary)) {
    throw "FeatureBinary is required unless GuardOnly is set"
}

$featurePath = (Resolve-Path $FeatureBinary).Path
$featureBinaryCommit = Assert-ContainsGuard -BinaryPath $featurePath
$featureDigest = (Get-FileHash -LiteralPath $featurePath -Algorithm SHA256).Hash.ToLowerInvariant()

function Invoke-CompatibilityTest {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Mode,

        [Parameter(Mandatory = $true)]
        [string]$DatabasePath,

        [Parameter(Mandatory = $true)]
        [string]$ArtifactPath
    )

    $env:FEATURE362_COMPAT_MODE = $Mode
    $env:FEATURE362_COMPAT_DB = $DatabasePath
    $env:FEATURE362_COMPAT_ARTIFACT = $ArtifactPath
    Push-Location $apiRoot
    try {
        Invoke-CheckedNative -Executable go -Arguments @(
            "test", "./integration",
            "-run", "^TestFeature362MixedBinaryDatabaseCompatibility$",
            "-count=1"
        )
    }
    finally {
        Pop-Location
    }
}

function Invoke-BinaryBootCheck {
    param(
        [Parameter(Mandatory = $true)]
        [string]$BinaryPath,

        [Parameter(Mandatory = $true)]
        [string]$DatabasePath,

        [Parameter(Mandatory = $true)]
        [string]$UploadPath,

        [Parameter(Mandatory = $true)]
        [string]$Label
    )

    $port = Get-Random -Minimum 20000 -Maximum 45000
    $stdout = Join-Path $compatRoot "$Label.stdout.log"
    $stderr = Join-Path $compatRoot "$Label.stderr.log"
    $env:DB_PATH = $DatabasePath
    $env:UPLOAD_DIR = $UploadPath
    $env:PORT = [string]$port
    $env:JWT_SECRET = "feature362-compatibility-secret-at-least-32-characters"
    $env:GIN_TRUSTED_PROXIES = "none"
    $process = Start-Process `
        -FilePath $BinaryPath `
        -PassThru `
        -NoNewWindow `
        -RedirectStandardOutput $stdout `
        -RedirectStandardError $stderr
    try {
        $deadline = (Get-Date).AddSeconds(20)
        do {
            if ($process.HasExited) {
                $errorText = if (Test-Path -LiteralPath $stderr) {
                    Get-Content -LiteralPath $stderr -Raw
                } else {
                    ""
                }
                throw "$Label binary exited before health check: $errorText"
            }
            try {
                $response = Invoke-WebRequest `
                    -Uri "http://127.0.0.1:$port/health" `
                    -TimeoutSec 2 `
                    -UseBasicParsing
                if ($response.StatusCode -eq 200) {
                    Write-Output "$Label`_health=200"
                    return
                }
            }
            catch {
                Start-Sleep -Milliseconds 250
            }
        } while ((Get-Date) -lt $deadline)
        throw "$Label binary did not become healthy within 20 seconds"
    }
    finally {
        if (-not $process.HasExited) {
            Stop-Process -Id $process.Id
            $process.WaitForExit()
        }
    }
}

$compatRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("feature362-compat-" + [guid]::NewGuid().ToString("N"))
$compatDatabase = Join-Path $compatRoot "compatibility.db"
$compatArtifact = Join-Path $compatRoot "fixture.bin"
$compatUploads = Join-Path $compatRoot "uploads"
New-Item -ItemType Directory -Path $compatRoot, $compatUploads | Out-Null
try {
    Invoke-CompatibilityTest -Mode "seed" -DatabasePath $compatDatabase -ArtifactPath $compatArtifact
    Invoke-BinaryBootCheck -BinaryPath $featurePath -DatabasePath $compatDatabase -UploadPath $compatUploads -Label "feature-before"
    Invoke-BinaryBootCheck -BinaryPath $guardPath -DatabasePath $compatDatabase -UploadPath $compatUploads -Label "guard"
    Invoke-CompatibilityTest -Mode "verify-guard" -DatabasePath $compatDatabase -ArtifactPath $compatArtifact
    Invoke-BinaryBootCheck -BinaryPath $featurePath -DatabasePath $compatDatabase -UploadPath $compatUploads -Label "feature-after"
    Invoke-CompatibilityTest -Mode "verify-feature" -DatabasePath $compatDatabase -ArtifactPath $compatArtifact
}
finally {
    Remove-Item -LiteralPath $compatRoot -Recurse -Force
    Remove-Item Env:\FEATURE362_COMPAT_MODE -ErrorAction SilentlyContinue
    Remove-Item Env:\FEATURE362_COMPAT_DB -ErrorAction SilentlyContinue
    Remove-Item Env:\FEATURE362_COMPAT_ARTIFACT -ErrorAction SilentlyContinue
}

Write-Output "feature_commit=$featureBinaryCommit"
Write-Output "feature_sha256=$featureDigest"
Write-Output "feature362_compatibility=full-pass"
