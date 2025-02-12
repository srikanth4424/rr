package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"flag"
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

// Structs for detailed file coverage report
type FileCoverage struct {
	Name   string `json:"name"`
	Totals struct {
		Lines    int     `json:"lines"`
		Hits     int     `json:"hits"`
		Misses   int     `json:"misses"`
		Coverage float64 `json:"coverage"`
	} `json:"totals"`
}

type CodecovReport struct {
	Totals struct {
		Coverage float64 `json:"coverage"`
	} `json:"totals"`
	Files []FileCoverage `json:"files"`
}

// Fetch latest commit test coverage for a repository
func getRepoCoverage(org, repo, token string) (float64, bool) {
	url := fmt.Sprintf("%s/%s/repos/%s/commits", codecovAPIBase, org, repo)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		return 0, false
	}
	defer resp.Body.Close()

	var data struct {
		Results []struct {
			Totals struct {
				Coverage float64 `json:"coverage"`
			} `json:"totals"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, false
	}

	if len(data.Results) == 0 || data.Results[0].Totals.Coverage == 0 {
		return 0, false
	}

	return data.Results[0].Totals.Coverage, true
}

// Fetch detailed code coverage report
func getDetailedCoverageReport(org, repo, token string) (*CodecovReport, error) {
	url := fmt.Sprintf("https://api.codecov.io/api/v2/gh/%s/repos/%s/report", org, repo)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to fetch detailed report for %s", repo)
	}
	defer resp.Body.Close()

	var report CodecovReport
	if err := json.NewDecoder(resp.Body).Decode(&report); err != nil {
		return nil, err
	}

	return &report, nil
}

// Generate a CSV file for detailed coverage
func generateCSVReport(repo string, report *CodecovReport) error {
	filename := fmt.Sprintf("detailed_%s_coverage_report.csv", repo)
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	writer.Write([]string{"File", "Total Lines", "Covered Lines", "Missed Lines", "Coverage %"})

	// Sort files by lowest coverage
	sort.Slice(report.Files, func(i, j int) bool {
		return report.Files[i].Totals.Coverage < report.Files[j].Totals.Coverage
	})

	// Write data
	for _, file := range report.Files {
		writer.Write([]string{
			file.Name,
			fmt.Sprintf("%d", file.Totals.Lines),
			fmt.Sprintf("%d", file.Totals.Hits),
			fmt.Sprintf("%d", file.Totals.Misses),
			fmt.Sprintf("%.2f", file.Totals.Coverage),
		})
	}

	return nil
}

func main() {
	// Parse flags
	verbose := flag.Bool("v", false, "Enable verbose mode to generate detailed coverage reports")
	flag.Parse()

	org := "openshift" // Organization name

	// Get API tokens from environment variables
	codecovToken := os.Getenv("CODECOV_TOKEN")
	if codecovToken == "" {
		log.Fatal("❌ Please set the CODECOV_TOKEN environment variable")
	}

	// List of example repositories to test
	repos := []string{"backplane-cli", "example-repo", "another-repo"} // Replace with actual fetched repo list

	// Print header for CSV output
	fmt.Println("Repository, Coverage Percentage")

	// Fetch coverage for each repository
	for _, repo := range repos {
		coverage, configured := getRepoCoverage(org, repo, codecovToken)
		if configured {
			fmt.Printf("%s, %.2f%%\n", repo, coverage)

			// Generate detailed report if verbose mode is enabled
			if *verbose {
				report, err := getDetailedCoverageReport(org, repo, codecovToken)
				if err == nil {
					_ = generateCSVReport(repo, report)
				}
			}
		} else {
			fmt.Printf("%s, Not Configured\n", repo)
		}
	}
}
