package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"code.gitea.io/sdk/gitea"
	"github.com/alecthomas/kong"
	git "github.com/gogs/git-module"
	"github.com/gogs/go-gogs-client"
	"github.com/google/go-github/github"
	"github.com/gookit/color"
	"github.com/xanzy/go-gitlab"
	"golang.org/x/oauth2"
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
		panic("Cannot open config file from " + red(configfile))
	}

	t := Conf{}

	err = yaml.Unmarshal([]byte(cfgdata), &t)

	if err != nil {
		panic("Cannot map yml config file to interface, possible syntax error")
	}

	return &t
}

func Locally(repo Repo, path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := os.MkdirAll(path, 0777)
		if err != nil {
			panic(err)
		}
	}
	os.Chdir(path)
	tries := 5

	for x := 1; x <= tries; x++ {
		if _, err := os.Stat(repo.Name); os.IsNotExist(err) {
			fmt.Printf("cloning %s\n", green(repo.Name))
			err := git.Clone(repo.Url, repo.Name, git.CloneOptions{Quiet: false, Timeout: 5 * time.Minute})
			if err != nil {
				if x == tries {
					panic(err)
				} else {
					fmt.Printf("retry %s from %s\n", red(x), red(tries))
					time.Sleep(5 * time.Second)
					continue
				}
			}
		} else {
			fmt.Printf("pulling %s\n", green(repo.Name))
			r, err := git.Open(repo.Name)
			if err != nil {
				if x == tries {
					panic(err)
				} else {
					os.RemoveAll(repo.Name)
					fmt.Printf("retry %s from %s\n", red(x), red(tries))
					time.Sleep(5 * time.Second)
					continue
				}
			}
			err = r.Pull(git.PullOptions{All: true, Branch: repo.Defaultbranch})
			if err != nil {
				if x == tries {
					panic(err)
				} else {
					os.RemoveAll(repo.Name)
					fmt.Printf("retry %s from %s\n", red(x), red(tries))
					time.Sleep(5 * time.Second)
					continue
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
	fmt.Printf("mirroring %s to %s\n", blue(r.Name), d.Url)
	giteaclient, err := gitea.NewClient(d.Url)
	if err != nil {
		panic(err)
	}
	giteaclient.SetBasicAuth(d.Token, "")
	user, _, err := giteaclient.GetMyUserInfo()
	if err != nil {
		panic(err)
	}
	_, _, err = giteaclient.GetRepo(user.UserName, r.Name)
	if err != nil {
		_, _, err := giteaclient.MigrateRepo(gitea.MigrateRepoOption{RepoName: r.Name, RepoOwner: user.UserName, Mirror: true, CloneAddr: r.Url, AuthToken: r.Token})
		if err != nil {
			panic(err)
		}
	}
}

func BackupGogs(r Repo, d Gogs) {
	fmt.Printf("mirroring %s to %s\n", blue(r.Name), d.Url)
	gogsclient := gogs.NewClient(d.Url, d.Token)

	user, err := gogsclient.GetSelfInfo()
	if err != nil {
		panic(err)
	}
	_, err = gogsclient.GetRepo(user.UserName, r.Name)
	if err != nil {
		_, err := gogsclient.MigrateRepo(gogs.MigrateRepoOption{RepoName: r.Name, UID: int(user.ID), Mirror: true, CloneAddr: r.Url, AuthUsername: r.Token})
		if err != nil {
			panic(err)
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
	fmt.Printf("mirroring %s to %s\n", blue(r.Name), d.Url)
	if err != nil {
		panic(err)
	}

	True := true
	opt := gitlab.ListProjectsOptions{Search: &r.Name, Owned: &True}
	projects, _, err := gitlabclient.Projects.ListProjects(&opt)
	if err != nil {
		panic(err)
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
			panic(err)
		}
	}
}

func Backup(repos []Repo, conf *Conf) {
	for _, r := range repos {
		fmt.Println(r.Url)
		for _, d := range conf.Destination.Local {
			fmt.Printf("cloning %s to local\n", blue(r.Name))
			Locally(r, d.Path)
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
		fmt.Printf("github.com: grabbing the repositories from %s\n", repo.User)
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
				panic(err)
			}
			if len(repos) == 0 {
				break
			}
			githubrepos = append(githubrepos, repos...)
			i++
		}

		for _, r := range githubrepos {
			repos = append(repos, Repo{Name: r.GetName(), Url: r.GetCloneURL(), Token: repo.Token, Defaultbranch: r.GetDefaultBranch()})
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
		fmt.Printf("%s: grabbing repositories from %s\n", repo.Url, repo.User)
		opt := gitea.ListReposOptions{}
		opt.PageSize = 50
		i := 0
		gitearepos := []*gitea.Repository{}
		for {
			opt.Page = i
			client, err := gitea.NewClient(repo.Url)
			if err != nil {
				panic(err)
			}
			if repo.Token != "" {
				client.SetBasicAuth(repo.Token, "")
			}
			repos, _, err := client.ListUserRepos(repo.User, opt)
			if err != nil {
				panic(err)
			}
			if len(repos) == 0 {
				break
			}
			gitearepos = append(gitearepos, repos...)
			i++
		}

		for _, r := range gitearepos {
			repos = append(repos, Repo{Name: r.Name, Url: r.CloneURL, Token: repo.Token, Defaultbranch: r.DefaultBranch})
		}
	}
	return repos
}

func getGogs(conf *Conf) []Repo {
	repos := []Repo{}
	for _, repo := range conf.Source.Gogs {
		fmt.Printf("%s: grabbing repositories from %s\n", repo.Url, repo.User)
		client := gogs.NewClient(repo.Url, repo.Token)
		gogsrepos, err := client.ListUserRepos(repo.User)
		if err != nil {
			panic(err)
		}

		for _, r := range gogsrepos {
			repos = append(repos, Repo{Name: r.Name, Url: r.CloneURL, Token: repo.Token, Defaultbranch: r.DefaultBranch})
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
		fmt.Printf("%s: grabbing repositories from %s\n", repo.Url, repo.User)
		gitlabrepos := []*gitlab.Project{}
		client, err := gitlab.NewClient(repo.Token, gitlab.WithBaseURL(repo.Url))
		if err != nil {
			panic(err)
		}
		opt := &gitlab.ListProjectsOptions{}
		users, _, err := client.Users.ListUsers(&gitlab.ListUsersOptions{Username: &repo.User})
		if err != nil {
			panic(err)
		}

		opt.PerPage = 50
		i := 0
		for _, user := range users {
			if user.Username == repo.User {
				for {
					projects, _, err := client.Projects.ListUserProjects(user.ID, opt)
					if err != nil {
						panic(err)
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
			repos = append(repos, Repo{Name: r.Name, Url: r.HTTPURLToRepo, Token: repo.Token, Defaultbranch: r.DefaultBranch})
		}
	}
	return repos
}

func main() {
	kong.Parse(&cli)
	fmt.Printf("Reading %s\n", green(cli.Configfile))
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
