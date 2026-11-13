# GoDeploy --- Go-Based DevOps Automation & Monitoring Platform

A local-first DevOps portfolio project demonstrating a Go application,
CI/CD automation, container-based deployment, metrics collection,
dashboards, and alert notifications.

> **Project scope:** GoDeploy v1 runs locally using Podman and Floci.
> The EC2 environment is a simulated EC2 instance running in a
> container; it is not a real AWS EC2 virtual machine. No public hosting
> or paid AWS infrastructure is required for this version.

## Architecture

``` mermaid
flowchart TB
    DEV["Developer"] --> GH["GitHub Repository<br/>GoDeploy"]
    GH --> J["Jenkins CI/CD"]
    J --> BUILD["Build and Validation<br/>Go format · Unit tests · go vet"]
    BUILD --> BIN["Build GoDeploy CLI<br/>and OpsPulse Linux binary"]
    BIN --> DEPLOY["SSH / SCP deployment"]
    DEPLOY --> EC2["Floci simulated EC2<br/>Podman container"]
    EC2 --> APP["OpsPulse Go application"]

    CLI["GoDeploy CLI"] --> POD["Local Podman<br/>container operations"]
    CLI --> S3["Floci S3 emulator<br/>deployment metadata"]

    APP --> HEALTH["Health endpoint"]
    APP --> METRICS["/metrics endpoint"]
    METRICS --> PROM["Prometheus<br/>metrics scraping"]
    PROM --> GRAF["Grafana dashboard"]
    PROM --> ALERT["Grafana alert rule"]
    ALERT --> WEB["Webhook.site<br/>test notification receiver"]
```

## Project components

  -----------------------------------------------------------------------
  Component                           Responsibility
  ----------------------------------- -----------------------------------
  GoDeploy CLI                        Go-based command-line tool for
                                      local Podman container operations
                                      and Floci S3 deployment metadata
                                      workflows

  GitHub                              Hosts the project source code

  Jenkins                             Checks out source, validates Go
                                      code, runs tests, builds binaries,
                                      and deploys OpsPulse

  Podman                              Runs the local containers used by
                                      the project

  Floci                               Provides local AWS-style services,
                                      including a simulated EC2
                                      environment and S3 emulator

  OpsPulse                            Go application deployed to the
                                      simulated EC2 environment

  Prometheus                          Scrapes OpsPulse metrics from
                                      `/metrics`

  Grafana                             Displays metrics and evaluates
                                      alert rules

  Webhook.site                        Receives test webhook notifications
                                      from Grafana
  -----------------------------------------------------------------------

## CI/CD workflow

1.  A developer pushes a change to the GitHub repository.
2.  Jenkins checks out the source code.
3.  The pipeline checks Go formatting.
4.  The pipeline runs Go unit tests.
5.  The pipeline runs `go vet`.
6.  Jenkins builds the GoDeploy CLI and the Linux OpsPulse binary.
7.  Jenkins transfers the OpsPulse binary to the Floci simulated EC2
    container using SSH/SCP.
8.  The remote deployment starts OpsPulse and checks its health
    endpoint.

The Jenkins deployment stage performs the remote SSH/SCP deployment. The
GoDeploy CLI's current deployment operations are for local Podman
containers; do not describe the CLI itself as performing the remote EC2
deployment.

## Monitoring and alerting

Prometheus scrapes the OpsPulse metrics endpoint every 15 seconds.

Configured metrics:

  --------------------------------------------------------------------------------
  Metric                           Type                    Purpose
  -------------------------------- ----------------------- -----------------------
  `godeploy_app_info`              Gauge                   Application name,
                                                           version, and
                                                           environment labels

  `godeploy_http_requests_total`   Counter                 Total homepage requests

  `godeploy_app_uptime_seconds`    Gauge                   Application uptime in
                                                           seconds
  --------------------------------------------------------------------------------

Prometheus target configuration (`monitoring/prometheus.yml`):

``` yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: "opspulse"
    metrics_path: /metrics
    static_configs:
      - targets: ["10.88.0.22:8080"]
```

> The target above is the current Podman network address used in this
> local setup. Container IP addresses can change after containers are
> recreated. If that happens, update the target to the current OpsPulse
> container address, or configure a stable network alias.

### Grafana dashboard

Dashboard: **GoDeploy - OpsPulse Monitoring**

Panels: - OpsPulse Application Uptime ---
`godeploy_app_uptime_seconds` - Total HTTP Requests ---
`godeploy_http_requests_total` - OpsPulse Application Info ---
`godeploy_app_info`

### Alert rule

-   Rule name: `OpsPulse-Application-Down`
-   Prometheus expression: `up{job="opspulse"}`
-   Condition: query result is below `1`
-   Evaluation group: `GoDeploy-Monitoring`
-   Evaluation interval: 30 seconds
-   Pending period: 1 minute
-   Notification contact point: `GoDeploy-Alerts`
-   Notification policy matcher: `alertname = OpsPulse-Application-Down`

Grafana sends webhook notifications to a temporary Webhook.site endpoint
for testing. Replace the temporary endpoint with a durable, secured
receiver before using this setup beyond a demonstration. Do not commit
webhook URLs, credentials, private keys, or other secrets to Git.

