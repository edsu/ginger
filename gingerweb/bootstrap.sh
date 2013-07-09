#!/bin/bash

pushd bower_components/bootstrap/
node ../../patch.js
npm install
make
make bootstrap
popd


