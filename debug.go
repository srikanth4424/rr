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

// Fetch all repositories using pagination
func getAllRepos(org, githubToken string) ([]string, error) {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: githubToken})
	tc := oauth2.NewClient(ctx, ts)
	ghClient := github.NewClient(tc)

	var allRepos []string
	opts := &github.RepositoryListByOrgOptions{ListOptions: github.ListOptions{PerPage: 100}} // Fetch 100 repos per request

	for {
		repos, resp, err := ghClient.Repositories.ListByOrg(ctx, org, opts)
		if err != nil {
			return nil, fmt.Errorf("error fetching repositories from GitHub: %v", err)
		}

		// Append repo names
		for _, repo := range repos {
			allRepos = append(allRepos, repo.GetName())
		}

		// Break loop if there are no more pages
		if resp.NextPage == 0 {
			break
		}

		// Move to next page
		opts.Page = resp.NextPage
	}

	// Print all repos retrieved
	fmt.Println("\n[DEBUG] Retrieved repositories from GitHub (Total:", len(allRepos), ")")
	for _, repo := range allRepos {
		fmt.Println("- ", repo)
	}

	return allRepos, nil
}

// Fetch latest commit test coverage for a repository
func getRepoCoverage(org, repo, token string) (float64, error) {
	// Construct the Codecov API URL
	url := fmt.Sprintf("%s/%s/repos/%s/commits", codecovAPIBase, org, repo)
	fmt.Println("\n[DEBUG] Codecov API URL:", url) // Print constructed API URL

	// Make HTTP request to Codecov API
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token) // Use Bearer Token authentication

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to call Codecov API: %v", err)
	}
	defer resp.Body.Close()

	// Debugging: Print HTTP response status
	fmt.Printf("[DEBUG] Codecov API Response Status: %d\n", resp.StatusCode)

	// Read full response body for debugging
	body, _ := io.ReadAll(resp.Body)
	fmt.Println("[DEBUG] Codecov API Response Body:", string(body)) // Print API response body

	if resp.StatusCode != 200 {
		return 0, fmt.Errorf("Codecov API returned non-200 status: %d", resp.StatusCode)
	}

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

	// Get API tokens from environment variables
	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" {
		log.Fatal("❌ Please set the GITHUB_TOKEN environment variable")
	}

	codecovToken := os.Getenv("CODECOV_TOKEN")
	if codecovToken == "" {
		log.Fatal("❌ Please set the CODECOV_TOKEN environment variable")
	}

	// Fetch repositories from GitHub
	repos, err := getAllRepos(org, githubToken)
	if err != nil {
		log.Fatalf("❌ Error getting repos: %v", err)
	}

	// Fetch coverage for each repository
	fmt.Println("\n[INFO] Fetching test coverage for repositories...")
	for _, repo := range repos {
		fmt.Printf("\n🔹 Checking coverage for repo: %s\n", repo)
		coverage, err := getRepoCoverage(org, repo, codecovToken)
		if err == nil { // Only print repos with valid coverage
			fmt.Printf("✅ Repo: %s, Test Coverage: %.2f%%\n", repo, coverage)
		}
	}

	// Debug: Total repositories found
	fmt.Printf("\n✅ Total repositories processed: %d\n", len(repos))
}
