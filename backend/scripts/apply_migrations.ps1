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
$migrationsDir = Join-Path $root "migrations"

Write-Host "Applying migrations from: $migrationsDir"

Get-ChildItem -Path $migrationsDir -Filter *.sql | Sort-Object Name | ForEach-Object {
    Write-Host "-> $($_.Name)"
    psql $DatabaseUrl -v ON_ERROR_STOP=1 -f $_.FullName
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Migration failed: $($_.Name)"
        exit $LASTEXITCODE
    }
}

Write-Host "All migrations applied successfully."
