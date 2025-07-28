# StormDB Installation Script for Windows PowerShell
# Usage: iwr https://raw.githubusercontent.com/elchinoo/stormdb/main/install.ps1 | iex
# Or: iwr https://raw.githubusercontent.com/elchinoo/stormdb/main/install.ps1 | iex -Args "v0.1.0-alpha.2"

param(
    [string]$Version = "latest",
    [string]$InstallDir = "$env:ProgramFiles\StormDB"
)

# Configuration
$Repo = "elchinoo/stormdb"
$BinaryName = "stormdb.exe"

# Helper functions
function Write-Info {
    param([string]$Message)
    Write-Host "ℹ️  $Message" -ForegroundColor Blue
}

function Write-Success {
    param([string]$Message)
    Write-Host "✅ $Message" -ForegroundColor Green
}

function Write-Warning {
    param([string]$Message)
    Write-Host "⚠️  $Message" -ForegroundColor Yellow
}

function Write-Error {
    param([string]$Message)
    Write-Host "❌ $Message" -ForegroundColor Red
    exit 1
}

# Detect architecture
function Get-Architecture {
    $arch = $env:PROCESSOR_ARCHITECTURE
    switch ($arch) {
        "AMD64" { return "amd64" }
        "x86" { return "386" }
        default { Write-Error "Unsupported architecture: $arch" }
    }
}

# Get latest version from GitHub API
function Get-LatestVersion {
    $latestUrl = "https://api.github.com/repos/$Repo/releases/latest"
    try {
        $response = Invoke-RestMethod -Uri $latestUrl -Method Get
        return $response.tag_name
    }
    catch {
        Write-Error "Failed to fetch latest version: $_"
    }
}

# Check if version exists
function Test-VersionExists {
    param([string]$Version)
    $checkUrl = "https://api.github.com/repos/$Repo/releases/tags/$Version"
    try {
        Invoke-RestMethod -Uri $checkUrl -Method Get | Out-Null
        return $true
    }
    catch {
        return $false
    }
}

