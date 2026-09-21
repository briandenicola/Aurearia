$ErrorActionPreference = 'Stop'
$PSNativeCommandUseErrorActionPreference = $false
$root = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path
. (Join-Path $root '.specify\scripts\powershell\common.ps1')

foreach ($branch in @('beta', 'main', 'docs/delivery-validation', '2026031-143022-name', '20260319-143022')) {
    $result = @(Test-FeatureBranch -Branch $branch -HasGit $true)
    if ($result.Count -ne 1 -or $result[0] -isnot [bool] -or $result[0]) {
        throw "Invalid selection '$branch' did not return exactly one false Boolean"
    }
}
foreach ($branch in @('001-feature-name', '1234-feature-name', '20260319-143022-feature-name')) {
    $result = @(Test-FeatureBranch -Branch $branch -HasGit $true)
    if ($result.Count -ne 1 -or $result[0] -isnot [bool] -or -not $result[0]) {
        throw "Valid explicit selection '$branch' was rejected"
    }
}

$originalFeature = $env:SPECIFY_FEATURE
$temp = Join-Path ([IO.Path]::GetTempPath()) ('delivery-specify-' + [guid]::NewGuid())
try {
    New-Item -ItemType Directory -Path (Join-Path $temp '.specify\scripts\powershell') -Force | Out-Null
    New-Item -ItemType Directory -Path (Join-Path $temp 'specs\123-fixture') -Force | Out-Null
    Copy-Item -LiteralPath (Join-Path $root '.specify\scripts\powershell\common.ps1') -Destination (Join-Path $temp '.specify\scripts\powershell\common.ps1')
    Copy-Item -LiteralPath (Join-Path $root '.specify\scripts\powershell\check-prerequisites.ps1') -Destination (Join-Path $temp '.specify\scripts\powershell\check-prerequisites.ps1')
    foreach ($name in @('spec.md', 'plan.md', 'tasks.md')) {
        Set-Content -LiteralPath (Join-Path $temp "specs\123-fixture\$name") -Value '# Fixture'
    }
    git -C $temp init --quiet
    if ($LASTEXITCODE -ne 0) { throw 'Fixture git init failed' }
    Push-Location $temp
    try {
        $script = Join-Path $temp '.specify\scripts\powershell\check-prerequisites.ps1'
        $env:SPECIFY_FEATURE = 'beta'
        $result = & pwsh -NoProfile -File $script -Json -PathsOnly
        if ($LASTEXITCODE -eq 0 -or $result) { throw 'Invalid selection produced success-shaped paths' }
        $env:SPECIFY_FEATURE = '123-fixture'
        $result = & pwsh -NoProfile -File $script -Json -RequireTasks -IncludeTasks
        if ($LASTEXITCODE -ne 0) { throw 'Valid explicit feature prerequisites failed' }
        $parsed = $result | ConvertFrom-Json
        if ($parsed.FEATURE_DIR -ne (Join-Path $temp 'specs\123-fixture') -or 'tasks.md' -notin $parsed.AVAILABLE_DOCS) {
            throw 'Explicit feature returned wrong paths/tasks'
        }
        Remove-Item -LiteralPath (Join-Path $temp 'specs\123-fixture\tasks.md')
        $null = & pwsh -NoProfile -File $script -Json -RequireTasks
        if ($LASTEXITCODE -eq 0) { throw 'Missing tasks were accepted' }

        $common = Join-Path $temp '.specify\scripts\powershell\common.ps1'
        $source = Get-Content -LiteralPath $common -Raw
        $mutant = $source.Replace('[Console]::Error.WriteLine(', 'Write-Output (')
        if ($mutant -eq $source) { throw 'Mutation did not disable the stream-isolation guard' }
        Set-Content -LiteralPath $common -Value $mutant
        $env:SPECIFY_FEATURE = 'beta'
        $null = & pwsh -NoProfile -File $script -Json -PathsOnly
        if ($LASTEXITCODE -ne 0) { throw 'Negative control failed to reproduce the original Boolean-output bug' }
    } finally { Pop-Location }
} finally {
    $env:SPECIFY_FEATURE = $originalFeature
    Remove-Item -LiteralPath $temp -Recurse -Force
}
Write-Output 'SpecKit selection/stream regression and negative control passed.'
