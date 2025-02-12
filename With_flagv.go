package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
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
	req.Header.Set("Authorization", "Bearer "+token) // Use Bearer Token authentication

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("[DEBUG] ❌ Error calling Codecov API for %s: %v\n", repo, err)
		return 0, false
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Printf("[DEBUG] ❌ Codecov API returned status %d for repo: %s\n", resp.StatusCode, repo)
		return 0, false
	}

	var data struct {
		Results []struct {
			Totals struct {
				Coverage float64 `json:"coverage"`
			} `json:"totals"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		fmt.Printf("[DEBUG] ❌ Error decoding JSON for repo %s: %v\n", repo, err)
		return 0, false
	}

	if len(data.Results) == 0 || data.Results[0].Totals.Coverage == 0 {
		fmt.Printf("[DEBUG] ❌ No coverage data found for repo %s\n", repo)
		return 0, false
	}

	return data.Results[0].Totals.Coverage, true
}

// Fetch detailed code coverage report
func getDetailedCoverageReport(org, repo, token string) (*CodecovReport, error) {
	url := fmt.Sprintf("https://api.codecov.io/api/v2/gh/%s/repos/%s/report", org, repo)
	fmt.Printf("[DEBUG] 📢 Fetching detailed coverage report: %s\n", url) // Print API URL

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("[DEBUG] ❌ Error fetching detailed report for %s: %v\n", repo, err)
		return nil, err
	}
	defer resp.Body.Close()

	fmt.Printf("[DEBUG] 📢 Received HTTP status: %d for %s\n", resp.StatusCode, repo)

	if resp.StatusCode != 200 {
		fmt.Printf("[DEBUG] ❌ Codecov API returned non-200 status for detailed report of %s: %d\n", repo, resp.StatusCode)
		return nil, fmt.Errorf("Codecov API returned non-200 status for detailed report: %d", resp.StatusCode)
	}

	var report CodecovReport
	body, _ := io.ReadAll(resp.Body)
	if len(body) == 0 {
		fmt.Printf("[DEBUG] ❌ Empty response body for %s\n", repo)
		return nil, fmt.Errorf("empty response body")
	}

	if err := json.Unmarshal(body, &report); err != nil {
		fmt.Printf("[DEBUG] ❌ Error decoding JSON for detailed report of %s: %v\n", repo, err)
		return nil, err
	}

	return &report, nil
}

// Generate a Markdown report for the detailed coverage
func generateMarkdownReport(repo string, report *CodecovReport) error {
	// Sort files by lowest coverage
	sort.Slice(report.Files, func(i, j int) bool {
		return report.Files[i].Totals.Coverage < report.Files[j].Totals.Coverage
	})

	// Build report
	output := fmt.Sprintf("# %s Detailed Code Coverage Report\n\n", repo)
	output += fmt.Sprintf("## 📊 Overall Coverage\n- **Total Coverage**: `%.2f%%`\n\n", report.Totals.Coverage)
	output += "## 📉 Files with Low Test Coverage (Sorted in Ascending Order)\n"
	output += "| File | Total Lines | Covered Lines | Missed Lines | Coverage % |\n"
	output += "|------|------------|--------------|-------------|------------|\n"

	for _, file := range report.Files {
		output += fmt.Sprintf("| `%s` | %d | %d | %d | **%.2f%%** |\n",
			file.Name, file.Totals.Lines, file.Totals.Hits, file.Totals.Misses, file.Totals.Coverage)
	}

	filename := fmt.Sprintf("detailed_%s_coverage_report.md", repo)
	err := os.WriteFile(filename, []byte(output), 0644)
	if err != nil {
		fmt.Printf("[DEBUG] ❌ Error writing report for %s: %v\n", repo, err)
		return err
	}

	fmt.Printf("[DEBUG] ✅ Detailed coverage report generated: %s\n", filename)
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

	// Fetch coverage for each repository
	for _, repo := range repos {
		fmt.Printf("\n🔹 Checking coverage for repo: %s\n", repo)
		coverage, configured := getRepoCoverage(org, repo, codecovToken)
		if configured {
			fmt.Printf("%s: %.2f%%\n", repo, coverage)

			// Generate detailed report if verbose mode is enabled
			if *verbose {
				report, err := getDetailedCoverageReport(org, repo, codecovToken)
				if err == nil {
					err = generateMarkdownReport(repo, report)
					if err != nil {
						fmt.Printf("[DEBUG] ❌ Failed to generate markdown report for %s\n", repo)
					}
				} else {
					fmt.Printf("[DEBUG] ❌ Failed to fetch detailed report for %s\n", repo)
				}
			}
		} else {
			fmt.Printf("%s: Not Configured\n", repo)
		}
	}
}
