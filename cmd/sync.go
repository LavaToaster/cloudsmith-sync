package cmd

import (
	"fmt"
	"github.com/Lavoaster/cloudsmith-sync/cloudsmith"
	"github.com/Lavoaster/cloudsmith-sync/composer"
	config2 "github.com/Lavoaster/cloudsmith-sync/config"
	"github.com/Lavoaster/cloudsmith-sync/git"
	"github.com/briandowns/spinner"
	"github.com/spf13/cobra"
	git2 "gopkg.in/libgit2/git2go.v27"
	"strconv"
	"strings"
	"time"
)

func init() {
	rootCmd.AddCommand(runCmd)
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Performs a full sync on repositories",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Repository Sync")
		fmt.Println("===============")
		fmt.Println()

		totalRepositories := strconv.Itoa(len(config.Repositories))
		fmt.Println("Syncing " + totalRepositories + " repositories")

		client := cloudsmith.NewClient(config.ApiKey)

		fmt.Print("Loading existing packages...")

		s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
		s.Prefix = " "
		s.FinalMSG = "Done\n\n"
		s.Start()

		err := client.LoadPackages(config.Owner, config.TargetRepository)
		exitOnError(err)

		s.Stop()

		for _, repoCfg := range config.Repositories {
			// Repo Config
			repoDir, err := git.GitUrlToDirectory(repoCfg.Url)
			repoPath := config.GetRepoPath(repoDir)
			exitOnError(err)

			processLine := "Processing repository: " + repoCfg.Url
			outer := strings.Repeat("=", len(processLine))

			fmt.Println(outer)
			fmt.Println(processLine)
			fmt.Println(outer)
			fmt.Println()

			// Clone Repo
			repo, err := git.CloneOrOpenAndUpdate(repoCfg.Url, repoPath)
			exitOnError(err)

			// Loop through tags first
			tags, err := repo.Tags.List()

			for _, tag := range tags {
				oid, err := git.CheckoutTag(repo, tag)
				exitOnError(err)

				processPackage(client, &repoCfg, repoPath, tag, false, oid)
			}

			branchIterator, err := repo.NewBranchIterator(git2.BranchRemote)
			exitOnError(err)

			err = branchIterator.ForEach(func(branch *git2.Branch, _ git2.BranchType) error {
				branchName, err := branch.Name()
				exitOnError(err)

				oid, err := git.CheckoutBranch(repo, branchName)
				exitOnError(err)

				processPackage(client, &repoCfg, repoPath, branchName, true, oid)

				return nil
			})
			exitOnError(err)

			fmt.Println()
		}
	},
}

func processPackage(
	client *cloudsmith.Client,
	repoCfg *config2.Repository,
	repoPath, name string,
	isBranch bool,
	oid *git2.Oid,
) {
	version := composer.DeriveVersion(name, isBranch)
	composerData, err := composer.LoadFile(repoPath)
	exitOnError(err)

	packageName := composerData["name"].(string)

	fmt.Printf("Processing %s@%s...", packageName, version)

	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Prefix = " "
	s.Start()

	if client.IsAwareOfPackage(packageName, version) {
		s.FinalMSG = "already exists\n"
		s.Stop()
		return
	}

	createComposerPackage(
		client,
		composerData,
		repoCfg.Url,
		repoPath,
		version,
		oid,
		repoCfg.PublishSource,
	)

	s.FinalMSG = "done\n"
	s.Stop()
}

func createComposerPackage(
	client *cloudsmith.Client,
	composerData composer.ComposerFile,
	repoUrl, repoPath, version string,
	oid *git2.Oid,
	publishSource bool,
) {
	commitRef := oid.String()

	var source *composer.Source

	if publishSource {
		source = &composer.Source{
			Url:       repoUrl,
			Type:      "git",
			Reference: commitRef,
		}
	}

	// Mutate composer.json file
	err := composer.MutateComposerFile(repoPath, version, source)
	exitOnError(err)

	// Extract Info from the composer file
	packageName := composerData["name"]
	packageNameParts := strings.Split(packageName.(string), "/")
	namespace := packageNameParts[0]
	name := packageNameParts[1]

	artifactName := fmt.Sprintf("%v-%v-%v.zip", namespace, name, commitRef)
	artifactPath := config.GetArtifactPath(artifactName)

	// Create archive file
	err = git.CreateArtifactFromRepository(repoPath, artifactPath)
	exitOnError(err)

	if !dryRun {
		// Upload archive to cloudsmith
		_, err = client.UploadComposerPackage(config.Owner, config.TargetRepository, artifactPath)
		exitOnError(err)
	}
}
