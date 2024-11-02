package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

const (
	libsPath      = "libraries"
	servicesPath  = "services"
	packagePrefix = "github.com/emortalmc/mono-services/"
)

var currentSha = generateCurrentSha()
var githubAPIURL = generateGitHubAPIURL()

func generateCurrentSha() string {
	sha := os.Getenv("GITHUB_SHA")
	if sha != "" {
		return sha
	}

	cmd := exec.Command("git", "rev-parse", "HEAD")
	var out bytes.Buffer
	cmd.Stdout = &out
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Fatalf("failed to get current sha: %s: %v", stderr.String(), err)
	}

	return strings.TrimSpace(out.String())
}

func generateGitHubAPIURL() string {
	const baseURL = "https://api.github.com/repos/${{ github.repository }}/actions/workflows/${{ github.workflow }}/runs?branch=${{ github.ref_name }}&status=success&per_page=1"

	gitHubRef := os.Getenv("GITHUB_WORKFLOW_REF") // octocat/hello-world/.github/workflows/my-workflow.yml@refs/heads/my_branch
	parts := strings.Split(gitHubRef, "/")
	//log.Printf("parts: %v\n", parts)
	//log.Printf("original ref: %s\n", gitHubRef)
	for _, part := range parts {
		//log.Printf("part: %s\n", part)
		if strings.Contains(part, ".yaml") || strings.Contains(part, ".yml") {
			//log.Printf("found ref: %s\n", part)
			gitHubRef = part
			break
		}
	}

	if strings.Contains(gitHubRef, "@") {
		gitHubRef = strings.Split(gitHubRef, "@")[0] // my-workflow.y(a)ml
	}

	url := baseURL
	url = strings.ReplaceAll(url, "${{ github.repository }}", os.Getenv("GITHUB_REPOSITORY"))
	url = strings.ReplaceAll(url, "${{ github.workflow }}", gitHubRef)
	url = strings.ReplaceAll(url, "${{ github.ref_name }}", os.Getenv("GITHUB_REF_NAME"))
	return url
}

var NoRunsError = fmt.Errorf("no successful runs found")

func getLastSuccessfulBuildSha() (string, error) {
	gitHubToken := os.Getenv("GITHUB_TOKEN")

	req, err := http.NewRequest("GET", githubAPIURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	if gitHubToken != "" {
		req.Header.Set("Authorization", "Bearer "+gitHubToken)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get last successful build sha (URL: %s): %s", githubAPIURL, resp.Status)
	}

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	runs := data["workflow_runs"].([]interface{})
	if len(runs) == 0 {
		return "", NoRunsError
	}

	sha := runs[0].(map[string]interface{})["head_sha"].(string)
	return sha, nil
}

// getModules returns the list of modules referenced in the go work file.
// The return should be a list of modules in the format "services/mc-player-service", "libraries/libA", etc.
func getModules() ([]string, error) {
	cmd := exec.Command("go", "list", "-m", "-f", "{{.Dir}}")
	var out bytes.Buffer
	cmd.Stdout = &out
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to get modules: %s: %w", stderr.String(), err)
	}

	rawModules := strings.Split(out.String(), "\n")
	modules := make([]string, 0, len(rawModules))

	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	for _, module := range rawModules {
		if module == "" || module == wd {
			continue
		}

		modules = append(modules, strings.TrimPrefix(module, wd+string(os.PathSeparator)))
	}

	return modules, nil
}

func getChangedFiles(lastSuccessfulBuildSha string) ([]string, error) {
	cmd := exec.Command("git", "diff", "--name-only", lastSuccessfulBuildSha)
	var out bytes.Buffer
	cmd.Stdout = &out
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to get changed files: %s: %w", stderr.String(), err)
	}

	return strings.Split(out.String(), "\n"), nil
}

func getChangedModules(modules []string, files []string) ([]string, error) {
	changedModules := make([]string, 0, len(modules))
out:
	for _, module := range modules {
		for _, file := range files {
			if strings.HasPrefix(file, module) {
				changedModules = append(changedModules, module)
				continue out
			}
		}
	}

	return changedModules, nil
}

