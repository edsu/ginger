#!/bin/bash

pushd components
git clone https://github.com/twitter/bootstrap.git
git checkout 3.0.0-wip
cd bootstrap
npm install
make
make bootstrap
popd


