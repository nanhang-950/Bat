#!/bin/bash

set -euo pipefail

go build -trimpath -ldflags="-s -w" .
