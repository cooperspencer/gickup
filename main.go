package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/cooperspencer/gickup/onedev"
	"github.com/cooperspencer/gickup/s3"
	"github.com/cooperspencer/gickup/sourcehut"
	"github.com/go-git/go-git/v5"
	"github.com/google/go-cmp/cmp"

	"github.com/alecthomas/kong"
	"github.com/cooperspencer/gickup/bitbucket"
	"github.com/cooperspencer/gickup/gitea"
	"github.com/cooperspencer/gickup/github"
	"github.com/cooperspencer/gickup/gitlab"
	"github.com/cooperspencer/gickup/gogs"
	"github.com/cooperspencer/gickup/local"
	"github.com/cooperspencer/gickup/logger"
	"github.com/cooperspencer/gickup/metrics/gotify"
	"github.com/cooperspencer/gickup/metrics/heartbeat"
	"github.com/cooperspencer/gickup/metrics/ntfy"
	"github.com/cooperspencer/gickup/metrics/prometheus"
	"github.com/cooperspencer/gickup/types"
	"github.com/cooperspencer/gickup/whatever"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

var cli struct {
	Configfiles []string `arg name:"conf" help:"Path to the configfile." default:"conf.yml"`
	Version     bool     `flag name:"version" help:"Show version."`
	Dry         bool     `flag name:"dryrun" help:"Make a dry-run."`
	Debug       bool     `flag name:"debug" help:"Output debug messages"`
	Quiet       bool     `flag name:"quiet" help:"Output only warnings, errors, and fatal messages to stderr log output"`
	Silent      bool     `flag name:"silent" help:"Suppress all stderr log output"`
}

var version = "unknown"

func readConfigFile(configfile string) []*types.Conf {
	conf := []*types.Conf{}
	cfgdata, err := os.ReadFile(configfile)
	if err != nil {
		log.Fatal().
			Str("stage", "readconfig").
			Str("file", configfile).
			Msgf("Cannot open config file from %s", types.Red(configfile))
	}

	//	t := types.Conf{}

	dec := yaml.NewDecoder(bytes.NewReader(cfgdata))

	//	err = yaml.Unmarshal(cfgdata, &t)

	i := 0
	for {
		var c *types.Conf
		err = dec.Decode(&c)
		if err == io.EOF {
			break
		} else if err != nil {
			if len(conf) > 0 {
				log.Fatal().
					Str("stage", "readconfig").
					Str("file", configfile).
					Msgf("an error occured in the %d place of %s", i, configfile)
			} else {
				log.Fatal().
					Str("stage", "readconfig").
					Str("file", configfile).
					Msg(err.Error())
			}
		}

		if c == nil {
			continue
		}

		for i, local := range c.Destination.Local {
			c.Destination.Local[i].Path = substituteHomeForTildeInPath(local.Path)
		}

		if !reflect.ValueOf(c).IsZero() {
			if len(conf) > 0 {
				if len(c.Metrics.PushConfigs.Gotify) == 0 && len(c.Metrics.PushConfigs.Ntfy) == 0 {
					c.Metrics.PushConfigs = conf[0].Metrics.PushConfigs
				}
			}
			conf = append(conf, c)
			i++
		}
	}

	return conf
}

func getUserHome() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}

	return usr.HomeDir, nil
}

func substituteHomeForTildeInPath(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}

	if path == "~" {
		userHome, err := getUserHome()
		if err != nil {
			log.Fatal().
				Str("stage", "local ~ substitution").
				Str("path", path).
				Msg(err.Error())
		} else {
			return userHome
		}
	}

	if strings.HasPrefix(path, "~/") {
		userHome, err := getUserHome()
		if err != nil {
			log.Fatal().
				Str("stage", "local ~/ substitution").
				Str("path", path).
				Msg(err.Error())
		} else {
			return filepath.Join(userHome, path[2:])
		}
	}
	// in whatever other strange case
	return path
}

