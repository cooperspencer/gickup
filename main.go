package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"

	"code.gitea.io/sdk/gitea"
	"github.com/alecthomas/kong"
	"github.com/gogs/go-gogs-client"
	"github.com/google/go-github/github"
	"github.com/gookit/color"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/xanzy/go-gitlab"
	"golang.org/x/oauth2"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/http"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/ssh"
	"gopkg.in/yaml.v2"
)

var cli struct {
	Configfile string `arg required name:"conf" help:"path to the configfile." type:"existingfile"`
}

var (
	red   = color.FgRed.Render
	green = color.FgGreen.Render
	blue  = color.FgBlue.Render
)

func ReadConfigfile(configfile string) *Conf {
	cfgdata, err := ioutil.ReadFile(configfile)

	if err != nil {
		log.Panic().Str("stage", "readconfig").Str("file", configfile).Msgf("Cannot open config file from %s", red(configfile))
	}

	t := Conf{}

	err = yaml.Unmarshal([]byte(cfgdata), &t)

	if err != nil {
		log.Panic().Str("stage", "readconfig").Str("file", configfile).Msg("Cannot map yml config file to interface, possible syntax error")
	}

	return &t
}

func Locally(repo Repo, l Local) {
	if _, err := os.Stat(l.Path); os.IsNotExist(err) {
		err := os.MkdirAll(l.Path, 0777)
		if err != nil {
			log.Panic().Str("stage", "locally").Str("path", l.Path).Msg(err.Error())
		}
	}
	os.Chdir(l.Path)
	tries := 5
	var err error
	var auth transport.AuthMethod
	if repo.Origin.SSH {
		if repo.Origin.SSHKey == "" {
			home := os.Getenv("HOME")
			repo.Origin.SSHKey = path.Join(home, ".ssh", "id_rsa")
		}
		auth, err = ssh.NewPublicKeysFromFile("git", repo.Origin.SSHKey, "")
		if err != nil {
			panic(err)
		}
	} else if repo.Token != "" {
		auth = &http.BasicAuth{
			Username: "xyz",
			Password: repo.Token,
		}
	} else if repo.Origin.Username != "" && repo.Origin.Password != "" {
		auth = &http.BasicAuth{
			Username: repo.Origin.Username,
			Password: repo.Origin.Password,
		}
	}
	for x := 1; x <= tries; x++ {
		if _, err := os.Stat(repo.Name); os.IsNotExist(err) {
			log.Info().Str("stage", "locally").Str("path", l.Path).Msgf("cloning %s", green(repo.Name))

			url := repo.Url
			if repo.Origin.SSH {
				url = repo.SshUrl
			}

			_, err = git.PlainClone(repo.Name, false, &git.CloneOptions{
				URL:          url,
				Auth:         auth,
				SingleBranch: false,
			})

			if err != nil {
				if x == tries {
					log.Panic().Str("stage", "locally").Str("path", l.Path).Msg(err.Error())
				} else {
					if strings.Contains(err.Error(), "remote repository is empty") {
						log.Warn().Str("stage", "locally").Str("path", l.Path).Msg(err.Error())
						break
					}
					log.Warn().Str("stage", "locally").Str("path", l.Path).Msgf("retry %s from %s", red(x), red(tries))
					time.Sleep(5 * time.Second)
					continue
				}
			}
		} else {
			log.Info().Str("stage", "locally").Str("path", l.Path).Msgf("opening %s locally", green(repo.Name))
			r, err := git.PlainOpen(repo.Name)
			if err != nil {
				log.Panic().Str("stage", "locally").Str("path", l.Path).Msg(err.Error())
			}
			w, err := r.Worktree()
			if err != nil {
				log.Panic().Str("stage", "locally").Str("path", l.Path).Msg(err.Error())
			}

			log.Info().Str("stage", "locally").Str("path", l.Path).Msgf("pulling %s", green(repo.Name))
			err = w.Pull(&git.PullOptions{Auth: auth, RemoteName: "origin", SingleBranch: false})
			if err != nil {
				if strings.Contains(err.Error(), "already up-to-date") {
					log.Info().Str("stage", "locally").Str("path", l.Path).Msg(err.Error())
				} else {
					if x == tries {
						log.Panic().Str("stage", "locally").Str("path", l.Path).Msg(err.Error())
					} else {
						os.RemoveAll(repo.Name)
						log.Warn().Str("stage", "locally").Str("path", l.Path).Msgf("retry %s from %s", red(x), red(tries))
						time.Sleep(5 * time.Second)
						continue
					}
				}
			}
		}

		x = 5
	}
}

