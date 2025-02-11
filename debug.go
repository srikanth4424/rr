package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/go-github/v53/github"
	"golang.org/x/oauth2"
)

// getCoverage constructs the Codecov API URL and prints it
func getCoverage(org, repo, token string) {
	// Construct the Codecov API URL
	url := fmt.Sprintf("https://codecov.io/api/gh/%s/%s?branch=main&token=%s", org, repo, token)
	fmt.Println(url) // Print only the constructed API URL

	// Make the request to validate if it works
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		fmt.Printf("Error calling Codecov API for %s: %v\n", repo, err)
		return
	}
	defer resp.Body.Close()

	// Print response status for debugging
	fmt.Printf("Repo: %s, Status Code: %d\n", repo, resp.StatusCode)
}

func main() {
	org := "openshift" // Set organization name

	// Get GitHub and Codecov tokens from environment variables
	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" {
		log.Fatal("Please set the GITHUB_TOKEN environment variable")
	}

	codecovToken := os.Getenv("CODECOV_TOKEN")
	if codecovToken == "" {
		log.Fatal("Please set the CODECOV_TOKEN environment variable")
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: githubToken})
	tc := oauth2.NewClient(ctx, ts)
	ghClient := github.NewClient(tc)

	// List repositories for the organization
	repos, _, err := ghClient.Repositories.ListByOrg(ctx, org, nil)
	if err != nil {
		log.Fatalf("Error listing repositories: %v", err)
	}

	// Iterate over each repository and print the Codecov API URL
	for _, repo := range repos {
		repoName := repo.GetName()
		getCoverage(org, repoName, codecovToken)
	}
}
