
pipeline {
    agent {
        label 'windows'
    }

    options {
        skipDefaultCheckout(true)
        timestamps()
    }

    stages {

        stage('Checkout') {
            steps {
                echo 'Checking out GoDeploy source code...'
                checkout scm
            }
        }

        stage('Verify Tools') {
            steps {
                powershell '''
                    $ErrorActionPreference = "Stop"

                    Write-Host "===== GO VERSION ====="
                    go version
                    if ($LASTEXITCODE -ne 0) {
                        throw "Go is not available."
                    }

                    Write-Host "===== GIT VERSION ====="
                    git --version
                    if ($LASTEXITCODE -ne 0) {
                        throw "Git is not available."
                    }

                    Write-Host "===== PODMAN VERSION ====="
                    podman --version
                    if ($LASTEXITCODE -ne 0) {
                        throw "Podman is not available."
                    }

                    Write-Host "===== PODMAN ENGINE ====="
                    podman info
                    if ($LASTEXITCODE -ne 0) {
                        throw "Podman engine is not reachable."
                    }

                    Write-Host "All required tools are available."
                '''
            }
        }

        stage('Go Format Check') {
            steps {
                powershell '''
                    $ErrorActionPreference = "Stop"

                    $goFiles = @(
                        Get-ChildItem -Path . -Recurse -Filter *.go |
                        Where-Object {
                            $_.FullName -notmatch '\\\\.git\\\\|\\\\vendor\\\\'
                        }
                    )

                    if ($goFiles.Count -eq 0) {
                        throw "No Go source files found."
                    }

                    Write-Host "Found $($goFiles.Count) Go files."

                    Write-Host "Normalizing Go files to LF line endings..."

                    foreach ($file in $goFiles) {
                        $content = [System.IO.File]::ReadAllText(
                            $file.FullName
                        )

                        $content = $content -replace "`r`n", "`n"

                        [System.IO.File]::WriteAllText(
                            $file.FullName,
                            $content,
                            [System.Text.UTF8Encoding]::new($false)
                        )
                    }

                    Write-Host "Running gofmt..."

                    $unformatted = @(
                        & gofmt -l @($goFiles | ForEach-Object {
                            $_.FullName
                        })
                    )

                    if ($LASTEXITCODE -ne 0) {
                        throw "gofmt execution failed."
                    }

                    if ($unformatted.Count -gt 0) {
                        Write-Host "Files requiring gofmt:"
                        $unformatted
                        throw "Go formatting check failed."
                    }

                    Write-Host "All Go files are formatted successfully."
                '''
            }
        }

        stage('Unit Tests') {
            steps {
                powershell '''
                    $ErrorActionPreference = "Stop"

                    Write-Host "Running Go unit tests..."

                    go test -v ./...

                    if ($LASTEXITCODE -ne 0) {
                        throw "Go unit tests failed."
                    }

                    Write-Host "All Go tests passed."
                '''
            }
        }

        stage('Static Analysis') {
            steps {
                powershell '''
                    $ErrorActionPreference = "Stop"

                    Write-Host "Running go vet..."

                    go vet ./...

                    if ($LASTEXITCODE -ne 0) {
                        throw "Go static analysis failed."
                    }

                    Write-Host "Go vet completed successfully."
                '''
            }
        }

        stage('Build GoDeploy CLI') {
            steps {
                powershell '''
                    $ErrorActionPreference = "Stop"

                    New-Item -ItemType Directory -Force -Path .\\dist |
                        Out-Null

                    Write-Host "Building GoDeploy CLI..."

                    go build -trimpath `
                        -o .\\dist\\godeploy.exe `
                        .\\cmd\\godeploy

                    if ($LASTEXITCODE -ne 0) {
                        throw "GoDeploy CLI build failed."
                    }

                    if (-not (Test-Path .\\dist\\godeploy.exe)) {
                        throw "GoDeploy executable was not generated."
                    }

                    Get-Item .\\dist\\godeploy.exe |
                        Select-Object Name, Length

                    Write-Host "GoDeploy CLI build successful."
                '''
            }
        }

        stage('Build OpsPulse Linux Binary') {
            steps {
                powershell '''
                    $ErrorActionPreference = "Stop"

                    Write-Host "Building OpsPulse for Linux AMD64..."

                    $previousGOOS = $env:GOOS
                    $previousGOARCH = $env:GOARCH
                    $previousCGO = $env:CGO_ENABLED

                    try {
                        $env:GOOS = "linux"
                        $env:GOARCH = "amd64"
                        $env:CGO_ENABLED = "0"

                        go build -trimpath `
                            -ldflags="-s -w" `
                            -o .\\dist\\opspulse-linux-amd64 `
                            .\\app

                        if ($LASTEXITCODE -ne 0) {
                            throw "OpsPulse Linux build failed."
                        }
                    }
                    finally {
                        $env:GOOS = $previousGOOS
                        $env:GOARCH = $previousGOARCH
                        $env:CGO_ENABLED = $previousCGO
                    }

                    if (-not (Test-Path .\\dist\\opspulse-linux-amd64)) {
                        throw "OpsPulse Linux binary was not generated."
                    }

                    Get-Item .\\dist\\opspulse-linux-amd64 |
                        Select-Object Name, Length

                    Write-Host "OpsPulse Linux build successful."
                '''
            }
        }
    }

    post {
        success {
            echo 'SUCCESS: GoDeploy CI pipeline completed successfully.'
        }

        failure {
            echo 'FAILURE: GoDeploy CI pipeline failed. Check the stage logs.'
        }

        always {
            echo 'GoDeploy CI pipeline execution finished.'
        }
    }
}
