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
    exit 0
}

if ([string]::IsNullOrWhiteSpace($FeatureBinary)) {
    throw "FeatureBinary is required unless GuardOnly is set"
}

$featurePath = (Resolve-Path $FeatureBinary).Path
$featureBinaryCommit = Assert-ContainsGuard -BinaryPath $featurePath
$featureDigest = (Get-FileHash -LiteralPath $featurePath -Algorithm SHA256).Hash.ToLowerInvariant()

Push-Location $apiRoot
try {
    $fullPattern = "Feature362MixedBinary|RollbackCompatibility|AttributionHandoff"
    $listed = & go test ./integration -list $fullPattern
    if ($LASTEXITCODE -ne 0) {
        throw "Unable to enumerate the Feature 362 mixed-binary tests"
    }
    if (-not ($listed | Select-String -Pattern "^Test(Feature362MixedBinary|RollbackCompatibility|AttributionHandoff)")) {
        throw "The post-schema mixed-binary suite is not present; full compatibility cannot pass yet"
    }
    Invoke-CheckedNative -Executable go -Arguments @("test", "./integration", "-run", $fullPattern, "-count=1")
}
finally {
    Pop-Location
}

Write-Output "feature_commit=$featureBinaryCommit"
Write-Output "feature_sha256=$featureDigest"
Write-Output "feature362_compatibility=full-pass"
