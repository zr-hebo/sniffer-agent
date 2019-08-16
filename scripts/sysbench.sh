#!/usr/bin/env bash

sysbench /usr/share/sysbench/oltp_read_only.lua --mysql-host=192.XX.XX.XX --mysql-port=3306 --mysql-user=root --mysql-password='' --mysql-db=sniffer --db-driver=mysql --tables=10 --table-size=1000000 --report-interval=10 --threads=128 --time=120 prepare
sysbench /usr/share/sysbench/oltp_read_only.lua --mysql-host=192.XX.XX.XX --mysql-port=3306 --mysql-user=root --mysql-password='' --mysql-db=sniffer --db-driver=mysql --tables=10 --table-size=1000000 --report-interval=10 --threads=128 --time=120 run