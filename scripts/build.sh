#!/bin/bash

go build -o gourl
zip -r /tmp/gourl.zip gourl templates/
