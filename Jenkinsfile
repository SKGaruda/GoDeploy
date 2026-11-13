
pipeline {
    agent {
        label 'windows'
    }

    options {
        skipDefaultCheckout(true)
        timestamps()
    }

    environment {
        EC2_HOST = '172.27.152.177'
        EC2_PORT = '2201'
        APP_PORT = '8080'
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

                    go version
                    if ($LASTEXITCODE -ne 0) {
                        throw "Go is unavailable."
                    }

                    git --version
                    if ($LASTEXITCODE -ne 0) {
                        throw "Git is unavailable."
                    }

                    podman --version
                    if ($LASTEXITCODE -ne 0) {
                        throw "Podman is unavailable."
                    }

                    podman info
                    if ($LASTEXITCODE -ne 0) {
                        throw "Podman engine is unavailable."
                    }

                    Write-Host "All tools verified."
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
                        throw "No Go files found."
                    }

                    Write-Host "Normalizing Go files to LF..."

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
                        & gofmt -l @(
                            $goFiles | ForEach-Object {
                                $_.FullName
                            }
                        )
                    )

                    if ($LASTEXITCODE -ne 0) {
                        throw "gofmt execution failed."
                    }

                    if ($unformatted.Count -gt 0) {
                        Write-Host "Files requiring formatting:"
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

                    Write-Host "All tests passed."
                '''
            }
        }

        stage('Static Analysis') {
            steps {
                powershell '''
                    $ErrorActionPreference = "Stop"

                    go vet ./...

                    if ($LASTEXITCODE -ne 0) {
                        throw "Go static analysis failed."
                    }

                    Write-Host "Go vet passed."
                '''
            }
        }

        stage('Build GoDeploy CLI') {
            steps {
                powershell '''
                    $ErrorActionPreference = "Stop"

                    New-Item -ItemType Directory -Force -Path .\\dist |
                        Out-Null

                    go build -trimpath `
                        -o .\\dist\\godeploy.exe `
                        .\\cmd\\godeploy

                    if ($LASTEXITCODE -ne 0) {
                        throw "GoDeploy CLI build failed."
                    }

                    if (-not (Test-Path .\\dist\\godeploy.exe)) {
                        throw "GoDeploy executable missing."
                    }

                    Get-Item .\\dist\\godeploy.exe |
                        Select-Object Name, Length
                '''
            }
        }

        stage('Build OpsPulse Linux Binary') {
            steps {
                powershell '''
                    $ErrorActionPreference = "Stop"

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
                        throw "Linux binary missing."
                    }

                    Get-Item .\\dist\\opspulse-linux-amd64 |
                        Select-Object Name, Length

                    Write-Host "Linux binary built successfully."
                '''
            }
        }

        stage('Deploy OpsPulse to Floci EC2') {
            steps {
                withCredentials([
                    sshUserPrivateKey(
                        credentialsId: 'floci-ec2-ssh',
                        keyFileVariable: 'SSH_KEY',
                        usernameVariable: 'SSH_USER'
                    )
                ]) {
                    powershell '''
                        $ErrorActionPreference = "Stop"

                        Write-Host "===== PREPARE DEPLOYMENT ====="

                        if (-not (Test-Path .\\dist\\opspulse-linux-amd64)) {
                            throw "OpsPulse Linux binary not found."
                        }

                        Write-Host "===== SECURE SSH KEY PERMISSIONS ====="

                        $who = & whoami.exe /user /fo csv /nh
                        $identity = ($who -join '') |
                            ConvertFrom-Csv -Header Account,SID

                        $agentSid = $identity.SID

                        if (-not $agentSid) {
                            throw "Could not identify Jenkins agent SID."
                        }

                        & icacls.exe $env:SSH_KEY /inheritance:r |
                            Out-Host

                        if ($LASTEXITCODE -ne 0) {
                            throw "Could not remove inherited key permissions."
                        }

                        & icacls.exe $env:SSH_KEY /remove:g `
                            "*S-1-1-0" `
                            "*S-1-5-11" `
                            "*S-1-5-32-545" |
                            Out-Host

                        & icacls.exe $env:SSH_KEY /grant:r "*${agentSid}:(R)" |
                            Out-Host

                        if ($LASTEXITCODE -ne 0) {
                            throw "Could not set SSH key permissions."
                        }

                        Write-Host "===== VALIDATE SSH PRIVATE KEY ====="

                        $null = & ssh-keygen.exe -y -f $env:SSH_KEY 2>&1

                        if ($LASTEXITCODE -ne 0) {
                            throw "SSH private key validation failed."
                        }

                        Write-Host "SSH private key validation successful."

                        Write-Host "===== CONFIGURE VERIFIED SSH HOST KEY ====="

                        $knownHosts = Join-Path $env:WORKSPACE "floci_known_hosts"

                        $hostKey = "[172.27.152.177]:2201 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIMUecn4BNx4oLRXFSY8kVnDnwG1zRlIaw6zM+fAR2Jj7"

                        [System.IO.File]::WriteAllText(
                            $knownHosts,
                            $hostKey + [Environment]::NewLine,
                            [System.Text.Encoding]::ASCII
                        )

                        $null = & ssh-keygen.exe -lf $knownHosts 2>&1

                        if ($LASTEXITCODE -ne 0) {
                            throw "Could not validate pinned SSH host key."
                        }

                        Write-Host "Verified ED25519 host key configured."

                        Write-Host "===== TRANSFER OPSPULSE BINARY ====="

                        & scp.exe `
                            -i $env:SSH_KEY `
                            -P $env:EC2_PORT `
                            -o BatchMode=yes `
                            -o StrictHostKeyChecking=yes `
                            -o "UserKnownHostsFile=$knownHosts" `
                            -o IdentitiesOnly=yes `
                            .\\dist\\opspulse-linux-amd64 `
                            "$($env:SSH_USER)@$($env:EC2_HOST):/tmp/opspulse-linux-amd64.new"

                        if ($LASTEXITCODE -ne 0) {
                            throw "SCP transfer failed."
                        }

                        Write-Host "Binary transferred successfully."

                        Write-Host "===== DEPLOY ON EC2 ====="

                        $remoteScript = @'
set -eu

APP_DIR="/opt/opspulse"
BIN="$APP_DIR/opspulse"
PIDFILE="$APP_DIR/opspulse.pid"
LOGFILE="$APP_DIR/opspulse.log"
UPLOAD="/tmp/opspulse-linux-amd64.new"

mkdir -p "$APP_DIR"

if [ ! -f "$UPLOAD" ]; then
    echo "Uploaded binary not found."
    exit 1
fi

echo "Checking previous OpsPulse process..."

if [ -f "$PIDFILE" ]; then
    old_pid="$(cat "$PIDFILE" 2>/dev/null || true)"

    case "$old_pid" in
        ''|*[!0-9]*)
            ;;
        *)
            if kill -0 "$old_pid" 2>/dev/null; then
                if [ -r "/proc/$old_pid/cmdline" ] &&
                   tr '\\000' ' ' < "/proc/$old_pid/cmdline" |
                   grep -F "$BIN" >/dev/null; then

                    echo "Stopping previous OpsPulse process..."
                    kill "$old_pid"

                    count=0
                    while kill -0 "$old_pid" 2>/dev/null; do
                        count=$((count + 1))
                        if [ "$count" -ge 10 ]; then
                            echo "Previous process did not stop."
                            exit 1
                        fi
                        sleep 1
                    done
                fi
            fi
            ;;
    esac
fi

echo "Installing new OpsPulse binary..."

cp "$UPLOAD" "$BIN"
chmod 755 "$BIN"
rm -f "$UPLOAD"

echo "Starting OpsPulse..."

nohup env \
    APP_NAME=OpsPulse \
    APP_VERSION=1.0.0 \
    APP_ENV=production \
    APP_PORT=8080 \
    "$BIN" > "$LOGFILE" 2>&1 &

new_pid=$!
echo "$new_pid" > "$PIDFILE"

echo "OpsPulse PID: $new_pid"

echo "Waiting for application health..."

count=0

while [ "$count" -lt 15 ]; do
    if ! kill -0 "$new_pid" 2>/dev/null; then
        echo "OpsPulse process exited."
        cat "$LOGFILE"
        exit 1
    fi

    if wget -q -O - http://127.0.0.1:8080/health; then
        echo ""
        echo "OpsPulse health check successful."
        exit 0
    fi

    count=$((count + 1))
    sleep 2
done

echo "OpsPulse health check failed."
cat "$LOGFILE"
exit 1
'@

                        $encodedScript = [Convert]::ToBase64String(
                            [System.Text.Encoding]::UTF8.GetBytes($remoteScript)
                        )

                        $remoteCommand = "echo $encodedScript | base64 -d | sh"

                        & ssh.exe `
                            -i $env:SSH_KEY `
                            -p $env:EC2_PORT `
                            -o BatchMode=yes `
                            -o StrictHostKeyChecking=yes `
                            -o "UserKnownHostsFile=$knownHosts" `
                            -o IdentitiesOnly=yes `
                            "$($env:SSH_USER)@$($env:EC2_HOST)" `
                            $remoteCommand

                        if ($LASTEXITCODE -ne 0) {
                            throw "Remote EC2 deployment or health check failed."
                        }

                        Write-Host "===== DEPLOYMENT SUCCESSFUL ====="
                    '''
                }
            }
        }
    }

    post {
        success {
            echo 'SUCCESS: GoDeploy CI/CD pipeline completed.'
        }

        failure {
            echo 'FAILURE: GoDeploy pipeline failed. Check Console Output.'
        }

        always {
            echo 'GoDeploy pipeline execution finished.'
        }
    }
}
