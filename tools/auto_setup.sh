#!/bin/bash

openBrowser(){
	sleep 3
	xdg-open "http://127.0.0.1:8080/vfs"
}

./oneshow su you

isLogin=$(./oneshow info|grep ower)

if [ -z "$isLogin" ]
then
	echo "init you"
	./oneshow auth
	./oneshow saveUser you

	echo "start you"
	openBrowser &
	./oneshow  web -u 127.0.0.1:8080
else
	echo "start you"
	openBrowser &
	./oneshow  web -u 127.0.0.1:8080
fi


