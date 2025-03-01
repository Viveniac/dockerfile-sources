package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// outputData defines the final JSON structure.
// Data maps "repoURL:commitSHA" to a mapping of Dockerfile paths and their FROM images.
// Errors captures any processing errors for each repository.
type outputData struct {
	Data   map[string]map[string][]string `json:"data"`
	Errors map[string]string              `json:"errors,omitempty"`
}

// validLineRegex validates lines with the format:
//
//	https://github.com/<org>/<repo>.git <commitSHA>
var validLineRegex = regexp.MustCompile(`^(https://github\.com/[\w\-\./]+\.git)\s+([0-9a-fA-F]{6,40})$`)

func main() {
	// Use --url flag, or fall back to REPOSITORY_LIST_URL environment variable.
	urlFlag := flag.String("url", "", "URL of the plaintext file containing repository list.")
	flag.Parse()
	repoListURL := *urlFlag
	if repoListURL == "" {
		repoListURL = os.Getenv("REPOSITORY_LIST_URL")
	}
	if repoListURL == "" {
		log.Fatalln("Error: no repository list URL provided. Use --url or set REPOSITORY_LIST_URL.")
	}

	// Download the repository list.
	lines, err := downloadRepoList(repoListURL)
	if err != nil {
		log.Fatalf("Error downloading repository list: %v\n", err)
	}

	// Initialize output structure.
	result := outputData{
		Data:   make(map[string]map[string][]string),
		Errors: make(map[string]string),
	}

	// Process each line. Skip invalid lines and record errors per repository.
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		repoURL, commitSHA, ok := parseLine(line)
		if !ok {
			log.Printf("Skipping invalid line: %q\n", line)
			continue
		}
		key := fmt.Sprintf("%s:%s", repoURL, commitSHA)
		dockerData, procErr := processRepository(repoURL, commitSHA)
		if procErr != nil {
			result.Errors[key] = procErr.Error()
		} else {
			result.Data[key] = dockerData
		}
	}

	// Remove Errors field if there were no errors.
	if len(result.Errors) == 0 {
		result.Errors = nil
	}

	// Marshal and print the final JSON.
	jsonBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		log.Fatalf("Error marshaling JSON: %v\n", err)
	}
	fmt.Println(string(jsonBytes))
}

// downloadRepoList retrieves the file from the provided URL and returns its lines.
func downloadRepoList(url string) ([]string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("HTTP GET error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	var lines []string
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanner error: %w", err)
	}
	return lines, nil
}

// parseLine extracts the repository URL and commit SHA from a line.
// Returns false if the line does not match the expected pattern.
func parseLine(line string) (string, string, bool) {
	matches := validLineRegex.FindStringSubmatch(line)
	if matches == nil {
		return "", "", false
	}
	return matches[1], matches[2], true
}

// processRepository clones the repository, checks out the specified commit,
// finds Dockerfiles, and extracts FROM statements from each Dockerfile.
func processRepository(repoURL, commitSHA string) (map[string][]string, error) {
	// Create a temporary directory for cloning.
	tmpDir, err := os.MkdirTemp("", "repo-")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	// Ensure cleanup of the temporary directory.
	defer os.RemoveAll(tmpDir)

	// Clone the repository.
	if err := gitClone(repoURL, tmpDir); err != nil {
		return nil, fmt.Errorf("git clone error: %w", err)
	}

	// Checkout the specific commit.
	if err := gitCheckout(tmpDir, commitSHA); err != nil {
		return nil, fmt.Errorf("git checkout error: %w", err)
	}

	// Find Dockerfiles in the repository.
	dockerfiles, err := findDockerfiles(tmpDir)
	if err != nil {
		return nil, fmt.Errorf("error finding Dockerfiles: %w", err)
	}

	// Parse FROM statements from each Dockerfile.
	data := make(map[string][]string)
	for _, df := range dockerfiles {
		fullPath := filepath.Join(tmpDir, df)
		images, err := parseFromStatements(fullPath)
		if err != nil {
			log.Printf("Error parsing %s: %v", fullPath, err)
			continue
		}
		data[df] = images
	}
	return data, nil
}

// gitClone runs the git clone command to clone the repository into the given directory.
func gitClone(repoURL, dir string) error {
	cmd := exec.Command("git", "clone", "--quiet", repoURL, dir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to clone %s: %s (output: %s)", repoURL, err, string(out))
	}
	return nil
}

// gitCheckout checks out the given commit within the repository directory.
func gitCheckout(dir, commitSHA string) error {
	cmd := exec.Command("git", "checkout", "--quiet", commitSHA)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to checkout commit %s: %s (output: %s)", commitSHA, err, string(out))
	}
	return nil
}

// findDockerfiles recursively finds files named exactly "Dockerfile" starting at root.
func findDockerfiles(root string) ([]string, error) {
	var dockerfiles []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return filepath.SkipDir
		}
		if !info.IsDir() && info.Name() == "Dockerfile" {
			rel, err := filepath.Rel(root, path)
			if err == nil {
				dockerfiles = append(dockerfiles, rel)
			}
		}
		return nil
	})
	return dockerfiles, err
}

// parseFromStatements opens a Dockerfile and extracts image names from lines starting with "FROM".
// It strips off any trailing alias (e.g., "AS builder").
func parseFromStatements(dockerfilePath string) ([]string, error) {
	file, err := os.Open(dockerfilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var images []string
	scanner := bufio.NewScanner(file)
	re := regexp.MustCompile(`(?i)^\s*FROM\s+([^\s]+)`)
	for scanner.Scan() {
		line := scanner.Text()
		if m := re.FindStringSubmatch(line); m != nil {
			image := strings.Split(m[1], "AS")[0]
			image = strings.TrimSpace(image)
			images = append(images, image)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanner error: %w", err)
	}
	return images, nil
}
