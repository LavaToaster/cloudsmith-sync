package cmd

import (
	"fmt"
	"github.com/Lavoaster/cloudsmith-sync/cloudsmith"
	"github.com/Lavoaster/cloudsmith-sync/composer"
	config2 "github.com/Lavoaster/cloudsmith-sync/config"
	"github.com/Lavoaster/cloudsmith-sync/git"
	"github.com/briandowns/spinner"
	"github.com/spf13/cobra"
	git2 "gopkg.in/src-d/go-git.v4"
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
		git.Config = config

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

			// Get Remote
			remote, err := repo.Remote("origin")
			exitOnError(err)

			auth, err := git.GetAuth()
			exitOnError(err)

			refList, err := remote.List(&git2.ListOptions{Auth: auth})
			exitOnError(err)

			worktree, err := repo.Worktree()
			exitOnError(err)

			for _, ref := range refList {
				isBranch := strings.HasPrefix(ref.Name().String(), "refs/heads/")
				isTag := strings.HasPrefix(ref.Name().String(), "refs/tags/")

				if !isBranch && !isTag {
					continue
				}

				// Tags
				if isTag {
					_, err := git.CheckoutTag(repo, worktree, ref)

					if err != nil {
						fmt.Printf("Skipping tag %v - %v\n", ref, err.Error())
						continue
					}
				}

				// Branch
				if isBranch {
					_, err := git.CheckoutBranch(repo, worktree, ref)

					if err != nil {
						fmt.Printf("Skipping branch %v - %v\n", ref, err.Error())
						continue
					}
				}

				processPackage(client, &repoCfg, repoPath, ref.Name().Short(), isBranch, ref.Hash().String())

				worktree.Reset(&git2.ResetOptions{
					Mode: git2.HardReset,
				})
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
	commitRef string,
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

			s.Suffix = " Waiting for package to be deleted"

			for {
				exists, err := client.RemoteCheckPackageExists(config.Owner, config.TargetRepository, packageName, version)
				exitOnError(err)

				if !exists {
					s.Suffix = ""
					break
				}

				time.Sleep(2 * time.Second)
			}
		} else {
			s.FinalMSG = "already exists\n"
			s.Stop()
			return
		}
	}

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
