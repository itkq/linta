# linta

GitHub Actions' permissions linter.

## Usage

```
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
$ go run . ./testdata/1.yml
excessive permission (job:1): action:write
excessive permission (job:1): contents:write
insufficient permission (job:1): id-token:write (aws-actions/configure-aws-credentials)
insufficient permission (job:1): pull-requests:write (actions/labeler)
exit status 1
```

## How it works

Just compare the permissions by knowledge-based config like:

```yaml
- repository: "actions/checkout"
  permissions:
    contents: "read"
```

There's no magic!

## TODO
- Support custom configuration
- Support flexible inputs
- Support workflow-level permissions
- Output problem position
