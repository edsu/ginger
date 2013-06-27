ginger
======

Ginger is a set of programs for checking large sets of URLs in a distributed fashion using Amazon Web Services.
Think of it as web harvesting rebooted for the cloud. If you are wondering why "ginger" it's because @eikeon
likes golang and AWS, and ginger -- and we hope you will too.

Components
----------

* web - a Web application and REST API
* aaa - query for and queue work
* bbb - checks the web for the resource
* ccc - persists resource metadata
* ddd - checks web archives for resource
* eee - example importer for checking external links in Wikipedia 
