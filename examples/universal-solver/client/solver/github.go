package solver

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
)

type GithubRepo struct {
	ID          int64       `json:"id"`
	Name        string      `json:"name"`
	FullName    string      `json:"full_name"`
	Description string      `json:"description"`
	HTMLURL     string      `json:"html_url"`
	Private     bool        `json:"private"`
	Owner       Owner       `json:"owner"`
	Args        interface{} `json:"args"`
	PortNo      int64       `json:"port_no"`
}

type Owner struct {
	Login string `json:"login"`
	ID    int64  `json:"id"`
	URL   string `json:"url"`
}

type GithubAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	FixedToken   string
}

type GithubContent struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	Path        string `json:"path"`
	SHA         string `json:"sha"`
	Size        int    `json:"size"`
	URL         string `json:"url"`
	DownloadURL string `json:"download_url"`
}

// RepoFile represents a file in the repository with its content
type RepoFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Type    string `json:"type"`
	Size    int    `json:"size"`
	SHA     string `json:"sha"`
}

// Update the ExampleRepo struct to include a Type field
type ExampleRepo struct {
	Owner  string
	Repo   string
	Type   string // New field to specify the language type
	Args   string
	PortNo int64
}

// Update the ExampleRepos variable to include the type for each repository
var ExampleRepos = []ExampleRepo{
	{
		Owner:  "bhaagiKenpachi",
		Repo:   "universal-solver-linera",
		Type:   "wasm",
		Args:   "",
		PortNo: 0,
	},
	{
		Owner:  "bhaagiKenpachi",
		Repo:   "linera-non-fungible",
		Type:   "wasm",
		Args:   "--required-application-ids 3252b5949106daa5cf7557e0a41beefa959a8ee2b684ec92ba5efe7c89810aaf5dbd4f9c9e4b3ff73cfcba4fc77808ed9537290b1e70f9a40f11073e6c45d6fae476187f6ddfeb9d588c7b45d3df334d5501d6499b3f9ad5595cae86cce16a65330000000000000000000000 --json-parameters \"3252b5949106daa5cf7557e0a41beefa959a8ee2b684ec92ba5efe7c89810aaf5dbd4f9c9e4b3ff73cfcba4fc77808ed9537290b1e70f9a40f11073e6c45d6fae476187f6ddfeb9d588c7b45d3df334d5501d6499b3f9ad5595cae86cce16a65330000000000000000000000\"",
		PortNo: 0,
	},
	{
		Owner:  "bhaagiKenpachi",
		Repo:   "non-fungile-client",
		Type:   "go",
		Args:   "-seed-phrase \"indoor dish desk flag debris potato excuse depart ticket judge file exit\" -solver-url  https://linera-api.ngrok.io/chains/e476187f6ddfeb9d588c7b45d3df334d5501d6499b3f9ad5595cae86cce16a65/applications/249979fb024ea93cd229bd36b4940ee0451156bdf8f379e61947673566fb3c1aaaf5aa0e834e1fdf7321a32a6257e5b7728b71a67c4f28d98b5d5fddfb455687e476187f6ddfeb9d588c7b45d3df334d5501d6499b3f9ad5595cae86cce16a65050000000000000000000000 -solana-url \"https://sol-test.dojima.network\" -ethereum-url \"https://eth-test.dojima.network\" --non-fungible-url https://linera-api.ngrok.io/chains/e476187f6ddfeb9d588c7b45d3df334d5501d6499b3f9ad5595cae86cce16a65/applications/923fa4bfb37c9aae3fb6901a6230cbcfb675de2cd07c709ea18d209912ac69116c4302192e50d455e7d5c8e554c69177d509ad28228e8c29fb6abd478fbd6abee476187f6ddfeb9d588c7b45d3df334d5501d6499b3f9ad5595cae86cce16a65070000000000000000000000 --nft-address 0x335Fe1De453F85cFAa8B1d0c902E2d89F461f3E",
		PortNo: 3000,
	},
	{
		Owner:  "bhaagiKenpachi",
		Repo:   "universal-solver-client",
		Type:   "go", // Specify the language type
		Args:   "-seed-phrase \"indoor dish desk flag debris potato excuse depart ticket judge file exit\" -solver-url https://linera-api.ngrok.io/chains/e476187f6ddfeb9d588c7b45d3df334d5501d6499b3f9ad5595cae86cce16a65/applications/3252b5949106daa5cf7557e0a41beefa959a8ee2b684ec92ba5efe7c89810aaf5dbd4f9c9e4b3ff73cfcba4fc77808ed9537290b1e70f9a40f11073e6c45d6fae476187f6ddfeb9d588c7b45d3df334d5501d6499b3f9ad5595cae86cce16a65330000000000000000000000 -solana-url  \"https://sol-test.dojima.network\"  -ethereum-url  \"https://eth-test.dojima.network\"",
		PortNo: 3001,
	},
	{
		Owner:  "bhaagiKenpachi",
		Repo:   "linera-fungible",
		Type:   "wasm", // Specify the language type
		Args:   "--json-argument {\"accounts\":{\"User:c3562b79502f7e0ae98b471b275846f91ad052f6ab9bb5fba1ecc1a9dd5a79c9\":\"100\"}} --json-parameters {\"ticker_symbol\":\"FUN\"}",
		PortNo: 0,
	},
	{
		Owner:  "bhaagiKenpachi",
		Repo:   "linera-counter",
		Type:   "wasm",
		Args:   "--json-argument 1",
		PortNo: 0,
	},
	{
		Owner:  "bhaagiKenpachi",
		Repo:   "linera-crowd-funding",
		Type:   "wasm",
		Args:   "--required-application-ids c85ece3934d1f286edb4c5c838606e3847269366e499da6f60f180088cfb207dae378075709425bee2e559036c03785899660d4b83418e95aead3b7ad301735ce476187f6ddfeb9d588c7b45d3df334d5501d6499b3f9ad5595cae86cce16a65270000000000000000000000 --json-argument {\"owner\":\"User:c3562b79502f7e0ae98b471b275846f91ad052f6ab9bb5fba1ecc1a9dd5a79c9\",\"deadline\":4102473600000000,\"target\":\"100\"} --json-parameters \"c85ece3934d1f286edb4c5c838606e3847269366e499da6f60f180088cfb207dae378075709425bee2e559036c03785899660d4b83418e95aead3b7ad301735ce476187f6ddfeb9d588c7b45d3df334d5501d6499b3f9ad5595cae86cce16a65270000000000000000000000\"",
		PortNo: 0,
	},
	// Add more example repos as needed
}

