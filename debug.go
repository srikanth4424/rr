package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/go-github/v53/github"
	"golang.org/x/oauth2"
)

// Codecov API base URL
const codecovAPIBase = "https://codecov.io/api/v2/github"

// Repository structure for parsing API response
type Repo struct {
	Name string `json:"name"`
}

// Commit structure to fetch latest test coverage
type Commit struct {
	Totals struct {
		Coverage float64 `json:"coverage"`
	} `json:"totals"`
}

// Fetch list of repositories from Codecov
func getAllRepos(org, token string) ([]string, error) {
	url := fmt.Sprintf("%s/%s/repos", codecovAPIBase, org)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Codecov API: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Codecov API returned non-200 status: %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var data struct {
		Results []Repo `json:"results"`
	}

	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("error decoding repo list: %v", err)
	}

	var repos []string
	for _, repo := range data.Results {
		repos = append(repos, repo.Name)
	}

	return repos, nil
}

// Fetch latest commit test coverage for a repository
func getRepoCoverage(org, repo, token string) (float64, error) {
	url := fmt.Sprintf("%s/%s/repos/%s/commits", codecovAPIBase, org, repo)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to call Codecov API: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return 0, fmt.Errorf("Codecov API returned non-200 status: %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var data struct {
		Results []Commit `json:"results"`
	}

	if err := json.Unmarshal(body, &data); err != nil {
		return 0, fmt.Errorf("error decoding coverage data: %v", err)
	}

	if len(data.Results) == 0 || data.Results[0].Totals.Coverage == 0 {
		return 0, fmt.Errorf("no coverage data found for repo %s", repo)
	}

	return data.Results[0].Totals.Coverage, nil
}

func main() {
	org := "openshift" // Organization name

	// Get Codecov token from environment variable
	codecovToken := os.Getenv("CODECOV_TOKEN")
	if codecovToken == "" {
		log.Fatal("Please set the CODECOV_TOKEN environment variable")
	}

	// Fetch repositories from Codecov
	repos, err := getAllRepos(org, codecovToken)
	if err != nil {
		log.Fatalf("Error getting repos: %v", err)
	}

	// Fetch coverage for each repository
	for _, repo := range repos {
		coverage, err := getRepoCoverage(org, repo, codecovToken)
		if err == nil { // Only print repos with coverage
			fmt.Printf("Repo: %s, Test Coverage: %.2f%%\n", repo, coverage)
		}
	}
}
