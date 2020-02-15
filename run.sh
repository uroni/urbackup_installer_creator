#!/bin/bash

export WORKON_HOME=/home/app/.virtualenvs
export VIRTUALENVWRAPPER_PYTHON=python3
source /usr/local/bin/virtualenvwrapper.sh
workon main

set -e

cd /home/app

python3 run.py