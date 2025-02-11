package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"time"

	"github.com/google/go-github/v53/github"
	"golang.org/x/oauth2"
)

// Codecov API base URL
const codecovAPIBase = "https://codecov.io/api/v2/github"

// Commit structure to fetch latest test coverage
type Commit struct {
	Totals struct {
		Coverage float64 `json:"coverage"`
	} `json:"totals"`
}

// RepoCoverage stores repo name and its coverage percentage
type RepoCoverage struct {
	Name      string
	Coverage  float64
	Configured bool
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

	return allRepos, nil
}

// Fetch latest commit test coverage for a repository
func getRepoCoverage(org, repo, token string) (float64, bool) {
	// Construct the Codecov API URL
	url := fmt.Sprintf("%s/%s/repos/%s/commits", codecovAPIBase, org, repo)

	// Make HTTP request to Codecov API
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token) // Use Bearer Token authentication

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, false
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return 0, false
	}

	var data struct {
		Results []Commit `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, false
	}

	if len(data.Results) == 0 || data.Results[0].Totals.Coverage == 0 {
		return 0, false
	}

	return data.Results[0].Totals.Coverage, true
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

	// Store coverage details
	var coveredRepos []RepoCoverage
	var notConfiguredRepos []RepoCoverage

	// Fetch coverage for each repository
	for _, repo := range repos {
		coverage, configured := getRepoCoverage(org, repo, codecovToken)
		if configured {
			coveredRepos = append(coveredRepos, RepoCoverage{Name: repo, Coverage: coverage, Configured: true})
		} else {
			notConfiguredRepos = append(notConfiguredRepos, RepoCoverage{Name: repo, Coverage: 0, Configured: false})
		}
	}

	// Sort repositories with coverage in descending order
	sort.Slice(coveredRepos, func(i, j int) bool {
		return coveredRepos[i].Coverage > coveredRepos[j].Coverage
	})

	// Print repositories with coverage first
	for _, repo := range coveredRepos {
		fmt.Printf("%s: %.2f%%\n", repo.Name, repo.Coverage)
	}

	// Print repositories without coverage
	for _, repo := range notConfiguredRepos {
		fmt.Printf("%s: Not Configured\n", repo.Name)
	}
}
