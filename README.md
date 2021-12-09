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

## How to make an Configuration file?
[Here is an example](https://github.com/cooperspencer/gickup/blob/main/config.example.yml)

## How to run the Binary version
`./gickup path-to-config.yml`

## How to run the Docker image
```bash
mkdir gickup
wget https://raw.githubusercontent.com/cooperspencer/gickup/main/docker-compose.yml
nano config.yml # Make here your config
docker-compose up
```
## Compile the Binary version
`go build .`

## Compile the Docker Image
```bash
git clone https://github.com/cooperspencer/gickup.git
cd gickup
nano docker-compose.yml # Uncomment the Build
nano config.yml # Make here your config
docker-compose build
docker-compose up
```