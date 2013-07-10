ginger
======

Ginger is a set of programs for harvesting large sets of URLs in a distributed 
fashion using Amazon Web Services. Think of ginger as Web harvesting rebooted 
for the cloud, so that you can reasonably rent the machines when you need them,
and can retire them when you don't. If you are wondering why "ginger" let's 
just say it's because @eikeon likes golang, AWS, the Web ... and ginger. We
think you will too.

Run
---

* git clone http://github.com/me/github

Components
----------

* gingerweb - a Web application and REST API
* ??? - query for and queue work
* ??? - checks the web for the resource
* ??? - persists resource metadata
* ??? - checks web archives for resource
* ??? - example importer for checking external links in Wikipedia 

Develop
-------

* fork ginger on github!
* git clone http://github.com/me/ginger
* mkdir -p $GOPATH/src/github.com/eikeon/ginger
* ln -s ginger $GOPATH/src/github.com/eikeon/ginger
* hack hack hack
* git push
* send pull request to eikeon on github
