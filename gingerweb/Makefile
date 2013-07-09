all:  bootstrap jquery

clean:
	rm -r static/bootstrap
	rm -r static/jquery

bower:
	bower install

bootstrap:  bower
	mkdir -p static
	./bootstrap.sh
	cp -r bower_components/bootstrap/bootstrap static/

jquery:  bower
	mkdir -p static
	cp -r bower_components/jquery static/


