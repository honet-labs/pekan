param(
    [string]$DatabaseUrl = $env:DATABASE_URL
)

if ([string]::IsNullOrWhiteSpace($DatabaseUrl)) {
    Write-Error "DATABASE_URL is required"
    exit 1
}

if (-not (Get-Command psql -ErrorAction SilentlyContinue)) {
    Write-Error "psql command not found. Install PostgreSQL client tools first."
    exit 1
}

$root = Resolve-Path (Join-Path $PSScriptRoot "..")
$seedFile = Join-Path $root "seeds/001_demo_tenant.sql"

if (-not (Test-Path $seedFile)) {
    Write-Error "Seed file not found: $seedFile"
    exit 1
}

Write-Host "Applying demo seed: 001_demo_tenant.sql"
psql $DatabaseUrl -v ON_ERROR_STOP=1 -f $seedFile
if ($LASTEXITCODE -ne 0) {
    Write-Error "Demo seed failed"
    exit $LASTEXITCODE
}

Write-Host "Demo seed applied successfully."
