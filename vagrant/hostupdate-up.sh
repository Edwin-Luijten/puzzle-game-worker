#!/usr/bin/env bash
if [ -f "C:/Windows/System32/drivers/etc/hosts" ]
then
	echo "$1  $2 $3 $4 $5 $6 $7 $8 $9" >> C:/Windows/System32/drivers/etc/hosts
else
	echo "$1  $2 $3 $4 $5 $6 $7 $8 $9" >> /etc/hosts
fi