# Download and install binary
function Install-Binary {
    param(
        [string]$Architecture,
        [string]$Version
    )
    
    $platform = "windows-$Architecture"
    $downloadUrl = "https://github.com/$Repo/releases/download/$Version/stormdb-$platform.zip"
    $tempDir = New-TemporaryFile | ForEach-Object { Remove-Item $_; New-Item -ItemType Directory -Path $_ }
    $tempFile = Join-Path $tempDir "stormdb.zip"
    
    Write-Info "Downloading StormDB $Version for $platform..."
    
    try {
        # Download binary
        Invoke-WebRequest -Uri $downloadUrl -OutFile $tempFile
        
        # Extract binary
        Write-Info "Extracting binary..."
        Expand-Archive -Path $tempFile -DestinationPath $tempDir -Force
        
        # Create install directory
        Write-Info "Installing to $InstallDir..."
        if (!(Test-Path $InstallDir)) {
            New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
        }
        
        # Copy binary
        $binaryPath = Join-Path $tempDir $BinaryName
        if (Test-Path $binaryPath) {
            Copy-Item $binaryPath -Destination $InstallDir -Force
        } else {
            # Binary might be in a subdirectory
            $binaryPath = Get-ChildItem -Path $tempDir -Name $BinaryName -Recurse | Select-Object -First 1
            if ($binaryPath) {
                Copy-Item (Join-Path $tempDir $binaryPath) -Destination $InstallDir -Force
            } else {
                Write-Error "Binary not found in downloaded archive"
            }
        }
        
        Write-Success "StormDB installed successfully!"
    }
    catch {
        Write-Error "Installation failed: $_"
    }
    finally {
        # Clean up
        Remove-Item $tempDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}

# Add to PATH
function Add-ToPath {
    param([string]$Directory)
    
    $currentPath = [Environment]::GetEnvironmentVariable("PATH", "User")
    if ($currentPath -notlike "*$Directory*") {
        Write-Info "Adding $Directory to PATH..."
        $newPath = "$currentPath;$Directory"
        [Environment]::SetEnvironmentVariable("PATH", $newPath, "User")
        $env:PATH = "$env:PATH;$Directory"
        Write-Success "Added to PATH. Restart PowerShell or your terminal to use 'stormdb' command globally."
    } else {
        Write-Info "Directory already in PATH."
    }
}

# Verify installation
function Test-Installation {
    $binaryPath = Join-Path $InstallDir $BinaryName
    if (Test-Path $binaryPath) {
        try {
            $version = & $binaryPath --version 2>$null | Select-Object -First 1
            Write-Success "Installation verified: $version"
            Write-Info "Try running: stormdb --help"
        }
        catch {
            Write-Success "Binary installed at: $binaryPath"
            Write-Info "You can run it directly or add to PATH for global access."
        }
    } else {
        Write-Error "Installation verification failed. Binary not found at: $binaryPath"
    }
}

# Check administrator privileges
function Test-Administrator {
    $currentUser = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($currentUser)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

# Main installation process
function Main {
    Write-Host "🌟 StormDB Installation Script for Windows" -ForegroundColor Cyan
    Write-Host "==========================================" -ForegroundColor Cyan
    Write-Host ""
    
    # Check PowerShell version
    if ($PSVersionTable.PSVersion.Major -lt 3) {
        Write-Error "PowerShell 3.0 or higher is required. Please upgrade PowerShell."
    }
    
    # Detect architecture
    $architecture = Get-Architecture
    Write-Info "Detected architecture: $architecture"
    
    # Determine version
    if ($Version -eq "latest") {
        Write-Info "Fetching latest version..."
        $installVersion = Get-LatestVersion
        Write-Info "Latest version: $installVersion"
    } else {
        $installVersion = $Version
        Write-Info "Installing version: $installVersion"
        
        # Check if version exists
        if (!(Test-VersionExists $installVersion)) {
            Write-Error "Version $installVersion not found. Please check https://github.com/$Repo/releases"
        }
    }
    
    # Check privileges for system-wide install
    if ($InstallDir.StartsWith($env:ProgramFiles)) {
        if (!(Test-Administrator)) {
            Write-Warning "Installing to $InstallDir requires administrator privileges."
            Write-Info "Either run as administrator or specify a user directory with -InstallDir parameter."
            Write-Info "Example: -InstallDir `"$env:LOCALAPPDATA\StormDB`""
        }
    }
    
    # Check if already installed
    $existingBinary = Join-Path $InstallDir $BinaryName
    if (Test-Path $existingBinary) {
        try {
            $currentVersion = & $existingBinary --version 2>$null | Select-Object -First 1
            Write-Warning "StormDB is already installed: $currentVersion"
        }
        catch {
            Write-Warning "StormDB binary already exists at: $existingBinary"
        }
        
        $response = Read-Host "Do you want to continue with installation? [y/N]"
        if ($response -notmatch '^[yY]') {
            Write-Info "Installation cancelled."
            exit 0
        }
    }
    
    # Install binary
    Install-Binary $architecture $installVersion
    
    # Add to PATH
    Add-ToPath $InstallDir
    
    # Verify installation
    Test-Installation
    
    Write-Host ""
    Write-Host "🎉 Installation completed!" -ForegroundColor Green
    Write-Host ""
    Write-Host "📖 Next steps:"
    Write-Host "  1. Restart PowerShell to use 'stormdb' command globally"
    Write-Host "  2. Run 'stormdb --help' to see available options"
    Write-Host "  3. Download sample configs: https://github.com/$Repo/tree/main/config"
    Write-Host "  4. Read the docs: https://github.com/$Repo/blob/main/README.md"
    Write-Host ""
    Write-Host "🐛 Need help? https://github.com/$Repo/issues"
}

# Run main function
Main
