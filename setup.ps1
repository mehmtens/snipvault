$ErrorActionPreference = 'Stop'

if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
    Write-Error 'Docker Desktop is required. Install it from https://www.docker.com/products/docker-desktop/'
}

docker compose version | Out-Null

Write-Host 'SnipVault local setup' -ForegroundColor Green
Write-Host 'Your database and session secrets will be generated automatically.'

$brevoKeySecure = Read-Host 'Brevo API key' -AsSecureString
$brevoKeyPointer = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($brevoKeySecure)
try {
    $brevoKey = [Runtime.InteropServices.Marshal]::PtrToStringBSTR($brevoKeyPointer)
} finally {
    [Runtime.InteropServices.Marshal]::ZeroFreeBSTR($brevoKeyPointer)
}

$senderEmail = Read-Host 'Verified Brevo sender email'
if ([string]::IsNullOrWhiteSpace($brevoKey) -or $senderEmail -notmatch '^[^@\s]+@[^@\s]+\.[^@\s]+$') {
    Write-Error 'A Brevo API key and valid sender email are required.'
}

$randomBytes = New-Object byte[] 32
[Security.Cryptography.RandomNumberGenerator]::Fill($randomBytes)
$postgresPassword = [Convert]::ToHexString($randomBytes).ToLowerInvariant()
[Security.Cryptography.RandomNumberGenerator]::Fill($randomBytes)
$jwtSecret = [Convert]::ToHexString($randomBytes).ToLowerInvariant()

$environment = @(
    "POSTGRES_PASSWORD=$postgresPassword"
    "JWT_SECRET=$jwtSecret"
    "BREVO_API_KEY=$brevoKey"
    "MAIL_FROM_EMAIL=$senderEmail"
    'MAIL_FROM_NAME=SnipVault'
)

Set-Content -LiteralPath '.env.docker' -Value $environment -Encoding utf8NoBOM
docker compose --env-file .env.docker up --build -d

Write-Host ''
Write-Host 'SnipVault is ready: http://localhost:8090' -ForegroundColor Green
Write-Host 'Stop it with: docker compose --env-file .env.docker down'
