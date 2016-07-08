# Hookah

[![Go Report Card](https://goreportcard.com/badge/github.com/donatj/hookah)](https://goreportcard.com/report/github.com/donatj/hookah)
[![GoDoc](https://godoc.org/github.com/donatj/hookah?status.svg)](https://godoc.org/github.com/donatj/hookah)

Hookah is a simple server for Github Webhooks that forwards the hooks messsage to any manner of script, be they PHP, Ruby, Python or even straight up shell. 

It simply passes the message on to the STDIN of any script.

### Simple Example Script

```php
#!/usr/bin/php
<?php

$input = file_get_contents("php://stdin");

$data = json_decode($input, true);
print_r($data);

```