// DependencyGraph is a map of modules to their dependency modules.
type DependencyGraph map[string][]string

func getModuleDependencies(module string) ([]string, error) {
	cmd := exec.Command("go", "list", "-m", "-f", "{{.Path}}", "all")
	var out bytes.Buffer
	cmd.Stdout = &out
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	cmd.Dir = module
	cmd.Env = append(cmd.Env, "GOWORK=off")

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to get dependencies for module %s: %s: %w", module, stderr.String(), err)
	}

	lines := strings.Split(out.String(), "\n")
	var dependencies []string

	for _, line := range lines {
		dep := strings.TrimSpace(line)
		if !strings.HasPrefix(dep, packagePrefix) || dep == packagePrefix+module {
			continue
		}

		dependencies = append(dependencies, strings.TrimPrefix(dep, packagePrefix))
	}

	return dependencies, nil
}

func buildDependencyGraph(modules []string) (DependencyGraph, error) {
	graph := make(DependencyGraph)

	for _, module := range modules {
		deps, err := getModuleDependencies(module)
		if err != nil {
			return nil, err
		}
		graph[module] = deps
	}

	return graph, nil
}

// validateDependencyGraph checks that no services depend on other services. Crashes out if it finds a service dependency.
func validateDependencyGraph(graph DependencyGraph) {
	for module, deps := range graph {
		for _, dep := range deps {
			if strings.HasPrefix(dep, servicesPath) {
				log.Fatalf("service %s was depended on by %s. Services cannot be dependencies", dep, module)
			}
		}
	}
}

// Our process is as follows:
// 1. Get the list of changed files
// 2. Get the list of modules (libraries and services) referenced in the go work file
// 3. Calculate the list of changed modules
// 4. Get the dependencies of the changed modules
// 5. Build the changed services or services that depend on the changed libraries
func main() {
	modules, err := getModules()
	if err != nil {
		panic(err)
	}

	lastSuccessfulBuildSha, err := getLastSuccessfulBuildSha()
	if err != nil {
		if errors.Is(err, NoRunsError) { // if no runs, build all services
			servicesToBuild := make([]string, 0)
			for _, module := range modules {
				if strings.HasPrefix(module, servicesPath) {
					servicesToBuild = append(servicesToBuild, strings.TrimPrefix(module, "services/"))
				}
			}

			jsonOutput, err := json.Marshal(servicesToBuild)
			if err != nil {
				log.Fatalf("failed to marshal services to build: %v", err)
			}

			fmt.Println(string(jsonOutput))
			return
		} else {
			panic(err)
		}
	}

	changedFiles, err := getChangedFiles(lastSuccessfulBuildSha)
	if err != nil {
		panic(err)
	}
	//fmt.Printf("modules: %v\n", modules)

	changedModules, err := getChangedModules(modules, changedFiles)
	if err != nil {
		panic(err)
	}
	//fmt.Printf("changed modules: %v\n", changedModules)

	graph, err := buildDependencyGraph(modules)
	if err != nil {
		panic(err)
	}

	// Flag any problems with dependencies
	validateDependencyGraph(graph)

	// create servicesToBuild with changed services. Libraries are handled differently as they're just dependencies.
	servicesToBuild := make([]string, 0)
	for _, module := range changedModules {
		if strings.HasPrefix(module, servicesPath) {
			servicesToBuild = append(servicesToBuild, strings.TrimPrefix(module, "services/"))
		}
	}

	// Add any additional services that depend on the changed libraries
out:
	for module, deps := range graph {
		if contains(servicesToBuild, strings.TrimPrefix(module, "services/")) { // Skip modules that are already in servicesToBuild
			continue
		}

		for _, dep := range deps {
			if contains(changedModules, dep) {
				servicesToBuild = append(servicesToBuild, strings.TrimPrefix(module, "services/"))
				break out
			}
		}
	}

	//fmt.Printf("modules to build: %v\n", servicesToBuild)
	jsonOutput, err := json.Marshal(servicesToBuild)
	if err != nil {
		log.Fatalf("failed to marshal services to build: %v", err)
	}

	fmt.Println(string(jsonOutput))
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
