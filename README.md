## Overview

The **Dockerfile Sources Tool** is a production-grade utility written in Go as part of the Red Hat SRE Challenge. The tool performs the following tasks:

- **Download**: Retrieves a plaintext file (from a given URL) where each line contains a GitHub repository URL and a commit SHA.
- **Validation**: Validates each line, ignoring any invalid entries.
- **Cloning & Checkout**: Clones each valid repository and checks out the specified commit.
- **Analysis**: Recursively searches for all Dockerfiles within the repository and extracts the base image names from `FROM` statements.
- **Aggregation**: Aggregates the data into a JSON object, including any errors encountered during processing.
- **Output**: Prints the JSON to standard output.

This tool is built with maintainability, fault tolerance, scalability, and testability in mind. It is containerized using Docker and is designed to run as a Kubernetes Job for cloud-native environments.

## Project Structure
dockerfile-sources/ ├── main.go # Main source code (Go) ├── go.mod # Go module file ├── go.sum # Go module checksum file (may be empty if only standard library is used) ├── Dockerfile # Dockerfile to build the container image ├── configmap.yaml # Kubernetes configmap ├── job.yaml # Kubernetes Job manifest for deployment ├── README.md # Project documentation (this file) 


## Features

- **Robust Input Validation**: Uses regular expressions to verify repository URL and commit SHA format.
- **Fault Tolerance**: Processes each repository independently and aggregates errors.
- **Cloud-Native**: Packaged as a Docker image and deployable via a Kubernetes Job.
- **Modular & Testable**: Well-structured code to facilitate unit and integration testing.

## Getting Started

### Prerequisites

- [Go 1.21](https://golang.org/dl/)
- [Docker](https://www.docker.com/)
- [Minikube](https://minikube.sigs.k8s.io/docs/start/) (for local Kubernetes testing) or access to another Kubernetes cluster
- [kubectl](https://kubernetes.io/docs/tasks/tools/)

### Running Locally

1. **Clone the Repository**:
   ```bash
   git clone https://github.com/Viveniac/dockerfile-sources.git
   cd dockerfile-sources

2. **Run the Tool with Go: You can run the tool directly using:**:
    ```bash
    REPOSITORY_LIST_URL=https://your-source-file-url go run main.go

    example url: https://gist.githubusercontent.com/jmelis/c60e61a893248244dc4fa12b946585c4/raw/25d39f67f2405330a6314cad64fac423a171162c/sources.txt

### Docker

- **Building the Docker Image from root directory**
```bash
    docker build -t viveniac/2025:latest .
```
- **Run the Container Locally**
```bash
    docker run --rm -it -e REPOSITORY_LIST_URL=https://your-source-file-url viveniac/2025:latest
    #This will run the container and print the JSON output.
```

### Kubernetes

**ConfigMap for Configuration Management**
 - *A ConfigMap is used to externalize the REPOSITORY_LIST_URL so that configuration can be updated without rebuilding the container image. This practice aligns with production best practices by separating configuration from code.*

**Deploying as a Kubernetes Job**
1. **Start Minikube**
```bash
    minikube start --driver=docker
```
2. **Deploy the ConfigMap**
```bash
   kubectl apply -f configmap.yaml
```
3. **Deploy the job**
```bash
   kubectl apply -f job.yaml
```
3. **Monitor the job**
```bash
# Check job status:
    kubectl get jobs
# Get pod names:
    kubectl get pods
#  View logs (replace <pod-name> with your pod’s name):
    kubectl logs <pod-name>
```

Additional Enhancements

    Concurrency:
    Future versions could process multiple repositories in parallel using goroutines.

    Enhanced Logging & Metrics:
    Consider integrating structured logging and monitoring (e.g., Prometheus) for production environments.

    Testing:
    The code is designed for unit and integration testing using Go’s testing framework.

Production Readiness Considerations

      Multi-Architecture Build (Cross-Compilation)

      To ensure the image runs on various hardware platforms, we use Docker Buildx for cross-compilation. This process            builds multi-architecture images and pushes a unified manifest to Docker Hub, so the correct image variant is               automatically pulled on each node.
      Commands for Multi-Architecture Build

    1.) Create and Use a Builder with Docker-Container Driver:
    ```bash
    docker buildx create --name multi-builder --driver docker-container --use
    docker buildx inspect multi-builder --bootstrap
    ```
    2.) Build and Push the Image for Multiple Platforms:
    ````bash
    docker buildx build --platform linux/amd64,linux/arm64,linux/arm/v7 -t blahblah:latest --push .
    ````


    Use of Go Standard Library:
    All imported packages are part of the Go standard library, ensuring stability, maintenance, and security through FOSS.      This minimizes external dependency risks and simplifies audits.

    Separation of Configuration:
    The use of a ConfigMap (dockerfile-sources-config) externalizes environment-specific settings (like                   REPOSITORY_LIST_URL), allowing configuration changes without rebuilding the image. This adheres to the Twelve-Factor App principles and improves scalability.

    Resource Management:
    Kubernetes manifests include resource requests and limits to ensure that the application runs efficiently without overloading cluster resources.

    Security & Auditability:
    Running containers with non-root users and defining pod-level security contexts enhances the security and auditability of the deployment. Because the code is open source, you or your organization can audit it if needed—this is often a requirement in production systems, especially when deployed on-premises.

    No Additional External Dependencies:
    Since all packages (e.g., bufio, encoding/json, os/exec) are part of the standard library, you don't need to worry about external dependencies that might introduce vulnerabilities or require separate maintenance.




