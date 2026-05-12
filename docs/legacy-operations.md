# Legacy Operations Notes

This file was moved from the old root `README.md`. It is retained as historical
reference only and is not an active production or local-development runbook.
The current production entry point is
[`docs/production-runbook.md`](production-runbook.md). The current local
workflow is documented in the root [`README.md`](../README.md),
[`docs/local-docker-runtime.md`](local-docker-runtime.md), and
[`docs/database-baseline.md`](database-baseline.md).

Do not copy credentials, service units, database commands, paths, or dependency
versions from this file into active deployments without revalidating them
against the current runbooks.

## Original Notes

Chapter 1: Start NATS Server Automatically

1.1 Install the server
go get github.com/nats-io/gnatsd

1.2 Create the service file in /etc/systemd/system/nats-server.service
>>>>>>>>>>>
[Unit]
Description=nats-server service
After=network-online.target
Before=aofei.service

[Service]
Type=simple
ExecStart=/home/winter/go/bin/nats-server

[Install]
WantedBy=multi-user.target aofei.service
<<<<<<<<<<<

Start it: sudo systemctl start gnatsd
Get it to run at boot: sudo systemctl enable gnatsd
stop it: sudo systemctl stop gnatsd
disable it: sudo systemctl disable gnatsd

1.3 Create the httpd file in /etc/systemd/system/aofei.service
>>>>>>>>>>>
[Unit]
Description=Aofei DSP service

[Service]
Type=simple
Environment=SUMMER=/home/winter/aofei/etc/summer.local.json AOFEI=/home/winter/aofei/etc/aofei.local.json
ExecStart=/home/winter/aofei/cmd/unify/unify -log_dir=/home/winter/logs

[Install]
WantedBy=multi-user.target
<<<<<<<<<<<

Start it: sudo systemctl start web
Get it to run at boot: sudo systemctl enable web
stop it: sudo systemctl stop web
disable it: sudo systemctl disable web



Chapter 2: Install redis

Follow-up here: https://www.digitalocean.com/community/tutorials/how-to-install-and-configure-redis-on-ubuntu-16-04

After that: systemctl enable redis
to have it run at reboot.


Chapter 3: Configuration

3.1) pzutil/config historical examples are in backup/, product e.g. etc/aofei.local.json
3.2) genet/config historical examples are in backup/, product e.g. etc/summer.local.json


Chapter 4: Run Test with the "gotest" database

4.1) "go test" the follow packages:
demo dmp genelet ipsearch match pzutil uadevice
Make sure in each directory, so the config file is correctly located

4.2) initialize the database in conf
mysql -ueightran -p12pass34 gotest  < summer.sql
Add any SQL/VIEW/TBL with series > 100...

4.3) loading testing data in src/summer
mysql -ueightran -p12pass34 gotest  < sample.sql
mysql -ueightran -p12pass34 gotest  < more.sql
and then
go test summer

4.4) fill-in weights and run weight test in src/summer/weight
go test summer/weight

4.5) go to ssp, make sure using DBGetNWeights in redis.go and make sure size_id
= "1", ..."10" are defined in aofei.json for PSA. Run the test:
go test ssp
This should work. In production, we may use DbGetNWhites


Chapter 5: Build a real database
5.1) start with summer.sql
5.2) start with any additional table/proc/view/trigger/sql/go from >= 100....


Chapter 6: How to set up Condition_uri?

Chapeter 7: From Bitbucket to Github
$ cd $HOME/Code/repo-directory
$ git remote rename origin bitbucket
$ git remote add origin https://github.com/mandiwise/awesome-new-repo.git
$ git push origin master

$ git remote rm bitbucket

Chapeter 8: taosd

manual install
after "make install":
Directory/File	Description
/etc/taos/taos.cfg	TDengine configuration file
/usr/local/taos/driver	TDengine dynamic link library
/var/lib/taos	TDengine default data directory
/var/log/taos	TDengine default log directory
/usr/local/taos/bin.	TDengine executables

After installation. These lines pop up:
To configure TDengine : edit /etc/taos/taos.cfg
To start TDengine     : sudo systemctl start taosd
To access TDengine    : use taos in shell