### Failure and recovery test

The alert lifecycle was manually verified:

1.  OpsPulse was running and Prometheus reported the target as `UP`.
2.  The OpsPulse process was stopped inside the simulated EC2 container.
3.  Prometheus detected the target as `DOWN`.
4.  Grafana changed `OpsPulse-Application-Down` to `Firing`.
5.  OpsPulse was restarted.
6.  Prometheus returned to `UP`.
7.  Grafana automatically returned the alert to `Normal`.

A Grafana test notification was also sent successfully and a
corresponding HTTP `POST` was observed at Webhook.site.

## Local environment

The project was developed on Windows using PowerShell, with Podman
containers and a Windows Jenkins inbound agent.

Known local endpoints:

  Service              Local URL / port
  -------------------- ---------------------------------
  Jenkins              `http://localhost:8080`
  Prometheus           `http://localhost:9090`
  Prometheus targets   `http://localhost:9090/targets`
  Grafana              `http://localhost:3001`
  Simulated EC2 SSH    `localhost:2201`

OpsPulse listens on port `8080` inside the simulated EC2 container. That
port is not currently published to the Windows host; Prometheus reaches
it over the Podman network.

## Repository layout

The project includes the Go application and tests, Jenkins pipeline,
GoDeploy CLI source, and monitoring configuration. The monitoring
configuration is stored at:

``` text
monitoring/
└── prometheus.yml
```

Review the actual repository tree and update this section with the final
directory listing before publishing the repository.

## Prerequisites

-   Windows with PowerShell
-   Go toolchain
-   Git
-   Podman
-   Jenkins controller and a connected Windows inbound agent
-   Floci local environment
-   Prometheus and Grafana containers

Versions and exact installation commands depend on the local
environment. Use the versions pinned or documented in the project
configuration rather than assuming `latest` images are reproducible.

## Running and validating

This repository's exact CLI subcommands and bootstrap scripts should be
documented from the current source and Jenkinsfile. Before publishing,
add the verified commands for:

-   Starting the required Podman/Floci services
-   Building and testing the Go application locally
-   Running the GoDeploy CLI
-   Starting or triggering the Jenkins pipeline
-   Checking OpsPulse health and metrics
-   Starting Prometheus and Grafana

Useful validation checks:

-   Jenkins latest pipeline build: `SUCCESS`
-   Prometheus target: `UP`
-   OpsPulse health endpoint: returns a healthy response
-   Grafana dashboard: all three panels populated
-   Alert rule: `Normal` when OpsPulse is healthy
-   Alert rule: `Firing` when OpsPulse is stopped
-   Alert rule: returns to `Normal` after OpsPulse is restored

## Troubleshooting notes

  -----------------------------------------------------------------------
  Symptom                             Checks
  ----------------------------------- -----------------------------------
  Jenkins pipeline fails              Review Jenkins console output,
                                      checkout, Go formatting, unit
                                      tests, `go vet`, and build stages

  SSH deployment fails                Confirm the simulated EC2 container
                                      is running, SSH daemon is started,
                                      port `2201` is reachable, and the
                                      pinned host key is current

  Prometheus target is `DOWN`         Check that OpsPulse is running,
                                      port `8080` is listening inside the
                                      container, and the Prometheus
                                      target address is correct

  Grafana shows no data               Confirm the Prometheus data source
                                      URL and run the metric queries
                                      directly in Prometheus

  Alert does not fire                 Check `up{job="opspulse"}`, the
                                      below-1 condition, evaluation
                                      interval, pending period, and
                                      notification policy matcher

  Webhook does not receive a          Test the contact point, verify the
  notification                        URL and POST method, and inspect
                                      the receiver's request history
  -----------------------------------------------------------------------

## Security and operational notes

-   This is a local learning and portfolio environment, not a production
    deployment.
-   Do not commit secrets, SSH private keys, access tokens, passwords,
    or temporary webhook URLs.
-   Use a durable webhook receiver with authentication and TLS for a
    real environment.
-   The Floci EC2 instance is simulated by a container; it does not
    provide the isolation or behavior of a full EC2 virtual machine.
-   Pin container image versions for repeatable builds.
-   Review firewall and port exposure before running services on a
    shared network.

## Resume project description

**GoDeploy --- Go-Based DevOps Automation and Monitoring Platform**

Built a Go-based DevOps portfolio project integrating GitHub, Jenkins,
Podman, and Floci to automate code validation, testing, binary builds,
and deployment of a Go application to a simulated EC2 environment using
SSH/SCP. Implemented Prometheus metrics scraping and Grafana dashboards
and alerting, then validated application outage detection and automatic
alert recovery through a failure/recovery simulation and webhook
delivery test.

## Future improvements

Potential future enhancements, kept outside the current v1 scope:

-   Replace the temporary webhook receiver with a persistent
    authenticated notification endpoint.
-   Use stable Podman network aliases instead of relying on a container
    IP.
-   Add a repeatable one-command local bootstrap and cleanup workflow.
-   Add integration tests for health and metrics endpoints.
-   Add release versioning and artifact retention.
-   Add a documented deployment rollback procedure.
