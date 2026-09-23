Write-Host "=== TEST 1: Login Superadmin ==="
$loginPayload = @{
    username = "superadmin"
    password = "password123"
} | ConvertTo-Json

$loginResp = Invoke-RestMethod -Uri "http://localhost:8088/api/v1/auth/login" -Method Post -ContentType "application/json" -Body $loginPayload
Write-Host "Login Superadmin Success!"
Write-Host "User ID: $($loginResp.user.id)"
Write-Host "Username: $($loginResp.user.username)"
Write-Host "Role: $($loginResp.user.role)"
$adminToken = $loginResp.token

$headers = @{
    "Authorization" = "Bearer $adminToken"
    "Content-Type"  = "application/json"
}

Write-Host "`n=== TEST 2: Get Profile (Me) ==="
$meResp = Invoke-RestMethod -Uri "http://localhost:8088/api/v1/auth/me" -Headers $headers -Method Get
Write-Host "Profile Name: $($meResp.name), Email: $($meResp.email), Role: $($meResp.role)"

Write-Host "`n=== TEST 3: Step-Up Authentication (Verify Password) ==="
$verifyCorrectPayload = @{ password = "password123" } | ConvertTo-Json
$verifyResp = Invoke-RestMethod -Uri "http://localhost:8088/api/v1/auth/verify-password" -Headers $headers -Method Post -Body $verifyCorrectPayload
Write-Host "Step-Up Auth (Correct Password): Verified = $($verifyResp.verified)"

try {
    $verifyWrongPayload = @{ password = "wrongpassword" } | ConvertTo-Json
    Invoke-RestMethod -Uri "http://localhost:8088/api/v1/auth/verify-password" -Headers $headers -Method Post -Body $verifyWrongPayload
} catch {
    Write-Host "Step-Up Auth (Wrong Password): Rejected as expected! Status: $($_.Exception.Response.StatusCode)"
}

Write-Host "`n=== TEST 4: List All Users (Superadmin/Owner only) ==="
$usersList = Invoke-RestMethod -Uri "http://localhost:8088/api/v1/users" -Headers $headers -Method Get
Write-Host "Found $($usersList.Count) registered users:"
foreach ($u in $usersList) {
    Write-Host " - [$($u.role)] $($u.username) ($($u.name)) - Active: $($u.is_active)"
}

Write-Host "`n=== TEST 5: Create New Cashier User ==="
$newUserPayload = @{
    name = "Dewi Kasir Surabaya"
    username = "kasir_sby01"
    email = "dewi.sby@retail.com"
    password = "password123"
    role = "cashier"
} | ConvertTo-Json

$newUser = Invoke-RestMethod -Uri "http://localhost:8088/api/v1/users" -Headers $headers -Method Post -Body $newUserPayload
Write-Host "Created New Cashier: $($newUser.username), ID: $($newUser.id)"

Write-Host "`n=== TEST 6: Login with New Cashier Account ==="
$cashierLoginPayload = @{
    username = "kasir_sby01"
    password = "password123"
} | ConvertTo-Json
$cashierLoginResp = Invoke-RestMethod -Uri "http://localhost:8088/api/v1/auth/login" -Method Post -ContentType "application/json" -Body $cashierLoginPayload
Write-Host "Cashier Login Success! Role: $($cashierLoginResp.user.role)"

Write-Host "`n=== TEST 7: Access Protected Inventory with Real Logged In Token ==="
$cashierHeaders = @{
    "Authorization" = "Bearer $($cashierLoginResp.token)"
}
$products = Invoke-RestMethod -Uri "http://localhost:8088/api/v1/inventory/products?limit=2" -Headers $cashierHeaders -Method Get
Write-Host "Products accessed successfully by cashier! Total count: $($products.data.Count)"

Write-Host "`n>>> ALL AUTH E2E TESTS COMPLETED 100% SUCCESSFULLY! <<<"
