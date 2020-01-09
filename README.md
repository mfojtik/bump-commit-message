### git-bump-commit-message

A simple Git helper that generate commit message when committing updated `go.mod` file after bumping one or more dependencies.

The helper will read `go.mod` file from `upstream/master` branch (so an `upstream` remote must exists and it must point to original repository).
Then it will read the local `go.mod` file and compare both to get list of modules that were updated.

After that, it will list commits between current version and previous version and produce commit message content that lists all changes in all
updated modules to standard output.

### Install

```
go install github.com/mfojtik/git-bump-commit-message
```

### Usage

In order to use this helpers, you have to set the `GITHUB_TOKEN` environment variable to your personal Github access token.
This is needed in order to list the commits for repositories that were updated in the bump change without being restricted by Github API request limits.

```bash
$ git checkout -b test-bump
$ go get -u github.com/openshift/library-go
$ git status
On branch test-bump
Changes to be committed:
  (use "git restore --staged <file>..." to unstage)
	modified:   go.mod
	modified:   go.sum
$ git commit -m "$(git bump-commit-message)"
```

The resulting commit message will be:

```
commit bd7824d8c4efa19c86cdf23563f2b08d05921ab6 (HEAD -> test-bump)
Author: Michal Fojtik <mfojtik@redhat.com>
Date:   Wed Jan 8 13:07:50 2020 +0100

    bump(*): vendor update

    * openshift/library-go@0b9c208d: build-machinery: Add human readable messages to go mod verify-deps
    * openshift/library-go@886b6c5c: config: default bind network to tcp instead of tcp4
```

* In case the bump is happening in branch other than `master` you can specify the base branch via `--base-branch` flag.
* In case you only want to list commits for specific paths, you can list the paths via `--paths` flag.

License
-------

This helper is licensed under the [Apache License, Version 2.0](http://www.apache.org/licenses/).
