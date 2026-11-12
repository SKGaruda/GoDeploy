package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"godeploy/internal/storage"
)

const (
	appName       = "GoDeploy"
	appVersion    = "1.0.0"
	imageName     = "godeploy-app:v1"
	containerName = "godeploy-app"
	appPort       = "8080"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	command := os.Args[1]

	switch command {
	case "version":
		fmt.Printf("%s version %s\n", appName, appVersion)

	case "doctor":
		runDoctor()

	case "build":
		runBuild()

	case "deploy":
		runDeploy()

	case "status":
		runStatus()

	case "history":
		runHistory()

	case "down":
		runDown()

	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("GoDeploy - Local Deployment Automation CLI")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  godeploy version   Show CLI version")
	fmt.Println("  godeploy doctor    Check required tools")
	fmt.Println("  godeploy build     Build application container image")
	fmt.Println("  godeploy deploy    Deploy application container")
	fmt.Println("  godeploy status    Show application deployment status")
	fmt.Println("  godeploy history   Show deployment history")
	fmt.Println("  godeploy down      Stop and remove application container")
}

func checkTool(name, command string, args ...string) {
	path, err := exec.LookPath(command)
	if err != nil {
		fmt.Printf("[FAIL] %s is not installed or not in PATH\n", name)
		return
	}

	cmd := exec.Command(command, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("[FAIL] %s found at %s but could not run: %v\n", name, path, err)
		return
	}

	fmt.Printf("[PASS] %s: %s", name, output)
}

func runDoctor() {
	fmt.Println("GoDeploy Environment Check")
	fmt.Println("--------------------------")

	checkTool("Go", "go", "version")
	checkTool("Git", "git", "--version")
	checkTool("Podman", "podman", "version")
}

func runBuild() {
	fmt.Println("GoDeploy: Building OpsPulse container image...")
	fmt.Printf("Image: %s\n", imageName)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(
		ctx,
		"podman",
		"build",
		"-t",
		imageName,
		"-f",
		"Containerfile",
		".",
	)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			fmt.Println("[FAIL] Container build timed out.")
		} else {
			fmt.Printf("[FAIL] Container build failed: %v\n", err)
		}
		os.Exit(1)
	}

	fmt.Println("[SUCCESS] Container image built successfully.")
}

func runDeploy() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	fmt.Println("GoDeploy: Starting deployment...")
	fmt.Printf("Image: %s\n", imageName)

	// Step 1: Verify that the required image exists.
	if err := runPodman(ctx, "image", "exists", imageName); err != nil {
		fmt.Printf(
			"[FAIL] Required image %s does not exist. Run 'godeploy build' first.\n",
			imageName,
		)
		os.Exit(1)
	}

	// Step 2: Check whether an existing container is available.
	oldContainerExists := runPodman(ctx, "container", "exists", containerName) == nil

	if oldContainerExists {
		output, err := podmanOutput(
			ctx,
			"inspect",
			"--format",
			"{{.ImageName}}",
			containerName,
		)

		if err != nil {
			fmt.Printf("[FAIL] Could not inspect existing container: %v\n", err)
			os.Exit(1)
		}

		oldImage := strings.TrimSpace(output)

		if oldImage != imageName && !strings.HasSuffix(oldImage, "/"+imageName) {
			fmt.Printf(
				"[FAIL] Existing container uses unexpected image %q. Leaving it untouched.\n",
				oldImage,
			)
			os.Exit(1)
		}

		fmt.Println("Existing GoDeploy container found. Preparing replacement...")

		if err := runPodman(ctx, "stop", containerName); err != nil {
			fmt.Printf("[FAIL] Could not stop existing container: %v\n", err)
			os.Exit(1)
		}

		if err := runPodman(
			ctx,
			"rename",
			containerName,
			containerName+"-rollback",
		); err != nil {
			fmt.Printf("[FAIL] Could not prepare rollback container: %v\n", err)
			_ = runPodman(ctx, "start", containerName)
			os.Exit(1)
		}
	}

	// Step 3: Start the new container.
	if err := runPodman(
		ctx,
		"run",
		"-d",
		"--name",
		containerName,
		"-p",
		appPort+":8080",
		imageName,
	); err != nil {
		fmt.Printf("[FAIL] New container could not start: %v\n", err)
		rollbackDeployment(ctx, oldContainerExists)
		os.Exit(1)
	}

	// Step 4: Verify application health.
	fmt.Println("Container started. Checking application health...")

	if err := waitForHealth(ctx); err != nil {
		fmt.Printf("[FAIL] Health check failed: %v\n", err)

		_ = runPodman(ctx, "stop", containerName)
		_ = runPodman(ctx, "rm", containerName)

		rollbackDeployment(ctx, oldContainerExists)
		os.Exit(1)
	}

	// Step 5: Remove the previous container after successful health check.
	if oldContainerExists {
		if err := runPodman(ctx, "rm", containerName+"-rollback"); err != nil {
			fmt.Printf(
				"[WARNING] Deployment succeeded, but old container cleanup failed: %v\n",
				err,
			)
		}
	}

	// Step 6: Save deployment metadata to Floci S3.
	// Metadata failure must not undo a healthy application deployment.
	saveDeploymentMetadata()

	fmt.Println("[SUCCESS] OpsPulse deployed successfully!")
	fmt.Println("Application: http://localhost:8080")
	fmt.Println("Health:      http://localhost:8080/health")
	fmt.Println("Metrics:     http://localhost:8080/metrics")
}

