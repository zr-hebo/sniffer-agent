#!/usr/bin/env bash

function execute_real(){
    mysql_host=192.168.XX.XX
    mysql_port=3358
    user_name=user
    passwd=123456

    mysql -h$mysql_host -P$mysql_port -u$user_name -p$passwd sniffer -e "select 1"
    sleep 1
    mysql -h$mysql_host -P$mysql_port -u$user_name -p$passwd sniffer -e "use sniffer;show tables;create table haha(id int, name text)"
    sleep 1
    mysql -h$mysql_host -P$mysql_port -u$user_name -p$passwd sniffer -e ""
    sleep 1
    mysql -h$mysql_host -P$mysql_port -u$user_name -p$passwd sniffer -e ""
    sleep 1
    insert_cmd="insert into unibase.haha(id, name) values(10, 'aaaa')"
    insert_cmd="$insert_cmd,(10, 'aaaa')"
    insert_cmd="$insert_cmd,(10, 'aaaa')"
    insert_cmd="$insert_cmd,(10, 'aaaa')"
    insert_cmd="$insert_cmd,(10, 'aaaa')"
    mysql -h$mysql_host -P$mysql_port -u$user_name -p$passwd sniffer -e "$insert_cmd"
    sleep 1
    mysql -h$mysql_host -P$mysql_port -u$user_name -p$passwd sniffer -e "use unibase; select * from haha; drop table haha"
    sleep 1
}

while true
    do
    execute_real
    done
