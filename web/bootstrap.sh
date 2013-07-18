#!/bin/bash

# this is called from the Makefile

pushd bower_components/bootstrap/

read -d '' PATCH <<-"EOF"
fs = require('fs');
var data = JSON.parse(fs.readFileSync("package.json"));
data["devDependencies"]["recess"] = "1.1.8";
fs.writeFileSync("package.json", JSON.stringify(data, null, 2));
EOF
node -e "${PATCH}"

npm install --dev
make
make bootstrap
popd