func BackupGitea(r Repo, d Gitea) {
	if d.Url == "" {
		d.Url = "https://gitea.com/"
	}
	log.Info().Str("stage", "gitea").Str("url", d.Url).Msgf("mirroring %s to %s", blue(r.Name), d.Url)
	giteaclient, err := gitea.NewClient(d.Url)
	if err != nil {
		log.Panic().Str("stage", "gitea").Str("url", d.Url).Msg(err.Error())
	}
	giteaclient.SetBasicAuth(d.Token, "")
	user, _, err := giteaclient.GetMyUserInfo()
	if err != nil {
		log.Panic().Str("stage", "gitea").Str("url", d.Url).Msg(err.Error())
	}
	_, _, err = giteaclient.GetRepo(user.UserName, r.Name)
	if err != nil {
		_, _, err := giteaclient.MigrateRepo(gitea.MigrateRepoOption{RepoName: r.Name, RepoOwner: user.UserName, Mirror: true, CloneAddr: r.Url, AuthToken: r.Token})
		if err != nil {
			log.Panic().Str("stage", "gitea").Str("url", d.Url).Msg(err.Error())
		}
	}
}

func BackupGogs(r Repo, d Gogs) {
	log.Info().Str("stage", "gogs").Str("url", d.Url).Msgf("mirroring %s to %s", blue(r.Name), d.Url)
	gogsclient := gogs.NewClient(d.Url, d.Token)

	user, err := gogsclient.GetSelfInfo()
	if err != nil {
		log.Panic().Str("stage", "gogs").Str("url", d.Url).Msg(err.Error())
	}
	_, err = gogsclient.GetRepo(user.UserName, r.Name)
	if err != nil {
		_, err := gogsclient.MigrateRepo(gogs.MigrateRepoOption{RepoName: r.Name, UID: int(user.ID), Mirror: true, CloneAddr: r.Url, AuthUsername: r.Token})
		if err != nil {
			log.Panic().Str("stage", "gogs").Str("url", d.Url).Msg(err.Error())
		}
	}
}

func BackupGitlab(r Repo, d Gitlab) {
	gitlabclient := &gitlab.Client{}
	var err error
	if d.Url == "" {
		d.Url = "https://gitlab.com"
		gitlabclient, err = gitlab.NewClient(d.Token)
	} else {
		gitlabclient, err = gitlab.NewClient(d.Token, gitlab.WithBaseURL(d.Url))
	}
	log.Info().Str("stage", "gitlab").Str("url", d.Url).Msgf("mirroring %s to %s", blue(r.Name), d.Url)
	if err != nil {
		log.Panic().Str("stage", "gitlab").Str("url", d.Url).Msg(err.Error())
	}

	True := true
	opt := gitlab.ListProjectsOptions{Search: &r.Name, Owned: &True}
	projects, _, err := gitlabclient.Projects.ListProjects(&opt)
	if err != nil {
		log.Panic().Str("stage", "gitlab").Str("url", d.Url).Msg(err.Error())
	}

	found := false
	for _, p := range projects {
		if p.Name == r.Name {
			found = true
		}
	}

	if !found {
		if r.Token != "" {
			splittedurl := strings.Split(r.Url, "//")
			r.Url = fmt.Sprintf("%s//%s@%s", splittedurl[0], r.Token, splittedurl[1])
		}
		opts := &gitlab.CreateProjectOptions{Mirror: &True, ImportURL: &r.Url, Name: &r.Name}
		_, _, err := gitlabclient.Projects.CreateProject(opts)
		if err != nil {
			log.Panic().Str("stage", "gitlab").Str("url", d.Url).Msg(err.Error())
		}
	}
}

func Backup(repos []Repo, conf *Conf) {
	for _, r := range repos {
		log.Info().Str("stage", "backup").Msgf("starting backup for %s", r.Url)
		for _, d := range conf.Destination.Local {
			Locally(r, d)
		}
		for _, d := range conf.Destination.Gitea {
			BackupGitea(r, d)
		}
		for _, d := range conf.Destination.Gogs {
			BackupGogs(r, d)
		}
		for _, d := range conf.Destination.Gitlab {
			BackupGitlab(r, d)
		}
	}
}

func getGithub(conf *Conf) []Repo {
	repos := []Repo{}
	for _, repo := range conf.Source.Github {
		log.Info().Str("stage", "github").Str("url", "https://github.com").Msgf("grabbing the repositories from %s", repo.User)
		client := &github.Client{}
		opt := &github.RepositoryListOptions{}
		opt.PerPage = 50
		i := 0
		githubrepos := []*github.Repository{}
		for {
			opt.Page = i
			if repo.Token == "" {
				client = github.NewClient(nil)
			} else {
				ts := oauth2.StaticTokenSource(
					&oauth2.Token{AccessToken: repo.Token},
				)
				tc := oauth2.NewClient(context.TODO(), ts)
				client = github.NewClient(tc)
			}
			repos, _, err := client.Repositories.List(context.TODO(), repo.User, opt)
			if err != nil {
				log.Panic().Str("stage", "github").Str("url", "https://github.com").Msg(err.Error())
			}
			if len(repos) == 0 {
				break
			}
			githubrepos = append(githubrepos, repos...)
			i++
		}

		for _, r := range githubrepos {
			repos = append(repos, Repo{Name: r.GetName(), Url: r.GetCloneURL(), SshUrl: r.GetSSHURL(), Token: repo.Token, Defaultbranch: r.GetDefaultBranch(), Origin: repo})
		}
	}
	return repos
}

