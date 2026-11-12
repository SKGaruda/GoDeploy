pipeline {
    agent {
        label 'windows'
    }

    options {
        timestamps()
        disableConcurrentBuilds()
    }

    stages {
        stage('Checkout') {
            steps {
                checkout scm
            }
        }

        stage('Verify Tools') {
            steps {
                powershell '''
                    $ErrorActionPreference = "Stop"

                    Write-Host "Checking Go..."
                    go version
                    if ($LASTEXITCODE -ne 0) {
                        throw "Go is unavailable."
                    }

                    Write-Host "Checking Git..."
                    git --version
                    if ($LASTEXITCODE -ne 0) {
                        throw "Git is unavailable."
                    }

                    Write-Host "Checking Podman..."
                    podman info --format "Podman engine is reachable"
                    if ($LASTEXITCODE -ne 0) {
                        throw "Podman engine is unavailable."
                    }
                '''
            }
        }

        stage('Go Format Check') {
            steps {
                powershell '''
                    $ErrorActionPreference = "Stop"

                    $goFiles = Get-ChildItem -Path . -Recurse -Filter *.go |
                        Where-Object {
                            $_.FullName -notmatch '\\\\.git\\\\|\\\\vendor\\\\'
                        } |
                        ForEach-Object { $_.FullName }

                    if (-not $goFiles) {
                        throw "No Go source files found."
                    }

                    $unformatted = gofmt -l $goFiles

                    if ($LASTEXITCODE -ne 0) {
                        throw "gofmt execution failed."
                    }

                    if ($unformatted) {
                        Write-Host "Files requiring gofmt:"
                        $unformatted
                        throw "Go formatting check failed."
                    }

                    Write-Host "All Go files are formatted."
                '''
            }
        }

        stage('Unit Tests') {
            steps {
                powershell '''
                    $ErrorActionPreference = "Stop"

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

                    go vet ./...

                    if ($LASTEXITCODE -ne 0) {
                        throw "Go vet detected an issue."
                    }

                    Write-Host "Go static analysis passed."
                '''
            }
        }

        stage('Build GoDeploy CLI') {
            steps {
                powershell '''
                    $ErrorActionPreference = "Stop"

                    New-Item -ItemType Directory -Force -Path .\\dist |
                        Out-Null

                    go build -trimpath -o .\\dist\\godeploy.exe .\\cmd\\godeploy

                    if ($LASTEXITCODE -ne 0) {
                        throw "GoDeploy CLI build failed."
                    }

                    Write-Host "GoDeploy CLI built successfully."
                '''
            }
        }

        stage('Build OpsPulse Linux Binary') {
            steps {
                powershell '''
                    $ErrorActionPreference = "Stop"

                    $env:GOOS = "linux"
                    $env:GOARCH = "amd64"
                    $env:CGO_ENABLED = "0"

                    go build -trimpath -ldflags="-s -w" `
                        -o .\\dist\\opspulse-linux-amd64 .\\app

                    if ($LASTEXITCODE -ne 0) {
                        throw "OpsPulse Linux build failed."
                    }

                    Write-Host "OpsPulse Linux binary built successfully."

                    Get-ChildItem .\\dist
                '''
            }
        }
    }

    post {
        success {
            echo 'GoDeploy CI pipeline completed successfully.'
        }

        failure {
            echo 'GoDeploy CI pipeline failed. Check the stage logs.'
        }
    }
}