func rollbackDeployment(ctx context.Context, oldContainerExists bool) {
	if !oldContainerExists {
		fmt.Println("[INFO] No previous container was available for rollback.")
		return
	}

	fmt.Println("[INFO] Attempting rollback to previous container...")

	if err := runPodman(
		ctx,
		"rename",
		containerName+"-rollback",
		containerName,
	); err != nil {
		fmt.Printf("[FAIL] Could not restore previous container name: %v\n", err)
		return
	}

	if err := runPodman(ctx, "start", containerName); err != nil {
		fmt.Printf("[FAIL] Could not restart previous container: %v\n", err)
		return
	}

	fmt.Println("[SUCCESS] Previous container restored.")
}

func waitForHealth(ctx context.Context) error {
	client := &http.Client{
		Timeout: 3 * time.Second,
	}

	url := "http://localhost:" + appPort + "/health"

	for attempt := 1; attempt <= 15; attempt++ {
		request, err := http.NewRequestWithContext(
			ctx,
			http.MethodGet,
			url,
			nil,
		)

		if err == nil {
			response, err := client.Do(request)

			if err == nil {
				response.Body.Close()

				if response.StatusCode == http.StatusOK {
					return nil
				}
			}
		}

		fmt.Printf("Health check attempt %d/15: waiting...\n", attempt)

		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-time.After(2 * time.Second):
		}
	}

	return fmt.Errorf("application did not become healthy within 30 seconds")
}

func runPodman(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, "podman", args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf(
			"podman %s: %s: %w",
			strings.Join(args, " "),
			strings.TrimSpace(string(output)),
			err,
		)
	}

	return nil
}

func podmanOutput(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "podman", args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf(
			"podman %s: %s: %w",
			strings.Join(args, " "),
			strings.TrimSpace(string(output)),
			err,
		)
	}

	return string(output), nil
}

// Save successful deployment metadata in Floci S3.
func saveDeploymentMetadata() {
	fmt.Println("GoDeploy: Saving deployment metadata to Floci S3...")

	ctx, cancel := context.WithTimeout(
		context.Background(),
		15*time.Second,
	)
	defer cancel()

	store, err := storage.NewS3Storage(ctx)
	if err != nil {
		fmt.Printf(
			"[WARNING] Application is healthy, but metadata storage is unavailable: %v\n",
			err,
		)
		return
	}

	if err := store.EnsureBucket(ctx, storage.BucketName); err != nil {
		fmt.Printf(
			"[WARNING] Application is healthy, but S3 bucket check failed: %v\n",
			err,
		)
		return
	}

	record := storage.DeploymentRecord{
		Application: "OpsPulse",
		Version:     appVersion,
		Image:       imageName,
		Status:      "SUCCESS",
		Environment: "production",
		DeployedAt:  time.Now().UTC().Format(time.RFC3339Nano),
	}

	if err := store.SaveDeployment(ctx, record); err != nil {
		fmt.Printf(
			"[WARNING] Application is healthy, but metadata could not be saved: %v\n",
			err,
		)
		return
	}

	fmt.Println("[SUCCESS] Deployment metadata recorded in Floci S3.")
}

