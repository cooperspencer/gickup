<h1 align="center">
    <img src="https://github.com/cooperspencer/gickup/blob/main/gickup.png" style="width: 20%;" alt="logo">
    <br/>
    GICKUP
</h1>

<h4 align="center">
    Backup your Git repositories with ease.
</h4>


<p align="center">
    <strong>
        <a href="https://cooperspencer.github.io/gickup-documentation/" target="_blank">Website</a>
        •
        <a href="https://github.com/cooperspencer/gickup/">GitHub</a>
        •
        <a href="https://cooperspencer.github.io/gickup-documentation/" target="_blank">Docs</a>
    </strong>
</p>

<p align="center">
    <a href="https://github.com/cooperspencer/gickup/actions/workflows/docker.yml">
        <img alt="Build and Publish" src="https://github.com/cooperspencer/gickup/actions/workflows/docker.yml/badge.svg">
    </a>
</p>



## What is GICKUP?
Gickup is a tool that allows you to clone/mirror repositories from one hoster to another.
This is useful if you want to have a backup of your repositories on another hoster or to a local server.


### Supported Source and Destionations
You can clone/mirror repositories from:
- Github
- Gitlab
- Gitea
- Gogs
- Bitbucket
- OneDev
- Sourcehut
- Any

You can clone/mirror repositories to:
- Github
- Gitlab
- Gitea
- Gogs
- OneDev
- Sourcehut
- Local
- S3


If your hoster is not listed, feel free to open an issue and I will add it.



## How to make a configuration file
[Here is an example](https://github.com/cooperspencer/gickup/blob/main/conf.example.yml)

## How to run the binary version
`./gickup path-to-conf.yml`

## How to run the Docker image
```bash
mkdir gickup
wget https://raw.githubusercontent.com/cooperspencer/gickup/main/docker-compose.yml
nano conf.yml # Make your config here
docker-compose up
```
## Compile the binary version
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

## Questions?
If anything is unclear or you have a great idea for the project, feel free to open a discussion about it.
https://github.com/cooperspencer/gickup/discussions

## Distribution Packages
|Distribution|Package|Maintainer|
|---|---|---|
|Arch|[gickup](https://aur.archlinux.org/packages/gickup/)|[me](https://github.com/cooperspencer)|
|Homebrew|[gickup](https://formulae.brew.sh/formula/gickup#default)||
|Fedora|[gickup](https://copr.fedorainfracloud.org/coprs/frostyx/gickup/)|[FrostyX](https://github.com/FrostyX)|

## Issues
The mirroring to Gitlab doesn't work, or at least I can't test it properly because I have no access to a Gitlab EE instance.

## Future Ideas
- Additional VCS
  - [GitBucket](https://gitbucket.github.io/)
- Add minio as a destination
