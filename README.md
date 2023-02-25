# linta

GitHub Actions' permissions linter.

## Usage

```
$ ./linta init testdata/*
Created .linta.yml
$ cat .linta.yml
repositories:
    actions/checkout:
        contents: read
    actions/labeler:
        contents: read
        pull-requests: write
    actions/setup-node: {}
    aws-actions/configure-aws-credentials:
        id-token: write
$ cat testdata/1.yml
name: 'basic'

on:
  push:

jobs:
  1:
    runs-on: ubuntu-latest
    permissions:
      action: write
      contents: write
      pull-requests: read
    steps:
      - uses: actions/checkout@v3
      - uses: aws-actions/configure-aws-credentials@v1
      - uses: actions/labeler@v3
$ ./linta run testdata/1.yml
./testdata/1.yml:10:15 job 1 has excessive permission: action:write
./testdata/1.yml:11:17 job 1 has excessive permission: contents:write
./testdata/1.yml:16:15 job 1 has insufficient permission: pull-requests:write (required by actions/labeler)
./testdata/1.yml:15:15 job 1 has insufficient permission: id-token:write (required by aws-actions/configure-aws-credentials)
$ echo $?
1
```

## How it works

Just compare the permissions by knowledge-based config like:

```yaml
repositories:
  "actions/checkout":
    contents: "read"
```

There's no magic!

## TODO
- Support workflow-level permissions
