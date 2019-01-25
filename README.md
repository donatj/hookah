# Hookah

[![Join the chat at https://gitter.im/hookah-server/Lobby](https://badges.gitter.im/hookah-server/Lobby.svg)](https://gitter.im/hookah-server/Lobby?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)

[![Go Report Card](https://goreportcard.com/badge/github.com/donatj/hookah)](https://goreportcard.com/report/github.com/donatj/hookah)
[![GoDoc](https://godoc.org/github.com/donatj/hookah?status.svg)](https://godoc.org/github.com/donatj/hookah)
[![Build Status](https://travis-ci.org/donatj/hookah.svg?branch=master)](https://travis-ci.org/donatj/hookah)

Hookah is a simple server for GitHub Webhooks that forwards the hooks messsage to any manner of script, be they PHP, Ruby, Python or even straight up shell.

It simply passes the message on to the STDIN of any script.

## Installation

### From Source:

```bash
go get -u -v github.com/donatj/hookah/cmd/hookah
```

## Usage

When receiving a webhook request from GitHub, Hookah checks `{server-root}/{vendor}/{product}/{X-GitHub-Event}/*` for any ***executable*** scripts, and executes them sequentially passing the JSON payload to it's standard in.

This allows actual hook scripts to be written in any language you prefer.

For example, a script `server/donatj/hookah/push/log.rb` would be executed every time a "push" event webhook was received from GitHub on the donatj/hookah repo.

## Example Hook Scripts

### bash + [jq](https://stedolan.github.io/jq/)

```bash
#!/bin/bash

set -e

json=`cat`
ref=$(<<< "$json" jq -r .ref)

echo "$ref"
if [ "$ref" == "refs/heads/master" ]
then
        echo "Ref was Master"
else
        echo "Ref was not Master"
fi

```

### PHP

```php
#!/usr/bin/php
<?php

$input = file_get_contents("php://stdin");
$data  = json_decode($input, true);

print_r($data);

```

### Note:

Don't forget to make your scripts executable (`chmod +x <script filename>`), and add a [shebang](https://en.m.wikipedia.org/wiki/Shebang_(Unix)) poiting to your desired interpreter (i.e. `#!/bin/bash`) as the first line.
