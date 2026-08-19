$ErrorActionPreference = 'Stop'

if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
    Write-Error 'Docker Desktop is required. Install it from https://www.docker.com/products/docker-desktop/'
}

docker compose version | Out-Null

Write-Host 'SnipVault local setup' -ForegroundColor Green
Write-Host 'Your database and session secrets will be generated automatically.'

if (-not (Test-Path -LiteralPath '.env.docker')) {
    $randomBytes = New-Object byte[] 32
    [Security.Cryptography.RandomNumberGenerator]::Fill($randomBytes)
    $postgresPassword = [Convert]::ToHexString($randomBytes).ToLowerInvariant()
    [Security.Cryptography.RandomNumberGenerator]::Fill($randomBytes)
    $jwtSecret = [Convert]::ToHexString($randomBytes).ToLowerInvariant()

    $environment = @(
        "POSTGRES_PASSWORD=$postgresPassword"
        "JWT_SECRET=$jwtSecret"
    )

    Set-Content -LiteralPath '.env.docker' -Value $environment -Encoding utf8NoBOM
} else {
    Write-Host 'Existing .env.docker found; keeping current data secrets.'
}
docker compose --env-file .env.docker up --build -d

Write-Host ''
Write-Host 'SnipVault is ready: http://localhost:8090' -ForegroundColor Green
Write-Host 'Local email inbox: http://localhost:8025' -ForegroundColor Green
Write-Host 'Stop it with: docker compose --env-file .env.docker down'
