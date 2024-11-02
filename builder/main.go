package main

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"strings"
)

const (
	goWorkFile    = "go.work"
	libsPath      = "libraries"
	servicesPath  = "services"
	packagePrefix = "github.com/emortalmc/mono-services/"
)

// getModules returns the list of modules referenced in the go work file.
// The return should be a list of modules in the format "services/mc-player-service", "libraries/libA", etc.
func getModules() ([]string, error) {
	cmd := exec.Command("go", "list", "-m", "-f", "{{.Dir}}")
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to get modules: %w", err)
	}

	rawModules := strings.Split(out.String(), "\n")
	modules := make([]string, 0, len(rawModules))
	for _, module := range rawModules {
		modules = append(modules, strings.Split(module, "github.com/emortalmc/mono-services/")[1])
	}

	return modules, nil
}

func getChangedFiles() ([]string, error) {
	cmd := exec.Command("git", "diff", "--name-only", "HEAD^", "HEAD")
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to get changed files: %w", err)
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
	cmd := exec.Command("go", "list", "-m", "all", packagePrefix+module)
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to get dependencies for module %s: %w", module, err)
	}

	lines := strings.Split(out.String(), "\n")
	var dependencies []string

	for _, line := range lines {
		dep := strings.TrimSpace(line)
		if dep == "" || !strings.HasPrefix(dep, packagePrefix) {
			continue
		}
		dependencies = append(dependencies, dep)
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

// Our process is as follows:
// 1. Get the list of changed files
// 2. Get the list of modules (libraries and services) referenced in the go work file
// 3. Calculate the list of changed modules
// 4. Get the dependencies of the changed modules
// 5. Build the changed services or services that depend on the changed libraries
func main() {
	changedFiles, err := getChangedFiles()
	if err != nil {
		panic(err)
	}

	modules, err := getModules()
	if err != nil {
		panic(err)
	}
	fmt.Printf("modules: %v\n", modules)

	changedModules, err := getChangedModules(modules, changedFiles)
	if err != nil {
		panic(err)
	}
	fmt.Printf("changed modules: %v\n", changedModules)

	graph, err := buildDependencyGraph(modules)

	// Flag any problems with dependencies
	for module, deps := range graph {
		for _, dep := range deps {
			if strings.HasPrefix(dep, servicesPath) {
				log.Fatalf("service %s was depended on by %s. Services cannot be dependencies.", dep, module)
			}
		}
	}

	// create modulesToBuild with changed services. Libraries are handled differently as they're just dependencies.
	modulesToBuild := make([]string, 0)
	for _, module := range changedModules {
		if strings.HasPrefix(module, servicesPath) {
			modulesToBuild = append(modulesToBuild, module)
		}
	}

	// Add any additional services that depend on the changed libraries
out:
	for module, deps := range graph {
		if contains(modulesToBuild, module) { // Skip modules that are already in modulesToBuild
			continue
		}

		for _, dep := range deps {
			if contains(changedModules, dep) {
				modulesToBuild = append(modulesToBuild, module)
				break out
			}
		}
	}

	fmt.Printf("modules to build: %v\n", modulesToBuild)
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