func NewGithubClient(clientid, clientSecret, redirectUri, fixedToken string) *GithubAuthConfig {
	return &GithubAuthConfig{
		ClientID:     clientid,
		ClientSecret: clientSecret,
		RedirectURI:  redirectUri,
		FixedToken:   fixedToken,
	}
}

func (c *GithubAuthConfig) ExchangeCodeForToken(code string) (string, error) {
	// Prepare request body
	data := strings.NewReader(fmt.Sprintf(
		"client_id=%s&client_secret=%s&code=%s",
		c.ClientID,
		c.ClientSecret,
		code,
	))

	// Make request to GitHub
	req, err := http.NewRequest("POST", "https://github.com/login/oauth/access_token", data)
	if err != nil {
		return "", err
	}

	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Parse response
	var result struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		Scope       string `json:"scope"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.AccessToken, nil
}

func FetchGithubRepos(token string) ([]GithubRepo, error) {
	// Make request to GitHub API
	req, err := http.NewRequest("GET", "https://api.github.com/user/repos", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	// Check for 403 Forbidden status
	if resp.StatusCode == http.StatusForbidden {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error fetching repos: 403 Forbidden - %s", string(body))
	}

	// Parse response
	var repos []GithubRepo
	if resp.StatusCode == http.StatusOK {
		if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
			return nil, fmt.Errorf("error parsing response: %v", err)
		}
	} else {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error: %s", string(body))
	}

	return repos, nil
}

func GenerateRandomState() string {
	// Generate a random string for CSRF protection
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 32)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// isExampleRepo checks if a repository is in the example repos list
func (c *GithubAuthConfig) IsExampleRepo(owner, repo string) bool {
	for _, example := range ExampleRepos {
		if example.Owner == owner && example.Repo == repo {
			return true
		}
	}
	return false
}

// getAppropriateToken returns either the fixed token or auth token based on the repo
func (c *GithubAuthConfig) getAppropriateToken(owner, repo, authToken string) string {
	if c.IsExampleRepo(owner, repo) {
		return c.FixedToken
	}
	return authToken
}

// FetchRepoFilesRecursively fetches all files from a repository recursively
func (c *GithubAuthConfig) FetchRepoFilesRecursively(authToken, owner, repo string) ([]RepoFile, error) {
	var allFiles []RepoFile

	// Get appropriate token based on repo
	token := c.getAppropriateToken(owner, repo, authToken)
	if token == "" {
		return nil, fmt.Errorf("no valid token available for %s/%s", owner, repo)
	}

	Logger.Printf("Fetching files for repository %s/%s", owner, repo)
	if c.IsExampleRepo(owner, repo) {
		Logger.Printf("Using fixed token for example repository")
	}

	// Start with root directory
	err := c.fetchDirectoryContents(token, owner, repo, "", &allFiles)
	if err != nil {
		return nil, fmt.Errorf("error fetching repository contents: %v", err)
	}

	Logger.Printf("Successfully fetched %d files from %s/%s", len(allFiles), owner, repo)
	return allFiles, nil
}

// fetchDirectoryContents recursively fetches contents of a directory
func (c *GithubAuthConfig) fetchDirectoryContents(token, owner, repo, path string, files *[]RepoFile) error {
	contents, err := c.FetchRepoContents(token, owner, repo, path)
	if err != nil {
		return err
	}

	for _, item := range contents {
		if item.Type == "dir" {
			// Recursively fetch directory contents
			err = c.fetchDirectoryContents(token, owner, repo, item.Path, files)
			if err != nil {
				return err
			}
		} else if item.Type == "file" {
			// Fetch file content
			content, err := c.FetchFileContent(token, item.DownloadURL)
			if err != nil {
				return err
			}

			*files = append(*files, RepoFile{
				Path:    item.Path,
				Content: string(content),
				Type:    item.Type,
				Size:    item.Size,
				SHA:     item.SHA,
			})
		}
	}

	return nil
}

// FetchExampleRepos fetches the predefined list of example repositories
func (c *GithubAuthConfig) FetchExampleRepos(lang string) ([]GithubRepo, error) {
	if c.FixedToken == "" {
		return nil, fmt.Errorf("GitHub token not configured")
	}

	var allRepos []GithubRepo

	for _, example := range ExampleRepos {

		if example.Type != lang {
			continue
		}

		// Create API URL for the specific repository
		apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s", example.Owner, example.Repo)

		// Create request
		req, err := http.NewRequest("GET", apiURL, nil)
		if err != nil {
			return nil, fmt.Errorf("error creating request for %s/%s: %v", example.Owner, example.Repo, err)
		}

		// Add headers
		req.Header.Set("Authorization", "token "+c.FixedToken)
		req.Header.Set("Accept", "application/vnd.github.v3+json")

		// Make request
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("error fetching repo %s/%s: %v", example.Owner, example.Repo, err)
		}
		defer resp.Body.Close()

		// Check status code
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("GitHub API error for %s/%s: %s", example.Owner, example.Repo, string(body))
		}

		// Parse response
		var repo GithubRepo
		if err := json.NewDecoder(resp.Body).Decode(&repo); err != nil {
			return nil, fmt.Errorf("error parsing response for %s/%s: %v", example.Owner, example.Repo, err)
		}

		repo.Args = example.Args
		repo.PortNo = example.PortNo

		allRepos = append(allRepos, repo)
		Logger.Printf("Successfully fetched repo: %s/%s", example.Owner, example.Repo)
	}

	return allRepos, nil
}

// Update FetchRepoContents to use appropriate token
func (c *GithubAuthConfig) FetchRepoContents(authToken, owner, repo, path string) ([]GithubContent, error) {
	// Get appropriate token
	token := c.getAppropriateToken(owner, repo, authToken)
	if token == "" {
		return nil, fmt.Errorf("no valid token available for %s/%s", owner, repo)
	}

	// Create API URL
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s", owner, repo, path)

	// Create request
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	// Add headers
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	// Make request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error: %s", string(body))
	}

	// Parse response
	var contents []GithubContent
	if err := json.NewDecoder(resp.Body).Decode(&contents); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return contents, nil
}

// Update FetchFileContent to use appropriate token
func (c *GithubAuthConfig) FetchFileContent(authToken, downloadURL string) ([]byte, error) {
	// Extract owner and repo from download URL
	// Example URL: https://raw.githubusercontent.com/owner/repo/...
	parts := strings.Split(downloadURL, "/")
	if len(parts) < 5 {
		return nil, fmt.Errorf("invalid download URL format")
	}
	owner := parts[3]
	repo := parts[4]

	// Get appropriate token
	token := c.getAppropriateToken(owner, repo, authToken)
	if token == "" {
		return nil, fmt.Errorf("no valid token available for %s/%s", owner, repo)
	}

	// Create request
	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	// Add headers
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Accept", "application/vnd.github.v3.raw")

	// Make request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error: %s", string(body))
	}

	// Read response body
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	return content, nil
}