func getGitea(conf *Conf) []Repo {
	repos := []Repo{}
	for _, repo := range conf.Source.Gitea {
		if repo.Url == "" {
			repo.Url = "https://gitea.com"
		}
		log.Info().Str("stage", "gitea").Str("url", repo.Url).Msgf("grabbing repositories from %s", repo.User)
		opt := gitea.ListReposOptions{}
		opt.PageSize = 50
		i := 0
		gitearepos := []*gitea.Repository{}
		for {
			opt.Page = i
			client, err := gitea.NewClient(repo.Url)
			if err != nil {
				log.Panic().Str("stage", "gitea").Str("url", repo.Url).Msg(err.Error())
			}
			if repo.Token != "" {
				client.SetBasicAuth(repo.Token, "")
			}
			repos, _, err := client.ListUserRepos(repo.User, opt)
			if err != nil {
				log.Panic().Str("stage", "gitea").Str("url", repo.Url).Msg(err.Error())
			}
			if len(repos) == 0 {
				break
			}
			gitearepos = append(gitearepos, repos...)
			i++
		}

		for _, r := range gitearepos {
			repos = append(repos, Repo{Name: r.Name, Url: r.CloneURL, SshUrl: r.SSHURL, Token: repo.Token, Defaultbranch: r.DefaultBranch, Origin: repo.Github})
		}
	}
	return repos
}

func getGogs(conf *Conf) []Repo {
	repos := []Repo{}
	for _, repo := range conf.Source.Gogs {
		log.Info().Str("stage", "gogs").Str("url", repo.Url).Msgf("grabbing repositories from %s", repo.User)
		client := gogs.NewClient(repo.Url, repo.Token)
		gogsrepos, err := client.ListUserRepos(repo.User)
		if err != nil {
			log.Panic().Str("stage", "gogs").Str("url", repo.Url).Msg(err.Error())
		}

		for _, r := range gogsrepos {
			repos = append(repos, Repo{Name: r.Name, Url: r.CloneURL, SshUrl: r.SSHURL, Token: repo.Token, Defaultbranch: r.DefaultBranch, Origin: repo.Github})
		}
	}
	return repos
}

func getGitlab(conf *Conf) []Repo {
	repos := []Repo{}
	for _, repo := range conf.Source.Gitlab {
		if repo.Url == "" {
			repo.Url = "https://gitlab.com"
		}
		log.Info().Str("stage", "gitlab").Str("url", repo.Url).Msgf("grabbing repositories from %s", repo.User)
		gitlabrepos := []*gitlab.Project{}
		client, err := gitlab.NewClient(repo.Token, gitlab.WithBaseURL(repo.Url))
		if err != nil {
			log.Panic().Str("stage", "gitlab").Str("url", repo.Url).Msg(err.Error())
		}
		opt := &gitlab.ListProjectsOptions{}
		users, _, err := client.Users.ListUsers(&gitlab.ListUsersOptions{Username: &repo.User})
		if err != nil {
			log.Panic().Str("stage", "gitlab").Str("url", repo.Url).Msg(err.Error())
		}

		opt.PerPage = 50
		i := 0
		for _, user := range users {
			if user.Username == repo.User {
				for {
					projects, _, err := client.Projects.ListUserProjects(user.ID, opt)
					if err != nil {
						log.Panic().Str("stage", "gitlab").Str("url", repo.Url).Msg(err.Error())
					}
					if len(projects) == 0 {
						break
					}
					gitlabrepos = append(gitlabrepos, projects...)
					i++
					opt.Page = i
				}
			}
		}
		for _, r := range gitlabrepos {
			repos = append(repos, Repo{Name: r.Name, Url: r.HTTPURLToRepo, SshUrl: r.SSHURLToRepo, Token: repo.Token, Defaultbranch: r.DefaultBranch, Origin: repo.Github})
		}
	}
	return repos
}

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	kong.Parse(&cli)
	log.Info().Str("file", cli.Configfile).Msgf("Reading %s", green(cli.Configfile))
	conf := ReadConfigfile(cli.Configfile)

	// Github
	repos := getGithub(conf)
	Backup(repos, conf)

	// Gitea
	repos = getGitea(conf)
	Backup(repos, conf)

	// Gogs
	repos = getGogs(conf)
	Backup(repos, conf)

	// Gitlab
	repos = getGitlab(conf)
	Backup(repos, conf)
}
