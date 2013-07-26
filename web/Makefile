SHELL := /bin/bash

install: bowerinstall
	mkdir -p static
	cp -r bower_components/bootstrap/bootstrap static/
	cp -r bower_components/jquery static/
	mkdir -p static/js
	cp -r js static/
	cp -r bower_components/angular/angular.min.js static/js/
	cp -r bower_components/angular-bootstrap/ui-bootstrap-tpls.min.js static/js/

bowerinstall:
	npm install
	./node_modules/bower/bin/bower install; 
	pushd bower_components/bootstrap/; npm install; make; make bootstrap; popd

developjs:
	ln -sf ../../js/marvin.js static/js/marvin.js


clean:
	rm -rf static
	rm -rf bower_components
	rm -rf node_modules
