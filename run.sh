#!/bin/bash

export WORKON_HOME=/home/app/.virtualenvs
export VIRTUALENVWRAPPER_PYTHON=python3
source /usr/local/bin/virtualenvwrapper.sh
workon main

set -e

cd /home/app


export PATH="/usr/local/go/bin:$PATH"
nice -n 19 python3 run.py
