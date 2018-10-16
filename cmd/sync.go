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

var Target string

func init() {
	runCmd.Flags().StringVarP(&Target, "target", "t", "both", "Target [tags, branches, both]")
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

			if Target == "tags" || Target == "both" {
				// Loop through tags first
				tags, err := repo.Tags.List()
				exitOnError(err)

				for _, tag := range tags {
					oid, err := git.CheckoutTag(repo, tag)
					exitOnError(err)

					processPackage(client, &repoCfg, repoPath, tag, false, oid)
				}
			}

			if (Target == "branches" || Target == "both") && false {

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
			}

			fmt.Println()
		}
	},
}

func processPackage(
	client *cloudsmith.Client,
	repoCfg *config2.Repository,
	repoPath, branchOrTagName string,
	isBranch bool,
	oid *git2.Oid,
) {
	composerData, err := composer.LoadFile(repoPath)
	exitOnError(err)

	packageName := composerData["name"].(string)

	version, normalisedVersion, err := composer.DeriveVersion(branchOrTagName, isBranch)

	if err != nil {
		fmt.Printf("Skipping %s@%s due to %s...\n", packageName, branchOrTagName, err)
		return
	}

	fmt.Printf("Processing %s@%s...", packageName, version)

	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Prefix = " "
	s.Start()

	if client.IsAwareOfPackage(packageName, version) {
		if isBranch {
			client.DeletePackageIfExists(config.Owner, config.TargetRepository, packageName, version)
		} else {
			s.FinalMSG = "already exists\n"
			s.Stop()
			return
		}
	}

	commitRef := oid.String()

	var source *composer.Source

	if repoCfg.PublishSource {
		source = &composer.Source{
			Url:       repoCfg.Url,
			Type:      "git",
			Reference: commitRef,
		}
	}

	// Mutate composer.json file
	err = composer.MutateComposerFile(repoPath, version, normalisedVersion, source)
	exitOnError(err)

	// Extract Info from the composer file
	packageNameParts := strings.Split(packageName, "/")
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

	s.FinalMSG = "done\n"
	s.Stop()
}
