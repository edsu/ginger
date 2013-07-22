SHELL := /bin/bash

bowerinstall:
	npm install
	./node_modules/bower/bin/bower install; 
	pushd bower_components/bootstrap/; npm install; make; make bootstrap; popd


install:
	mkdir -p static
	cp -r bower_components/bootstrap/bootstrap static/
	cp -r bower_components/jquery static/
	mkdir -p static/js
	cp -r js static/
	cp -r bower_components/angular/angular.min.js static/js/
	cp -r bower_components/angular-bootstrap/ui-bootstrap-tpls.min.js static/js/


developjs:
	ln -sf ../../js/marvin.js static/js/marvin.js


clean:
	rm -rf static
