# Hookah

[![Go Report Card](https://goreportcard.com/badge/github.com/donatj/hookah)](https://goreportcard.com/report/github.com/donatj/hookah)
[![GoDoc](https://godoc.org/github.com/donatj/hookah?status.svg)](https://godoc.org/github.com/donatj/hookah)
[![Build Status](https://travis-ci.org/donatj/hookah.svg?branch=master)](https://travis-ci.org/donatj/hookah)

Hookah is a simple server for Github Webhooks that forwards the hooks messsage to any manner of script, be they PHP, Ruby, Python or even straight up shell.

It simply passes the message on to the STDIN of any script.

## Installation

### From Source:

```bash
go get -u -v github.com/donatj/hookah/cmd/hookah
```

## Example Hook Scripts

### bash + [jq](https://stedolan.github.io/jq/)

```bash
#!/bin/bash

set -e

json=`cat`
ref=$(echo "$json" | jq -r .ref)

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

$data = json_decode($input, true);
print_r($data);

```
