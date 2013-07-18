#!/bin/bash

# this is called from the Makefile

pushd bower_components/bootstrap/
npm install
make
make bootstrap
popd


