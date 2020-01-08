### bump-commit-message

A simple Git helper that generate commit message when committing updated `go.mod` file after bumping one or more dependencies.

The helper will read `go.mod` file from `upstream/master` branch (so an `upstream` remote must exists and it must point to original repository).
Then it will read the local `go.mod` file and compare both to get list of modules that were updated.

After that, it will list commits between current version and previous version and produce commit message content that lists all changes in all
updated modules to standard output.

### Usage

```bash
$ git checkout -b test-bump
$ go get -u github.com/openshift/library-go
$ git status
On branch test-bump
Changes to be committed:
  (use "git restore --staged <file>..." to unstage)
	modified:   go.mod
	modified:   go.sum
$ git commit -m "$(bump-commit-message)"
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