func backup(repos []types.Repo, conf *types.Conf) {
	checkedpath := false

	for _, r := range repos {
		log.Info().
			Str("stage", "backup").
			Msgf("starting backup for %s", r.URL)

		if conf.Destination.Count() == 0 {
			log.Warn().Str("stage", "backup").Msg("No destinations configured!")
		}

		for _, d := range conf.Destination.Local {
			if !checkedpath {
				_, err := filepath.Abs(d.Path)
				if err != nil {
					log.Fatal().
						Str("stage", "locally").
						Str("path", d.Path).
						Msg(err.Error())
				}

				checkedpath = true
			}

			repotime := time.Now()
			status := 0
			if local.Locally(r, d, cli.Dry) {
				prometheus.RepoTime.WithLabelValues(r.Hoster, r.Name, r.Owner, "local", d.Path).Set(time.Since(repotime).Seconds())
				status = 1
			}

			prometheus.RepoSuccess.WithLabelValues(r.Hoster, r.Name, r.Owner, "local", d.Path).Set(float64(status))
			prometheus.DestinationBackupsComplete.WithLabelValues("local").Inc()
		}

		for _, d := range conf.Destination.S3 {
			repotime := time.Now()
			status := 0

			log.Info().
				Str("stage", "s3").
				Str("url", d.Endpoint).
				Msgf("pushing %s to %s", types.Blue(r.Name), d.Bucket)

			if !cli.Dry {
				tempname := fmt.Sprintf("s3-%x", repotime)
				tempdir, err := os.MkdirTemp(os.TempDir(), tempname)
				if err != nil {
					log.Error().
						Str("stage", "tempclone").
						Str("url", r.URL).
						Msg(err.Error())
					continue
				}

				if d.Structured {
					r.Name = path.Join(r.Hoster, r.Owner, r.Name)
				}

				defer os.RemoveAll(tempdir)
				_, err = local.TempClone(r, path.Join(tempdir, r.Name))
				if err != nil {
					if err == git.NoErrAlreadyUpToDate {
						log.Info().
							Str("stage", "s3").
							Str("url", r.URL).
							Msg(err.Error())
					} else {
						log.Error().
							Str("stage", "tempclone").
							Str("url", r.URL).
							Str("git", "clone").
							Msg(err.Error())
						os.RemoveAll(tempdir)
						continue
					}
				}

				// Check if environment variables are used for accesskey and secretkey
				d.AccessKey, err = d.GetKey(d.AccessKey)
				if err != nil {
					log.Error().Str("stage", "s3").Str("endpoint", d.Endpoint).Str("bucket", d.Bucket).Msg(err.Error())
				}
				d.SecretKey, err = d.GetKey(d.SecretKey)
				if err != nil {
					log.Error().Str("stage", "s3").Str("endpoint", d.Endpoint).Str("bucket", d.Bucket).Msg(err.Error())
				}
				d.Token, err = d.GetKey(d.Token)
				if err != nil {
					log.Error().Str("stage", "s3").Str("endpoint", d.Endpoint).Str("bucket", d.Bucket).Msg(err.Error())
				}
				err = s3.UploadDirToS3(tempdir, d)
				if err != nil {
					log.Error().Str("stage", "s3").Str("endpoint", d.Endpoint).Str("bucket", d.Bucket).Msg(err.Error())
				}
				err = s3.DeleteObjectsNotInRepo(tempdir, r.Name, d)
				if err != nil {
					log.Error().Str("stage", "s3").Str("endpoint", d.Endpoint).Str("bucket", d.Bucket).Msg(err.Error())
				}
				prometheus.RepoTime.WithLabelValues(r.Hoster, r.Name, r.Owner, "s3", d.Endpoint).Set(time.Since(repotime).Seconds())
				status = 1

				prometheus.RepoSuccess.WithLabelValues(r.Hoster, r.Name, r.Owner, "s3", d.Endpoint).Set(float64(status))
				prometheus.DestinationBackupsComplete.WithLabelValues("s3").Inc()
			}
		}

		for _, d := range conf.Destination.Gitea {
			if d.MirrorInterval != "" {
				log.Warn().
					Str("stage", "gitea").
					Str("url", d.URL).
					Msg("mirrorinterval is deprecated and will be removed in one of the next releases. please move it under the mirror parameter.")
			}
			if !strings.HasSuffix(r.Name, ".wiki") {
				repotime := time.Now()
				status := 0
				if d.Mirror.Enabled {
					log.Info().
						Str("stage", "gitea").
						Str("url", d.URL).
						Msgf("mirroring %s to %s", types.Blue(r.Name), d.URL)

					if !cli.Dry {
						tempdir, err := os.MkdirTemp(os.TempDir(), fmt.Sprintf("gitea-%x", repotime))
						if err != nil {
							log.Error().
								Str("stage", "tempclone").
								Str("url", r.URL).
								Msg(err.Error())
							continue
						}

						defer os.RemoveAll(tempdir)
						temprepo, err := local.TempClone(r, tempdir)
						if err != nil {
							if err == git.NoErrAlreadyUpToDate {
								log.Info().
									Str("stage", "gitea").
									Str("url", r.URL).
									Msg(err.Error())
							} else {
								log.Error().
									Str("stage", "tempclone").
									Str("url", r.URL).
									Str("git", "clone").
									Msg(err.Error())
								os.RemoveAll(tempdir)
								continue
							}
						}

						cloneurl, err := gitea.GetOrCreate(d, r)
						if err != nil {
							log.Error().
								Str("stage", "gitea").
								Str("url", r.URL).
								Msg(err.Error())
							os.RemoveAll(tempdir)
							continue
						}

						err = local.CreateRemotePush(temprepo, d, cloneurl, r.Origin.LFS)
						if err != nil {
							if err == git.NoErrAlreadyUpToDate {
								log.Info().
									Str("stage", "gitea").
									Str("url", r.URL).
									Msg(err.Error())
							} else {
								log.Error().
									Str("stage", "gitea").
									Str("url", r.URL).
									Str("git", "push").
									Msg(err.Error())
								os.RemoveAll(tempdir)
								continue
							}
						}

						prometheus.RepoTime.WithLabelValues(r.Hoster, r.Name, r.Owner, "gitea", d.URL).Set(time.Since(repotime).Seconds())
						status = 1

						prometheus.RepoSuccess.WithLabelValues(r.Hoster, r.Name, r.Owner, "gitea", d.URL).Set(float64(status))
					}
				} else {
					if gitea.Backup(r, d, cli.Dry) {
						prometheus.RepoTime.WithLabelValues(r.Hoster, r.Name, r.Owner, "gitea", d.URL).Set(time.Since(repotime).Seconds())
						status = 1
					}
				}

				prometheus.RepoSuccess.WithLabelValues(r.Hoster, r.Name, r.Owner, "gitea", d.URL).Set(float64(status))
				prometheus.DestinationBackupsComplete.WithLabelValues("gitea").Inc()
			}
		}

		for _, d := range conf.Destination.Gogs {
			if !strings.HasSuffix(r.Name, ".wiki") {
				repotime := time.Now()
				status := 0
				if d.Mirror.Enabled {
					log.Info().
						Str("stage", "gogs").
						Str("url", d.URL).
						Msgf("mirroring %s to %s", types.Blue(r.Name), d.URL)

					if !cli.Dry {
						tempdir, err := os.MkdirTemp(os.TempDir(), fmt.Sprintf("gogs-%x", repotime))
						if err != nil {
							log.Error().
								Str("stage", "tempclone").
								Str("url", r.URL).
								Msg(err.Error())
							continue
						}

						defer os.RemoveAll(tempdir)
						temprepo, err := local.TempClone(r, tempdir)
						if err != nil {
							if err == git.NoErrAlreadyUpToDate {
								log.Info().
									Str("stage", "gogs").
									Str("url", r.URL).
									Msg(err.Error())
							} else {
								log.Error().
									Str("stage", "tempclone").
									Str("url", r.URL).
									Str("git", "clone").
									Msg(err.Error())
								os.RemoveAll(tempdir)
								continue
							}
						}

						cloneurl, err := gogs.GetOrCreate(d, r)
						if err != nil {
							log.Error().
								Str("stage", "gogs").
								Str("url", r.URL).
								Msg(err.Error())
							os.RemoveAll(tempdir)
							continue
						}

						err = local.CreateRemotePush(temprepo, d, cloneurl, r.Origin.LFS)
						if err != nil {
							if err == git.NoErrAlreadyUpToDate {
								log.Info().
									Str("stage", "gogs").
									Str("url", r.URL).
									Msg(err.Error())
							} else {
								log.Error().
									Str("stage", "gogs").
									Str("url", r.URL).
									Str("git", "push").
									Msg(err.Error())
								os.RemoveAll(tempdir)
								continue
							}
						}

						prometheus.RepoTime.WithLabelValues(r.Hoster, r.Name, r.Owner, "gogs", d.URL).Set(time.Since(repotime).Seconds())
						status = 1

						prometheus.RepoSuccess.WithLabelValues(r.Hoster, r.Name, r.Owner, "gogs", d.URL).Set(float64(status))
					}
				} else {
					if gogs.Backup(r, d, cli.Dry) {
						prometheus.RepoTime.WithLabelValues(r.Hoster, r.Name, r.Owner, "gogs", d.URL).Set(time.Since(repotime).Seconds())
						status = 1
					}
				}

				prometheus.RepoSuccess.WithLabelValues(r.Hoster, r.Name, r.Owner, "gogs", d.URL).Set(float64(status))
				prometheus.DestinationBackupsComplete.WithLabelValues("gogs").Inc()
			}
		}

		for _, d := range conf.Destination.Gitlab {
			if !strings.HasSuffix(r.Name, ".wiki") {
				if d.URL == "" {
					d.URL = "https://gitlab.com"
				}

				repotime := time.Now()
				status := 0
				if d.Mirror.Enabled {
					log.Info().
						Str("stage", "gitlab").
						Str("url", d.URL).
						Msgf("mirroring %s to %s", types.Blue(r.Name), d.URL)

					if !cli.Dry {
						tempdir, err := os.MkdirTemp(os.TempDir(), fmt.Sprintf("gitlab-%x", repotime))
						if err != nil {
							log.Error().
								Str("stage", "tempclone").
								Str("url", r.URL).
								Msg(err.Error())
							continue
						}

						defer os.RemoveAll(tempdir)
						temprepo, err := local.TempClone(r, tempdir)
						if err != nil {
							if err == git.NoErrAlreadyUpToDate {
								log.Info().
									Str("stage", "gitlab").
									Str("url", r.URL).
									Msg(err.Error())
							} else {
								log.Error().
									Str("stage", "tempclone").
									Str("url", r.URL).
									Str("git", "clone").
									Msg(err.Error())
								os.RemoveAll(tempdir)
								continue
							}
						}

						cloneurl, err := gitlab.GetOrCreate(d, r)
						if err != nil {
							log.Error().
								Str("stage", "gitlab").
								Str("url", r.URL).
								Msg(err.Error())
							os.RemoveAll(tempdir)
							continue
						}

						err = local.CreateRemotePush(temprepo, d, cloneurl, r.Origin.LFS)
						if err != nil {
							if err == git.NoErrAlreadyUpToDate {
								log.Info().
									Str("stage", "gitlab").
									Str("url", r.URL).
									Msg(err.Error())
							} else {
								log.Error().
									Str("stage", "gitlab").
									Str("url", r.URL).
									Str("git", "push").
									Msg(err.Error())
								os.RemoveAll(tempdir)
								continue
							}
						}

						prometheus.RepoTime.WithLabelValues(r.Hoster, r.Name, r.Owner, "gitlab", d.URL).Set(time.Since(repotime).Seconds())
						status = 1

						prometheus.RepoSuccess.WithLabelValues(r.Hoster, r.Name, r.Owner, "gitlab", d.URL).Set(float64(status))
					}
				} else {
					if gitlab.Backup(r, d, cli.Dry) {
						prometheus.RepoTime.WithLabelValues(r.Hoster, r.Name, r.Owner, "gitlab", d.URL).Set(time.Since(repotime).Seconds())
						status = 1
					}
				}

				prometheus.RepoSuccess.WithLabelValues(r.Hoster, r.Name, r.Owner, "gitlab", d.URL).Set(float64(status))
				prometheus.DestinationBackupsComplete.WithLabelValues("gitlab").Inc()
			}
		}

		for _, d := range conf.Destination.Github {
			if !strings.HasSuffix(r.Name, ".wiki") {
				repotime := time.Now()
				status := 0

				log.Info().
					Str("stage", "github").
					Str("url", "https://github.com").
					Msgf("mirroring %s to %s", types.Blue(r.Name), "https://github.com")

				if !cli.Dry {
					tempdir, err := os.MkdirTemp(os.TempDir(), fmt.Sprintf("github-%x", repotime))
					if err != nil {
						log.Error().
							Str("stage", "tempclone").
							Str("url", r.URL).
							Msg(err.Error())
						continue
					}

					defer os.RemoveAll(tempdir)
					temprepo, err := local.TempClone(r, tempdir)
					if err != nil {
						if err == git.NoErrAlreadyUpToDate {
							log.Info().
								Str("stage", "github").
								Str("url", r.URL).
								Msg(err.Error())
						} else {
							log.Error().
								Str("stage", "tempclone").
								Str("url", r.URL).
								Str("git", "clone").
								Msg(err.Error())
							os.RemoveAll(tempdir)
							continue
						}
					}

					cloneurl, err := github.GetOrCreate(d, r)
					if err != nil {
						log.Error().
							Str("stage", "github").
							Str("url", r.URL).
							Msg(err.Error())
						os.RemoveAll(tempdir)
						continue
					}

					err = local.CreateRemotePush(temprepo, d, cloneurl, r.Origin.LFS)
					if err != nil {
						if err == git.NoErrAlreadyUpToDate {
							log.Info().
								Str("stage", "github").
								Str("url", r.URL).
								Msg(err.Error())
						} else {
							log.Error().
								Str("stage", "github").
								Str("url", r.URL).
								Str("git", "push").
								Msg(err.Error())
							os.RemoveAll(tempdir)
							continue
						}
					}

					prometheus.RepoTime.WithLabelValues(r.Hoster, r.Name, r.Owner, "github", "https://github.com").Set(time.Since(repotime).Seconds())
					status = 1

					prometheus.RepoSuccess.WithLabelValues(r.Hoster, r.Name, r.Owner, "github", "https://github.com").Set(float64(status))
					prometheus.DestinationBackupsComplete.WithLabelValues("github").Inc()
				}
			}
		}

		for _, d := range conf.Destination.OneDev {
			if !strings.HasSuffix(r.Name, ".wiki") {
				repotime := time.Now()
				status := 0
				if d.URL == "" {
					d.URL = "https://code.onedev.io/"
				}

				log.Info().
					Str("stage", "onedev").
					Str("url", d.URL).
					Msgf("mirroring %s to %s", types.Blue(r.Name), d.URL)

				if !cli.Dry {
					tempdir, err := os.MkdirTemp(os.TempDir(), fmt.Sprintf("onedev-%x", repotime))
					if err != nil {
						log.Error().
							Str("stage", "tempclone").
							Str("url", r.URL).
							Msg(err.Error())
						continue
					}

					defer os.RemoveAll(tempdir)
					temprepo, err := local.TempClone(r, tempdir)
					if err != nil {
						if err == git.NoErrAlreadyUpToDate {
							log.Info().
								Str("stage", "onedev").
								Str("url", r.URL).
								Msg(err.Error())
						} else {
							log.Error().
								Str("stage", "tempclone").
								Str("url", r.URL).
								Msg(err.Error())
							os.RemoveAll(tempdir)
							continue
						}
					}

					cloneurl, err := onedev.GetOrCreate(d, r)
					if err != nil {
						log.Error().
							Str("stage", "onedev").
							Str("url", r.URL).
							Msg(err.Error())
						os.RemoveAll(tempdir)
						continue
					}

					err = local.CreateRemotePush(temprepo, d, cloneurl, r.Origin.LFS)
					if err != nil {
						if err == git.NoErrAlreadyUpToDate {
							log.Info().
								Str("stage", "onedev").
								Str("url", r.URL).
								Msg(err.Error())
						} else {
							log.Error().
								Str("stage", "onedev").
								Str("url", r.URL).
								Msg(err.Error())
							os.RemoveAll(tempdir)
							continue
						}
					}

					prometheus.RepoTime.WithLabelValues(r.Hoster, r.Name, r.Owner, "onedev", d.URL).Set(time.Since(repotime).Seconds())
					status = 1

					prometheus.RepoSuccess.WithLabelValues(r.Hoster, r.Name, r.Owner, "onedev", d.URL).Set(float64(status))
					prometheus.DestinationBackupsComplete.WithLabelValues("onedev").Inc()
					os.RemoveAll(tempdir)
				}
			}
		}

		for _, d := range conf.Destination.Sourcehut {
			if !strings.HasSuffix(r.Name, "-docs") {
				repotime := time.Now()
				status := 0
				d.SSH = true
				if d.URL == "" {
					d.URL = "https://git.sr.ht"
				}

				log.Info().
					Str("stage", "sourcehut").
					Str("url", d.URL).
					Msgf("mirroring %s to %s", types.Blue(r.Name), d.URL)

				if !cli.Dry {
					tempdir, err := os.MkdirTemp(os.TempDir(), fmt.Sprintf("sourcehut-%x", repotime))
					if err != nil {
						log.Error().
							Str("stage", "tempclone").
							Str("url", r.URL).
							Msg(err.Error())
						continue
					}

					defer os.RemoveAll(tempdir)
					temprepo, err := local.TempClone(r, tempdir)
					if err != nil {
						if err == git.NoErrAlreadyUpToDate {
							log.Info().
								Str("stage", "sourcehut").
								Str("url", r.URL).
								Msg(err.Error())
						} else {
							log.Error().
								Str("stage", "tempclone").
								Str("url", r.URL).
								Msg(err.Error())
							os.RemoveAll(tempdir)
							continue
						}
					}

					cloneurl, err := sourcehut.GetOrCreate(d, r)
					if err != nil {
						log.Error().
							Str("stage", "sourcehut").
							Str("url", r.URL).
							Msg(err.Error())
						os.RemoveAll(tempdir)
						continue
					}

					err = local.CreateRemotePush(temprepo, d, cloneurl, r.Origin.LFS)
					if err != nil {
						if err == git.NoErrAlreadyUpToDate {
							log.Info().
								Str("stage", "sourcehut").
								Str("url", r.URL).
								Msg(err.Error())
						} else {
							log.Error().
								Str("stage", "sourcehut").
								Str("url", r.URL).
								Msg(err.Error())
							os.RemoveAll(tempdir)
							continue
						}
					}

					prometheus.RepoTime.WithLabelValues(r.Hoster, r.Name, r.Owner, "sourcehut", d.URL).Set(time.Since(repotime).Seconds())
					status = 1

					prometheus.RepoSuccess.WithLabelValues(r.Hoster, r.Name, r.Owner, "sourcehut", d.URL).Set(float64(status))
					prometheus.DestinationBackupsComplete.WithLabelValues("sourcehut").Inc()
					os.RemoveAll(tempdir)
				}
			}
		}

		prometheus.SourceBackupsComplete.WithLabelValues(r.Name).Inc()
	}
}

