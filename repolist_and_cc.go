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

// CodecovResponse represents the structure of the JSON response from Codecov API.
type CodecovResponse struct {
	Commit *struct {
		Totals *struct {
			C int     `json:"c"` // covered lines
			T int     `json:"t"` // total lines
			P float64 `json:"p"` // reported percentage (if provided)
		} `json:"totals"`
	} `json:"commit"`
}

// getCoverage calls the Codecov API for the given organization/repository and returns the coverage percentage.
func getCoverage(org, repo, token string) (float64, error) {
	// Construct the Codecov API URL
	url := fmt.Sprintf("https://codecov.io/api/gh/%s/%s?branch=main&token=%s", org, repo, token)
	fmt.Println("[DEBUG] Querying Codecov API:", url) // Print API URL for debugging

	// Create an HTTP client with a timeout.
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Get(url)
	if err != nil {
		return 0, fmt.Errorf("failed to call Codecov API: %v", err)
	}
	defer resp.Body.Close()

	// Debugging: Print HTTP status
	fmt.Printf("[DEBUG] Codecov API Response: %s (Status Code: %d)\n", resp.Status, resp.StatusCode)

	// Read full response body for debugging
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read Codecov API response: %v", err)
	}
	fmt.Println("[DEBUG] Codecov API Response Body:", string(body))

	if resp.StatusCode != 200 {
		return 0, fmt.Errorf("Codecov API returned non-200 status: %d", resp.StatusCode)
	}

	var covResp CodecovResponse
	if err := json.Unmarshal(body, &covResp); err != nil {
		return 0, fmt.Errorf("error decoding Codecov response: %v", err)
	}
	if covResp.Commit == nil || covResp.Commit.Totals == nil {
		return 0, fmt.Errorf("no coverage info found in response")
	}

	// If the API returns a percentage in the "p" field, use it.
	if covResp.Commit.Totals.P > 0 {
		return covResp.Commit.Totals.P, nil
	}
	// Otherwise, compute the percentage if total lines > 0.
	if covResp.Commit.Totals.T > 0 {
		return (float64(covResp.Commit.Totals.C) / float64(covResp.Commit.Totals.T)) * 100.0, nil
	}
	return 0, fmt.Errorf("invalid coverage numbers received")
}

func main() {
	org := "openshift" // Set your organization name here

	// Get GitHub token from environment variable.
	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" {
		log.Fatal("Please set the GITHUB_TOKEN environment variable")
	}

	// Get Codecov token from environment variable.
	codecovToken := os.Getenv("CODECOV_TOKEN")
	if codecovToken == "" {
		log.Fatal("Please set the CODECOV_TOKEN environment variable")
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	ghClient := github.NewClient(tc)

	// List repositories for the organization.
	repos, _, err := ghClient.Repositories.ListByOrg(ctx, org, nil)
	if err != nil {
		log.Fatalf("Error listing repositories: %v", err)
	}

	// Debugging: Print all retrieved repository names
	fmt.Println("[DEBUG] Retrieved repositories from GitHub:")
	for _, repo := range repos {
		fmt.Printf("- %s\n", repo.GetName())
	}

	// Iterate over each repository.
	for _, repo := range repos {
		repoName := repo.GetName()
		fmt.Printf("\n[INFO] Processing repository: %s\n", repoName)
		coverage, err := getCoverage(org, repoName, codecovToken)
		if err != nil {
			fmt.Printf("[ERROR] Repo: %s, error getting coverage: %v\n", repoName, err)
		} else {
			fmt.Printf("[SUCCESS] Repo: %s, Test Coverage: %.2f%%\n", repoName, coverage)
		}
	}
}
