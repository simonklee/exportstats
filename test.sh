#!/bin/bash

#curl -i "http://localhost:6070/v1/exportstats/rate/eu.website.load.webgl.launch.cnt/eu.website.load.webgl.download.cnt?t=1h3m&format=json" || exit 1
#curl -i "http://localhost:6070/v1/exportstats/rate/eu.website.load.webgl.win7.ch.launch.cnt/eu.u.webgl.win7.ch.Playing?t=1d10m&format=json&start=1464033600000"
curl -i "http://localhost:6070/v1/exportstats/rate/eu.website.load.webgl.win7.ch.launch.cnt/eu.u.webgl.win7.ch.Playing?t=2d30m&format=json&start=1464156000000"
