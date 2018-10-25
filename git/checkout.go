package git

import (
	"github.com/Lavoaster/cloudsmith-sync/config"
	"gopkg.in/src-d/go-git.v4"
	config2 "gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/ssh"
	"os"
)

var Config *config.Config

func CloneOrOpenAndUpdate(url, path string) (*git.Repository, error) {
	if _, err := os.Stat(path); err == nil {
		return OpenAndFetch(path)
	}

	return Clone(url, path)
}

func GetAuth() (*ssh.PublicKeys, error) {
	return ssh.NewPublicKeysFromFile("git", Config.SshKey, "")
}

func Clone(url, path string) (*git.Repository, error) {
	auth, err := GetAuth()

	if err != nil {
		return nil, err
	}

	git.PlainClone(path, false, &git.CloneOptions{
		URL:  url,
		Auth: auth,
	})

	return OpenAndFetch(path)
}

func OpenAndFetch(path string) (*git.Repository, error) {
	repo, err := git.PlainOpen(path)

	if err != nil {
		return nil, err
	}

	auth, err := GetAuth()

	if err != nil {
		return nil, err
	}

	err = repo.Fetch(&git.FetchOptions{
		RefSpecs: []config2.RefSpec{
			"refs/tags/*:refs/tags/*",
			"refs/heads/*:refs/heads/*",
		},
		Auth: auth,
	})

	if err != nil && err != git.NoErrAlreadyUpToDate {
		return nil, err
	}

	return repo, nil
}

func CheckoutBranch(repo *git.Repository, worktree *git.Worktree, ref *plumbing.Reference) (string, error) {
	err := worktree.Checkout(&git.CheckoutOptions{
		Branch: ref.Target(),
	})

	if err != nil {
		return "", err
	}

	head, err := repo.Head()

	if err != nil {
		return "", err
	}

	return head.Hash().String(), nil
}

func CheckoutTag(repo *git.Repository, worktree *git.Worktree, ref *plumbing.Reference) (string, error) {
	hash := ref.Hash()

	// test for annotated ref
	tagObject, err := repo.TagObject(ref.Hash())

	if err == nil {
		hash = tagObject.Target
	}

	err = worktree.Checkout(&git.CheckoutOptions{
		Hash: hash,
	})

	if err != nil {
		return "", err
	}

	return ref.Hash().String(), nil
}
