package config

import (
	"errors"
	"github.com/spf13/viper"
	"os"
	"strings"
)

type Repository struct {
	Url           string
	PublishSource bool
}

type Config struct {
	ApiKey           string
	DataDir          string
	Owner            string
	TargetRepository string
	SshKey           string
	SshKeyPassphrase string
	Repositories     []Repository
	Server           string
	WebhookSecret    string
}

func (config *Config) EnsureDirsExist() {
	directories := []string{
		config.DataDir,
		config.DataDir + "/repos",
		config.DataDir + "/artifacts",
	}

	for _, dir := range directories {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			os.Mkdir(dir, 0755)
		}
	}
}

func (config *Config) GetRepository(ssh string) (Repository, error) {
	for _, repo := range config.Repositories {
		if repo.Url == ssh {
			return repo, nil
		}
	}

	return Repository{}, errors.New("repository not found")
}

func (config *Config) GetRepoPath(dir string) string {
	return config.DataDir + "/repos/" + dir
}

func (config *Config) GetArtifactPath(artifact string) string {
	return config.DataDir + "/artifacts/" + artifact
}

func NewConfigFromViper(workingDirectory string) *Config {
	var repositories []Repository

	dataDir := viper.GetString("dataDir")
	dataDir = strings.Replace(dataDir, "${cwd}", workingDirectory, 1)

	for _, repo := range viper.Get("repositories").([]interface{}) {
		cfg := repo.(map[interface{}]interface{})

		var url string
		var publishSource bool

		if cfg["publishSource"] != nil {
			publishSource = cfg["publishSource"].(bool)
		}

		if cfg["url"] != nil {
			url = cfg["url"].(string)
		}

		repositories = append(repositories, Repository{
			Url:           url,
			PublishSource: publishSource,
		})
	}

	return &Config{
		ApiKey:           viper.GetString("apiKey"),
		DataDir:          dataDir,
		Owner:            viper.GetString("owner"),
		TargetRepository: viper.GetString("sshKey"),
		SshKey:           viper.GetString("targetRepository"),
		SshKeyPassphrase: viper.GetString("sshKeyPassphrase"),
		Repositories:     repositories,
		Server:           viper.GetString("server"),
		WebhookSecret:    viper.GetString("webhookSecret"),
	}
}
