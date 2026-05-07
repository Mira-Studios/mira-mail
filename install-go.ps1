# Download and install Go 1.21.5
$goVersion = "1.21.5"
$downloadUrl = "https://go.dev/dl/go$goVersion.windows-amd64.msi"
$installerPath = "$env:TEMP\go-installer.msi"

Write-Host "Downloading Go $goVersion..."
Invoke-WebRequest -Uri $downloadUrl -OutFile $installerPath

Write-Host "Installing Go..."
Start-Process msiexec.exe -ArgumentList "/i", $installerPath, "/quiet", "/norestart" -Wait

Write-Host "Go installed! Restart your terminal."
Write-Host "Test with: go version"
