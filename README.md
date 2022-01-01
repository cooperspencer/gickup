[![Build and Publish](https://github.com/cooperspencer/gickup/actions/workflows/docker.yml/badge.svg)](https://github.com/cooperspencer/gickup/actions/workflows/docker.yml)
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

## How to make a Configuration file?
[Here is an example](https://github.com/cooperspencer/gickup/blob/main/conf.example.yml)

## How to run the Binary version
`./gickup path-to-conf.yml`

## How to run the Docker image
```bash
mkdir gickup
wget https://raw.githubusercontent.com/cooperspencer/gickup/main/docker-compose.yml
nano conf.yml # Make your config here
docker-compose up
```
## Compile the Binary version
`go build .`

## Compile the Docker Image
```bash
git clone https://github.com/cooperspencer/gickup.git
cd gickup
nano docker-compose.yml # Uncomment the Build
nano conf.yml # Make your config here
docker-compose build
docker-compose up
```

## Distribution Packages
|Distribution|Package|Maintainer|
|---|---|---|
|Arch|[gickup](https://aur.archlinux.org/packages/gickup/)|[zhulik](https://github.com/zhulik)|

## Issues
The mirroring to Gitlab doesn't work, or at least I can't test it properly because I have no access to a Gitlab EE instance.
