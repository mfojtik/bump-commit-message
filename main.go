package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/google/go-github/v28/github"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"

	"github.com/mfojtik/git-bump-commit-message/pkg/golang"
	"github.com/mfojtik/git-bump-commit-message/pkg/resolve"
)

type module struct {
	name             string
	repository       string
	currentRevision  string
	previousRevision string
}

func readModules(goModBytes []byte) ([]module, error) {
	parsedModFile, err := golang.ParseModFile("go.mod", goModBytes, nil)
	if err != nil {
		return nil, err
	}
	modules := []module{}
	for _, r := range parsedModFile.Require {
		modules = append(modules, module{
			name:            r.Mod.Path,
			repository:      r.Mod.Path,
			currentRevision: r.Mod.Version,
		})
	}
	result := []module{}
	for _, m := range modules {
		foundReplace := false
		for _, replace := range parsedModFile.Replace {
			if m.name == replace.Old.Path {
				foundReplace = true
				result = append(result, module{
					name:            m.name,
					repository:      replace.New.Path,
					currentRevision: replace.New.Version,
				})
				break
			}
		}
		if foundReplace {
			continue
		}
		result = append(result, m)
	}
	return result, nil
}

func compareModules(new, old []module, filter []string) []module {
	result := []module{}
	for _, newModule := range new {
		previousVersion := ""
		for _, oldModule := range old {
			if oldModule.name != newModule.name {
				continue
			}
			if oldModule.currentRevision != newModule.currentRevision {
				previousVersion = oldModule.currentRevision
				break
			}
		}
		if len(previousVersion) == 0 {
			continue
		}
		if len(filter) > 0 {
			matchFilter := false
			for _, f := range filter {
				if !strings.HasPrefix(newModule.name, f) {
					matchFilter = true
					break
				}
			}
			if !matchFilter {
				continue
			}
		}
		result = append(result, module{
			name:             newModule.name,
			repository:       newModule.repository,
			currentRevision:  newModule.currentRevision,
			previousRevision: previousVersion,
		})
	}
	return result
}

func sanitizeCommitMessage(message string) string {
	firstLineTrimmed := strings.TrimSpace(strings.Split(strings.TrimSuffix(message, "\n"), "\n")[0])
	firstLineLength := len(firstLineTrimmed)
	if firstLineLength < 120 {
		return firstLineTrimmed
	}
	return firstLineTrimmed[0:120]
}

func listCommits(modulePath string, fromCommit, toCommit string, oauthClient *http.Client) ([]string, error) {
	client := github.NewClient(oauthClient)
	owner, repo := resolve.GetGithubOwnerAndRepo(resolve.RepositoryModulePath(modulePath))
	commits, _, err := client.Repositories.ListCommits(context.TODO(), owner, repo, &github.CommitsListOptions{
		SHA: fromCommit,
	})
	if err != nil {
		return nil, err
	}
	result := []string{}
	for _, c := range commits {
		if strings.HasPrefix(c.GetSHA(), toCommit) {
			break
		}
		if strings.HasPrefix(c.GetCommit().GetMessage(), "Merge pull request") {
			continue
		}
		result = append(result, fmt.Sprintf("%s/%s@%s: %s", owner, repo, c.GetSHA()[0:8], sanitizeCommitMessage(c.GetCommit().GetMessage())))
	}
	return result, nil
}

func getCommitFromVersion(version string) string {
	parts := strings.Split(version, "-")
	if len(parts) != 3 {
		return version
	}
	return parts[2]
}

var (
	filter     []string
	baseBranch = "master"
)

func init() {
	rootCmd.PersistentFlags().StringSliceVar(&filter, "paths", filter, "A comma separated list of import paths to include commit messages for")
	rootCmd.PersistentFlags().StringVar(&baseBranch, "base-branch", baseBranch, "A branch name to use as a base when comparing the previous go.mod")
}

var rootCmd = &cobra.Command{
	Use:   "git-bump-commit-message",
	Short: "A commit message generator for go.mod bumps",
	Run: func(cmd *cobra.Command, args []string) {
		run(args)
	},
}

func run(args []string) {
	// we need Github token so we won't be rate-limited when interacting with github API
	githubToken := os.Getenv("GITHUB_TOKEN")
	if len(githubToken) == 0 {
		log.Fatal("Set the GITHUB_TOKEN environment variable to personal access token created by https://github.com/settings/tokens")
	}
	githubClient := oauth2.NewClient(context.TODO(), oauth2.StaticTokenSource(&oauth2.Token{AccessToken: githubToken}))

	// read the go.mod from upstream/master branch
	upstreamModContent, err := exec.Command("git", "show", "upstream/"+baseBranch+":go.mod").CombinedOutput()
	if err != nil {
		log.Fatalf("Unable to read go.mod file from upstream/master branch: %v (%s)", err, string(upstreamModContent))
	}
	upstreamModules, err := readModules(upstreamModContent)
	if err != nil {
		log.Fatalf("Unable to parse upstream/master go.mod file: %v", err)
	}

	// read the go.mod from local branch
	currentModContent, err := ioutil.ReadFile("go.mod")
	if err != nil {
		log.Fatalf("Unable to read go.mod file: %v", err)
	}
	currentModules, err := readModules(currentModContent)
	if err != nil {
		log.Fatalf("Unable to parse upstream/master go.mod file: %v", err)
	}

	// compare upstream/master and local and get list of modules that were changed in go.mod
	updatedModules := compareModules(upstreamModules, currentModules, filter)

	if len(updatedModules) == 0 {
		log.Fatal("No modules were updated in this branch")
	}

	log.Printf("m: %#v", updatedModules)

	fmt.Fprintf(os.Stdout, "bump(*): vendor update\n\n")
	for _, m := range updatedModules {
		commitMessages, err := listCommits(m.name, getCommitFromVersion(m.previousRevision), getCommitFromVersion(m.currentRevision), githubClient)
		if err != nil {
			log.Fatalf("Unable to list commits for %q: %v", m.name, err)
		}
		for _, message := range commitMessages {
			fmt.Fprintf(os.Stdout, "* %s\n", message)
		}
	}
	fmt.Fprintf(os.Stdout, "\n")
}

func main() {
	rootCmd.Execute()
}
