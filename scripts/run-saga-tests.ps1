# run-saga-tests.ps1
# Pokrece SAGA integration testove za otc-service.
# Pokrecanje: .\scripts\run-saga-tests.ps1
# Samo testovi (bez restarta baza): .\scripts\run-saga-tests.ps1 -TestOnly
# Samo odredjeni test: .\scripts\run-saga-tests.ps1 -Run TestSG01
# Sa Toxiproxy testovima: .\scripts\run-saga-tests.ps1 -ToxiproxyAddr localhost:8474

param(
    [switch]$TestOnly,
    [string]$Run = "TestSG",
    [string]$ToxiproxyAddr = ""
)

$Root = Split-Path $PSScriptRoot -Parent

$Services = @(
    "otc-service",
    "account-service",
    "portfolio-service",
    "securities-service",
    "exchange-service",
    "employee-service",
    "client-service"
)

$DbPorts = @{
    "otc-service"         = 5444
    "account-service"     = 5436
    "portfolio-service"   = 5443
    "securities-service"  = 5441
    "exchange-service"    = 5438
    "employee-service"    = 5433
    "client-service"      = 5435
}

function Wait-DbReady {
    param([string]$Service, [int]$Port)
    $attempts = 0
    while ($attempts -lt 30) {
        $conn = Test-NetConnection -ComputerName localhost -Port $Port -WarningAction SilentlyContinue
        if ($conn.TcpTestSucceeded) {
            Write-Host "  [$Service] ready on :$Port" -ForegroundColor Green
            return $true
        }
        Start-Sleep -Seconds 1
        $attempts++
    }
    Write-Host "  [$Service] TIMEOUT waiting on :$Port" -ForegroundColor Red
    return $false
}

if (-not $TestOnly) {
    Write-Host ""
    Write-Host "==> Pokrecanje DB kontejnera..." -ForegroundColor Cyan

    foreach ($svc in $Services) {
        $composePath = Join-Path $Root "services\$svc\docker-compose.yml"
        Write-Host "  Starting $svc..." -NoNewline
        docker compose -f $composePath up -d 2>&1 | Out-Null
        Write-Host " OK" -ForegroundColor Green
    }

    Write-Host ""
    Write-Host "==> Cekanje da baze budu spremne..." -ForegroundColor Cyan

    $allReady = $true
    foreach ($svc in $Services) {
        if ($DbPorts.ContainsKey($svc)) {
            $ok = Wait-DbReady -Service $svc -Port $DbPorts[$svc]
            if (-not $ok) { $allReady = $false }
        }
    }

    if (-not $allReady) {
        Write-Host ""
        Write-Host "Neke baze nisu startovale. Pokusaj rucno ili dodaj -TestOnly kad su vec up." -ForegroundColor Red
        exit 1
    }

    # Toxiproxy — pokrecemo ga samo kad nije TestOnly (jer startujemo sve od nule)
    if ($ToxiproxyAddr -ne "") {
        Write-Host ""
        Write-Host "==> Pokrecanje Toxiproxy..." -ForegroundColor Cyan

        # Ukloni stari kontejner ako postoji
        docker rm -f toxiproxy-saga-test 2>&1 | Out-Null

        # Pokreni Toxiproxy kontejner
        # Ports: 8474=REST API, 15436+15437=proxy portovi koje SG-09b/c koriste
        docker run -d --name toxiproxy-saga-test `
            -p 8474:8474 `
            -p 15436:15436 `
            -p 15437:15437 `
            ghcr.io/shopify/toxiproxy:2.12.0 2>&1 | Out-Null

        # Sacekaj da Toxiproxy bude spreman (GET /version vraca 200)
        $toxiReady = $false
        for ($i = 0; $i -lt 15; $i++) {
            try {
                $r = Invoke-WebRequest -Uri "http://localhost:8474/version" -UseBasicParsing -UserAgent "saga-test" -ErrorAction Stop
                if ($r.StatusCode -eq 200) { $toxiReady = $true; break }
            } catch {}
            Start-Sleep -Seconds 1
        }
        if (-not $toxiReady) {
            Write-Host "  Toxiproxy nije startovao na vreme!" -ForegroundColor Red
            exit 1
        }

        # Povezi Toxiproxy u istu mrezu kao account-db da mu moze pristupiti direktno.
        # Bez ovoga Toxiproxy ne moze doseci postgres koji je u drugom Docker networku.
        docker network connect account-service_default toxiproxy-saga-test 2>&1 | Out-Null

        Write-Host "  Toxiproxy ready on :8474 (connected to account-service_default)" -ForegroundColor Green
    }
}

Write-Host ""
Write-Host "==> Pokrecanje testova (-run $Run)..." -ForegroundColor Cyan
Write-Host ""

$env:OTC_INTEGRATION_TEST = "true"
$env:OTC_SAGA_TEST_HOOKS  = "true"
$env:OTC_DOCKER_TEST      = "true"

if ($ToxiproxyAddr -ne "") {
    $env:TOXIPROXY_ADDR = $ToxiproxyAddr
    # Toxiproxy je u Docker kontejneru povezanom na account-service_default mrezu,
    # pa moze da dosegne postgres direktno po imenu servisa (account-db:5432).
    $env:TOXI_ACCT_UPSTREAM = "account-db:5432"
    Write-Host "==> Toxiproxy: $ToxiproxyAddr (upstream: account-db:5432)" -ForegroundColor Cyan
}

Push-Location (Join-Path $Root "services\otc-service")
try {
    go test ./handlers/... -run $Run -v -timeout 300s
    $exitCode = $LASTEXITCODE
} finally {
    Pop-Location
}

# Cleanup Toxiproxy kontejnera
if ($ToxiproxyAddr -ne "" -and -not $TestOnly) {
    docker rm -f toxiproxy-saga-test 2>&1 | Out-Null
}

Write-Host ""
if ($exitCode -eq 0) {
    Write-Host "==> Svi testovi PROSLI" -ForegroundColor Green
} else {
    Write-Host "==> Neki testovi PALI (exit $exitCode)" -ForegroundColor Red
}

exit $exitCode