// Display the current application container and health status.
func runStatus() {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		15*time.Second,
	)
	defer cancel()

	fmt.Println("GoDeploy Application Status")
	fmt.Println("---------------------------")

	// Step 1: Check whether the container exists.
	if err := runPodman(ctx, "container", "exists", containerName); err != nil {
		fmt.Println("[INFO] GoDeploy application container does not exist.")
		return
	}

	// Step 2: Retrieve container state.
	state, err := podmanOutput(
		ctx,
		"inspect",
		"--format",
		"{{.State.Status}}",
		containerName,
	)
	if err != nil {
		fmt.Printf("[FAIL] Could not retrieve container status: %v\n", err)
		return
	}

	state = strings.TrimSpace(state)

	fmt.Printf("Container: %s\n", containerName)
	fmt.Printf("State:     %s\n", state)

	if state != "running" {
		fmt.Println("[WARNING] Application container is not running.")
		return
	}

	// Step 3: Check application health.
	client := &http.Client{
		Timeout: 3 * time.Second,
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		"http://localhost:"+appPort+"/health",
		nil,
	)
	if err != nil {
		fmt.Printf("[FAIL] Could not create health request: %v\n", err)
		return
	}

	response, err := client.Do(request)
	if err != nil {
		fmt.Printf("[FAIL] Application health check failed: %v\n", err)
		return
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		fmt.Printf("[FAIL] Application returned HTTP %d\n", response.StatusCode)
		return
	}

	fmt.Println("[SUCCESS] Application is healthy.")
	fmt.Println("URL: http://localhost:8080")
}

// Retrieve and display deployment history from Floci S3.
func runHistory() {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		30*time.Second,
	)
	defer cancel()

	fmt.Println("GoDeploy Deployment History")
	fmt.Println("---------------------------")

	store, err := storage.NewS3Storage(ctx)
	if err != nil {
		fmt.Printf("[FAIL] Could not connect to Floci S3: %v\n", err)
		return
	}

	records, err := store.ListDeployments(ctx)
	if err != nil {
		fmt.Printf("[FAIL] Could not retrieve deployment history: %v\n", err)
		return
	}

	if len(records) == 0 {
		fmt.Println("[INFO] No deployment records found.")
		return
	}

	fmt.Printf("Total deployments: %d\n\n", len(records))

	fmt.Printf(
		"%-25s %-12s %-12s %-14s %-12s %s\n",
		"DEPLOYED AT (UTC)",
		"APPLICATION",
		"VERSION",
		"ENVIRONMENT",
		"STATUS",
		"IMAGE",
	)

	fmt.Println(strings.Repeat("-", 105))

	for _, record := range records {
		fmt.Printf(
			"%-25s %-12s %-12s %-14s %-12s %s\n",
			record.DeployedAt,
			record.Application,
			record.Version,
			record.Environment,
			record.Status,
			record.Image,
		)
	}
}

// Stop and remove only the GoDeploy application container.
func runDown() {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		30*time.Second,
	)
	defer cancel()

	fmt.Println("GoDeploy: Stopping application...")

	// Step 1: Check whether the application container exists.
	if err := runPodman(ctx, "container", "exists", containerName); err != nil {
		fmt.Println("[INFO] GoDeploy application container does not exist.")
		return
	}

	// Step 2: Stop the application container.
	fmt.Println("[INFO] Stopping container:", containerName)

	if err := runPodman(ctx, "stop", containerName); err != nil {
		fmt.Printf("[FAIL] Could not stop application container: %v\n", err)
		return
	}

	fmt.Println("[SUCCESS] Application container stopped.")

	// Step 3: Remove only the GoDeploy application container.
	if err := runPodman(ctx, "rm", containerName); err != nil {
		fmt.Printf("[FAIL] Could not remove application container: %v\n", err)
		return
	}

	fmt.Println("[SUCCESS] Application container removed.")
	fmt.Println("[INFO] Floci and deployment history remain untouched.")
}
