<#
.SYNOPSIS
  Pull this project's .env from Infisical using the machine credential stored by
  cauldnz-pos/infra/identities/pos-auth.ps1 (DPAPI-encrypted under %LOCALAPPDATA%\pos\).

  Every secret in the project/environment is written as KEY="VALUE" (quoted, as
  .env.template requires) to .env in the repo root. Existing .env is overwritten.

.EXAMPLE
  .\scripts\fetch_env.ps1                          # project 'showcase-countdown', env 'dev'
  .\scripts\fetch_env.ps1 -Project showcase-countdown -VaultEnv prod
  .\scripts\fetch_env.ps1 -List                    # just show the keys, don't write
#>
[CmdletBinding()]
param(
  [string]$Name     = "chris-p1",             # stored credential label
  [string]$Project  = "showcase-countdown",   # Infisical project name (or id)
  [string]$VaultEnv = "dev",
  [string]$OutFile  = (Join-Path (Split-Path $PSScriptRoot -Parent) ".env"),
  [switch]$List
)
$ErrorActionPreference = "Stop"

$path = Join-Path (Join-Path $env:LOCALAPPDATA "pos") "$Name.cred.json"
if (-not (Test-Path $path)) { throw "No stored credential '$Name'. Run cauldnz-pos\infra\identities\pos-auth.ps1 first." }
$c = Get-Content $path -Raw | ConvertFrom-Json

$sec   = ConvertTo-SecureString $c.clientSecretEnc
$plain = [Runtime.InteropServices.Marshal]::PtrToStringAuto([Runtime.InteropServices.Marshal]::SecureStringToBSTR($sec))
$login = Invoke-RestMethod -Method Post -Uri "$($c.host)/api/v1/auth/universal-auth/login" -ContentType "application/json" `
          -Body (@{ clientId = $c.clientId; clientSecret = $plain } | ConvertTo-Json)
$hdr = @{ Authorization = "Bearer $($login.accessToken)" }

# Resolve project name -> id (accepts an id directly too).
$ws = (Invoke-RestMethod -Uri "$($c.host)/api/v1/workspace" -Headers $hdr).workspaces
$w  = $ws | Where-Object { $_.name -eq $Project -or $_.id -eq $Project } | Select-Object -First 1
if (-not $w) {
  $seen = ($ws | ForEach-Object { $_.name }) -join ', '
  throw "Identity '$Name' cannot see a project called '$Project'. Visible: $seen. Grant it on the NAS: ssh unraid `"sh /tmp/new-machine-identity.sh $Name --project $Project --write`""
}

$secrets = (Invoke-RestMethod -Uri "$($c.host)/api/v3/secrets/raw?workspaceId=$($w.id)&environment=$VaultEnv" -Headers $hdr).secrets
if (-not $secrets) { throw "No secrets in $($w.name)/$VaultEnv." }

if ($List) { $secrets | ForEach-Object { $_.secretKey }; return }

$utf8 = New-Object Text.UTF8Encoding $false

# Preferred: a single DOTENV secret holding the whole file, written verbatim.
$dotenv = $secrets | Where-Object { $_.secretKey -eq "DOTENV" } | Select-Object -First 1
if ($dotenv) {
  if (-not $dotenv.secretValue.Trim()) { throw "DOTENV in $($w.name)/$VaultEnv is empty. Paste the .env contents into it in Infisical." }
  $text = $dotenv.secretValue -replace "`r`n", "`n"
  if (-not $text.EndsWith("`n")) { $text += "`n" }
  [IO.File]::WriteAllText($OutFile, $text, $utf8)
  Write-Host "Wrote DOTENV ($(($text -split "`n" | Where-Object { $_ -match '^\s*[A-Z_]+=' }).Count) keys) to $OutFile" -ForegroundColor Green
  return
}

# Otherwise: one Infisical secret per key, quoted as .env.template requires.
$lines = @("# Generated from Infisical $($w.name)/$VaultEnv on $(Get-Date -Format s). Do not commit.")
foreach ($s in ($secrets | Sort-Object secretKey)) {
  $v = $s.secretValue -replace '\\', '\\' -replace '"', '\"'
  $lines += ('{0}="{1}"' -f $s.secretKey, $v)
}
[IO.File]::WriteAllLines($OutFile, $lines, $utf8)
Write-Host "Wrote $($secrets.Count) key(s) to $OutFile" -ForegroundColor Green
