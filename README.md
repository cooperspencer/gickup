# GICKUP
You can clone/mirror repositories from:
- Github
- Gitlab
- Gitea
- Gogs

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
  gitea:
    - token: blabla
      user: blabla
      url: bla.bla.com
  gogs:
    - token: blabla
      user: blabla
      url: bla.bla.com
  gitlab:
    - token: blabla
      user: blabla
      url: bla.bla.com
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