# GICKUP
You can clone/mirror repositories from:
- Github
- Gitlab
- Gitea
- Gogs
- Bitbucket

You can clone/mirror them to:
- Gitlab
- Gitea
- Gogs
- Local

## Example Config
```yaml
source:
  github:
    - token: blabla
      user: blabla
      url: bla.bla.com
      username: bla
      password: bla
      ssh: true # can be true or false
      sshkey: /path/to/key # if empty, it uses your home directories' .ssh/id_rsa
      exclude: # this excludes the repos foo and bar
        - foo
        - bar
      excludeorgs:
        - foo
        - bar
  gitea:
    - token: blabla
      user: blabla
      url: bla.bla.com
      username: bla
      password: bla
      ssh: true # can be true or false
      sshkey: /path/to/key # if empty, it uses your home directories' .ssh/id_rsa
      exclude: # this excludes the repos foo and bar
        - foo
        - bar
  gogs:
    - token: blabla
      user: blabla
      url: bla.bla.com
      username: bla
      password: bla
      ssh: true # can be true or false
      sshkey: /path/to/key # if empty, it uses your home directories' .ssh/id_rsa
      exclude: # this excludes the repos foo and bar
        - foo
        - bar
  gitlab:
    - token: blabla
      user: blabla
      url: bla.bla.com
      username: bla
      password: bla
      ssh: true # can be true or false
      sshkey: /path/to/key # if empty, it uses your home directories' .ssh/id_rsa
      exclude: # this excludes the repos foo and bar
        - foo
        - bar
  bitbucket:
    - user: blabla
      url: blabla
      username: blabla
      password: blabla
      ssh: true # can be true or false
      sshkey: /path/to/key # if empty, it uses your home directories' .ssh/id_rsa
      exclude: # this excludes the repos foo and bar
        - foo
        - bar
destination:
  gitea:
    - token: blabla
      url: bla.bla.com
  gogs:
    - token: blabla
      url: bla.bla.com
  gitlab:
    - token: blabla
      url: bla.bla.com
  local:
    - path: /some/path/gickup
```

## How to run
`./gickup path-to-config.yml`

## Compile
`go build .`