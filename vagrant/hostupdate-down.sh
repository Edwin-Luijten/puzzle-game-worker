#!/usr/bin/env bash
if [ -f "C:/Windows/System32/drivers/etc/hosts" ]
then
	sed -i '/'$1'/d' C:/Windows/System32/drivers/etc/hosts
else
    sed -i .bak -e '/'$1'/d' /etc/hosts
fi