func runBackup(conf *types.Conf, num int) {
	log.Info().Msg("Backup run starting")

	numstring := strconv.Itoa(num)

	startTime := time.Now()

	prometheus.JobsStarted.Inc()

	// Github
	repos, ran := github.Get(conf)
	if ran {
		prometheus.CountReposDiscovered.WithLabelValues("github", numstring).Set(float64(len(repos)))
	}
	backup(repos, conf)

	// Gitea
	repos, ran = gitea.Get(conf)
	if ran {
		prometheus.CountReposDiscovered.WithLabelValues("gitea", numstring).Set(float64(len(repos)))
	}
	backup(repos, conf)

	// Gogs
	repos, ran = gogs.Get(conf)
	if ran {
		prometheus.CountReposDiscovered.WithLabelValues("gogs", numstring).Set(float64(len(repos)))
	}
	backup(repos, conf)

	// Gitlab
	repos, ran = gitlab.Get(conf)
	if ran {
		prometheus.CountReposDiscovered.WithLabelValues("gitlab", numstring).Set(float64(len(repos)))
	}
	backup(repos, conf)

	repos, ran = bitbucket.Get(conf)
	if ran {
		prometheus.CountReposDiscovered.WithLabelValues("bitbucket", numstring).Set(float64(len(repos)))
	}
	backup(repos, conf)

	repos, ran = whatever.Get(conf)
	if ran {
		prometheus.CountReposDiscovered.WithLabelValues("whatever", numstring).Set(float64(len(repos)))
	}
	backup(repos, conf)

	repos, ran = onedev.Get(conf)
	if ran {
		prometheus.CountReposDiscovered.WithLabelValues("onedev", numstring).Set(float64(len(repos)))
	}
	backup(repos, conf)

	repos, ran = sourcehut.Get(conf)
	if ran {
		prometheus.CountReposDiscovered.WithLabelValues("sourcehut", numstring).Set(float64(len(repos)))
	}
	backup(repos, conf)

	endTime := time.Now()
	duration := endTime.Sub(startTime)

	prometheus.JobsComplete.Inc()
	prometheus.JobDuration.Observe(duration.Seconds())

	if len(conf.Metrics.Heartbeat.URLs) > 0 {
		heartbeat.Send(conf.Metrics.Heartbeat)
	}

	if len(conf.Metrics.PushConfigs.Ntfy) > 0 {
		for _, pusher := range conf.Metrics.PushConfigs.Ntfy {
			pusher.ResolveToken()
			err := ntfy.Notify(fmt.Sprintf("backup took %v", duration), *pusher)
			if err != nil {
				log.Warn().Str("push", "ntfy").Err(err).Msg("couldn't send message")
			}
		}
	}

	if len(conf.Metrics.PushConfigs.Gotify) > 0 {
		for _, pusher := range conf.Metrics.PushConfigs.Gotify {
			pusher.ResolveToken()
			err := gotify.Notify(fmt.Sprintf("backup took %v", duration), *pusher)
			if err != nil {
				log.Warn().Str("push", "gotify").Err(err).Msg("couldn't send message")
			}
		}
	}

	log.Info().
		Str("duration", duration.String()).
		Msg("Backup run complete")

	if conf.HasValidCronSpec() {
		logNextRun(conf)
	}
}

