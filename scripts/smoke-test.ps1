#!/usr/bin/env pwsh
# CLI smoke test for SuperCSV validator
# Tests all spec examples and flag combinations

# Allow stderr output from validator without treating it as terminating error
$ErrorActionPreference = "Continue"

Write-Host "Building supercsv-validate..."
go build -o supercsv-validate.exe ./cmd/supercsv-validate
if ($LASTEXITCODE -ne 0) {
    Write-Error "Build failed"
    exit 1
}

Write-Host "`nChecking version..."
$versionOutput = & .\supercsv-validate.exe --version
Write-Host "  $versionOutput" -ForegroundColor Cyan
if ($LASTEXITCODE -ne 0) {
    Write-Error "Version check failed"
    exit 1
}

Write-Host "`nTesting spec examples..."
$specDir = "internal\v1_0\spec\examples"
$examples = Get-ChildItem -Path $specDir -Filter "*.supr"
$passed = 0
$failed = 0

foreach ($file in $examples) {
    Write-Host "  Testing $($file.Name)..." -NoNewline
    & .\supercsv-validate.exe $file.FullName 2>&1 | Out-Null
    if ($LASTEXITCODE -eq 0) {
        Write-Host " OK" -ForegroundColor Green
        $passed++
    }
    else {
        Write-Host " FAIL" -ForegroundColor Red
        $failed++
    }
}

Write-Host "`nTesting output formats..."

# Test JSON output
Write-Host "  Testing --json..." -NoNewline
& .\supercsv-validate.exe --json "internal\v1_0\spec\examples\people.supr" 2>&1 | Out-Null
if ($LASTEXITCODE -eq 0) {
    Write-Host " OK" -ForegroundColor Green
    $passed++
}
else {
    Write-Host " FAIL" -ForegroundColor Red
    $failed++
}

# Test default SUPR output format (no flags = supr format)
Write-Host "  Testing default supr format..." -NoNewline
$suprOutput = (& .\supercsv-validate.exe "internal\v1_0\spec\examples\people.supr" 2>&1) -join "`n"
if ($LASTEXITCODE -eq 0) {
    Write-Host " OK" -ForegroundColor Green
    $passed++
}
else {
    Write-Host " FAIL" -ForegroundColor Red
    $failed++
}

# Test quiet mode
Write-Host "  Testing --quiet..." -NoNewline
& .\supercsv-validate.exe --quiet "internal\v1_0\spec\examples\people.supr" 2>&1 | Out-Null
if ($LASTEXITCODE -eq 0) {
    Write-Host " OK" -ForegroundColor Green
    $passed++
}
else {
    Write-Host " FAIL" -ForegroundColor Red
    $failed++
}

# Test strict version (should fail on file without version directive)
Write-Host "  Testing --strict-version 1.0 (expect failure)..." -NoNewline
& .\supercsv-validate.exe --strict-version 1.0 "internal\v1_0\spec\examples\people.supr" 2>&1 | Out-Null
if ($LASTEXITCODE -ne 0) {
    Write-Host " OK" -ForegroundColor Green
    $passed++
}
else {
    Write-Host " FAIL (expected failure, got success)" -ForegroundColor Red
    $failed++
}

# Clean up
Remove-Item supercsv-validate.exe -ErrorAction SilentlyContinue

Write-Host "`n========================================="
Write-Host "SMOKE TEST RESULTS"
Write-Host "========================================="
Write-Host "Passed: $passed" -ForegroundColor Green
if ($failed -eq 0) {
    Write-Host "Failed: $failed" -ForegroundColor Green
}
else {
    Write-Host "Failed: $failed" -ForegroundColor Red
}
Write-Host "Total:  $($passed + $failed)"
Write-Host "========================================="

if ($failed -gt 0) {
    exit 1
}

Write-Host "`nAll smoke tests passed!" -ForegroundColor Green
exit 0
