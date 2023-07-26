package main

import (
	"context"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/google/go-github/v53/github"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

const (
	mqscFileExtension = ".mqsc"
)

var (
	githubToken        string
	githubRepoOwner    string
	githubRepoName     string
	githubRepoPath     string
	githubRepoRef      string
	githubPollInterval string
	queueManagerName   string
)

func main() {
	ctx := context.Background()

	// Read configuration from the environment
	githubToken = os.Getenv("GITHUB_TOKEN")
	githubRepoOwner = os.Getenv("GITHUB_REPO_OWNER")
	githubRepoName = os.Getenv("GITHUB_REPO_NAME")
	githubRepoPath = os.Getenv("GITHUB_REPO_PATH")
	githubRepoRef = os.Getenv("GITHUB_REPO_REF")
	githubPollInterval = os.Getenv("GITHUB_POLL_INTERVAL")
	queueManagerName = os.Getenv("QUEUE_MANAGER_NAME")

	// Configure logs
	log := logrus.New()
	log.SetOutput(os.Stdout)
	log.SetFormatter(&logrus.JSONFormatter{TimestampFormat: time.RFC3339})
	log.SetLevel(logrus.InfoLevel)

	sleep, err := time.ParseDuration(githubPollInterval)
	if err != nil {
		log.Fatal(err)
	}

	// Check if we have the runmqsc utility installed
	runMqscPath, err := exec.LookPath("runmqsc")
	if err != nil {
		log.Fatal(err)
	}

	// Set up GitHub client
	oauthToken := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: githubToken})
	oauthClient := oauth2.NewClient(ctx, oauthToken)
	githubClient := github.NewClient(oauthClient)

	log.Infoln("starting mq config sync loop")

	for {

		mqscPaths, err := discoverMqscFiles(ctx, githubClient, githubRepoOwner, githubRepoName, githubRepoPath, githubRepoRef)
		if err != nil {
			log.Error(err)
		} else {
			log.Infof("discovered %d mqsc files", len(mqscPaths))
		}

		for _, path := range mqscPaths {
			readCloser, resp, err := githubClient.Repositories.DownloadContents(ctx, githubRepoOwner, githubRepoName,
				path, &github.RepositoryContentGetOptions{Ref: githubRepoRef})
			if err != nil {
				body, err := io.ReadAll(resp.Body)
				log.Error(err, string(body))
			}

			mqsc, err := io.ReadAll(readCloser)
			if err != nil {
				log.Error(err)
			}

			log.Infof("running commands in mqsc file: %s", path)
			commandOutput, err := runMqscCommands(runMqscPath, queueManagerName, string(mqsc))
			if err != nil {
				log.Errorf("error running mqsc commands: %s, output: %s", err, commandOutput)
			}
			output := ""
			lines := strings.Split(commandOutput, "\n")
			//  remove trailing \n
			lines = lines[:len(lines)-1]
			numLines := len(lines)
			if numLines >= 3 {
				lastThreeLines := lines[numLines-3:]
				for _, line := range lastThreeLines {
					output = output + strings.ToLower(line)
				}
				log.Infof("mqsc commands ran successfully, output: %s", output)
			} else {
				log.Warnf("unexpected command output: %s", commandOutput)
			}
		}

		time.Sleep(sleep)
	}

}

func discoverMqscFiles(ctx context.Context, gh *github.Client, owner, repo, path, ref string) (mqscPaths []string, err error) {

	fileContent, dirContent, _, err := gh.Repositories.GetContents(ctx, owner, repo, path,
		&github.RepositoryContentGetOptions{Ref: githubRepoRef})
	if err != nil {
		return nil, err
	}

	if fileContent != nil {
		if strings.HasSuffix(fileContent.GetPath(), mqscFileExtension) {
			mqscPaths = append(mqscPaths, fileContent.GetPath())
			return mqscPaths, nil
		}
	}

	for _, c := range dirContent {
		if *c.Type == "file" {
			if strings.HasSuffix(c.GetPath(), mqscFileExtension) {
				mqscPaths = append(mqscPaths, c.GetPath())
			}
		}
		if *c.Type == "dir" {
			mqscPaths, err := discoverMqscFiles(ctx, gh, owner, repo, c.GetPath(), ref)
			if err != nil {
				return nil, err
			}
			return mqscPaths, nil
		}
	}
	return mqscPaths, nil
}

func runMqscCommands(runMqscPath string, queueManager string, commandsString string) (output string, err error) {
	cmd := exec.Command(runMqscPath, queueManager)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", err
	}

	go func() {
		defer stdin.Close()
		io.WriteString(stdin, commandsString)
	}()

	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), err
	}

	return string(out), nil
}