func playsForever(c *cron.Cron, conffiles []string, confs []*types.Conf) bool {
	for {
		checkconfigs := []*types.Conf{}
		for _, f := range conffiles {
			checkconfigs = append(checkconfigs, readConfigFile(f)...)
		}

		if checkconfigs[0].HasValidCronSpec() {
			for num, config := range checkconfigs {
				if !config.HasValidCronSpec() {
					checkconfigs[num].Cron = checkconfigs[0].Cron
				}
			}
		}

		if !cmp.Equal(confs, checkconfigs) {
			log.Info().Msg("config changed")
			log.Debug().Msg(cmp.Diff(confs, checkconfigs))
			for _, entry := range c.Entries() {
				c.Remove(entry.ID)
			}
			return true
		}

		time.Sleep(5 * time.Second)
	}
}

func main() {
	timeformat := "2006-01-02T15:04:05Z07:00"

	if len(os.Getenv("GICKUP_TIME_FORMAT")) > 0 {
		timeformat = os.Getenv("GICKUP_TIME_FORMAT")
	}

	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: timeformat,
	})

	kong.Parse(&cli, kong.Name("gickup"),
		kong.Description("a tool to backup all your favorite repos"))

	if cli.Version {
		fmt.Println(version)

		return
	}

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if cli.Quiet {
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	}

	if cli.Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	if cli.Silent {
		zerolog.SetGlobalLevel(zerolog.Disabled)
	}

	if cli.Dry {
		log.Info().
			Str("dry", "true").
			Msgf("this is a %s", types.Blue("dry run"))
	}

	init := true
	for {
		reload := false
		confs := []*types.Conf{}
		for i, f := range cli.Configfiles {
			log.Info().Str("file", f).
				Msgf("Reading %s", types.Green(f))
			absf, err := filepath.Abs(f)
			if err != nil {
				log.Panic().Err(err).Msgf("there is an issue with %s", f)
			}
			cli.Configfiles[i] = absf
			confs = append(confs, readConfigFile(absf)...)
		}

		logConf := confs[0].Log

		if logConf.Timeformat == "" {
			logConf.Timeformat = timeformat
		}

		log.Logger = logger.CreateLogger(logConf)

		validcron := confs[0].HasValidCronSpec()

		var c *cron.Cron

		if validcron {
			c = cron.New()
			c.Start()
		}

		sourcecount := 0
		destinationcount := 0
		// one pair per source-destination
		for num, conf := range confs {
			pairs := conf.Source.Count() * conf.Destination.Count()
			sourcecount += conf.Source.Count()
			destinationcount += conf.Destination.Count()
			log.Info().
				Int("sources", conf.Source.Count()).
				Int("destinations", conf.Destination.Count()).
				Int("pairs", pairs).
				Msg("Configuration loaded")

			if !conf.HasValidCronSpec() {
				conf.Cron = confs[0].Cron
			}

			if conf.HasValidCronSpec() && validcron {
				conf := conf // https://stackoverflow.com/questions/57095167/how-do-i-create-multiple-cron-function-by-looping-through-a-list
				num := num

				logNextRun(conf)

				_, err := c.AddFunc(conf.Cron, func() {
					runBackup(conf, num)
				})
				if err != nil {
					log.Fatal().
						Int("sources", conf.Source.Count()).
						Int("destinations", conf.Destination.Count()).
						Int("pairs", pairs).
						Msg(err.Error())
				}
			} else {
				runBackup(conf, num)
			}
		}

		if validcron {
			if confs[0].HasAllPrometheusConf() {
				prometheus.CountSourcesConfigured.Add(float64(sourcecount))
				prometheus.CountDestinationsConfigured.Add(float64(destinationcount))
				if init {
					go prometheus.Serve(confs[0].Metrics.Prometheus)
					init = false
				}
			}
			reload = playsForever(c, cli.Configfiles, confs)
			log.Info().Msg("reloading config...")
		}
		if !reload {
			break
		}
	}
	os.Exit(int(logger.GetExitCode()))
}

func logNextRun(conf *types.Conf) {
	nextRun, err := conf.GetNextRun()
	if err == nil {
		log.Info().
			Str("next", nextRun.String()).
			Str("cron", conf.Cron).
			Msg("Next cron run")
	}
}
