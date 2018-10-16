package git

import (
	"github.com/Lavoaster/cloudsmith-sync/config"
	"gopkg.in/libgit2/git2go.v27"
	"log"
	"os"
)

var remoteCallbacks = &git.RemoteCallbacks{
	CertificateCheckCallback: certificateCheckCallback,
	CredentialsCallback:      credentialsCallback,
}

var Config *config.Config

// TODO: Allow the config to specify call back :)
func credentialsCallback(url string, usernameFromUrl string, allowedTypes git.CredType) (git.ErrorCode, *git.Cred) {
	ret, cred := git.NewCredSshKey(usernameFromUrl, Config.SshKey+".pub", Config.SshKey+Config.SshKey, Config.SshKeyPassphrase)

	return git.ErrorCode(ret), &cred
}

// TODO: Host key check _shooooould_ probably be something better than just accept it :)
func certificateCheckCallback(cert *git.Certificate, valid bool, hostname string) git.ErrorCode {
	return 0
}

func CloneOrOpenAndUpdate(url, path string) (*git.Repository, error) {
	if _, err := os.Stat(path); err == nil {
		return OpenAndFetch(path)
	}

	return Clone(url, path)
}

func Clone(url, path string) (*git.Repository, error) {
	return git.Clone(url, path, &git.CloneOptions{
		FetchOptions: &git.FetchOptions{
			RemoteCallbacks: *remoteCallbacks,
		},
		CheckoutOpts: &git.CheckoutOpts{
			// We don't care about any changes in the repo so we just let them get discarded.
			Strategy: git.CheckoutForce,
		},
	})
}

func OpenAndFetch(path string) (*git.Repository, error) {
	repo, err := git.OpenRepository(path)

	if err != nil {
		return nil, err
	}

	remote, err := repo.Remotes.Lookup("origin")

	if err != nil {
		return nil, err
	}

	fetchOptions := &git.FetchOptions{
		RemoteCallbacks: *remoteCallbacks,
	}

	err = remote.Fetch([]string{}, fetchOptions, "")

	if err != nil {
		return nil, err
	}

	return repo, nil
}

func CheckoutBranch(repo *git.Repository, branchName string) (*git.Oid, error) {
	// Getting the reference for the remote branch
	remoteBranch, err := repo.LookupBranch(branchName, git.BranchRemote)
	if err != nil {
		log.Print("Failed to find remote branch: " + branchName)
		return nil, err
	}
	defer remoteBranch.Free()

	return remoteBranch.Target(), repo.SetHeadDetached(remoteBranch.Target())
}

func CheckoutTag(repo *git.Repository, tagName string) (*git.Oid, error) {
	// Getting the reference for the remote branch
	remoteBranch, err := repo.References.Lookup("refs/tags/" + tagName)
	if err != nil {
		log.Print("Failed to find tag: " + tagName)
		return nil, err
	}
	defer remoteBranch.Free()

	return remoteBranch.Target(), repo.SetHeadDetached(remoteBranch.Target())
}
