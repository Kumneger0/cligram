#!/bin/sh
set -e
rm -rf manpages
mkdir manpages
go run . man | gzip -c -9 >manpages/cligram.1.gz