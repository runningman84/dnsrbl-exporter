dnsrbl-exporter
============

[![](https://images.microbadger.com/badges/version/runningman84/dnsrbl-exporter.svg)](https://hub.docker.com/r/runningman84/dnsrbl-exporter "Click to view the image on Docker Hub")
[![](https://images.microbadger.com/badges/image/runningman84/dnsrbl-exporter.svg)](https://hub.docker.com/r/runningman84/dnsrbl-exporter "Click to view the image on Docker Hub")
[![](https://img.shields.io/docker/stars/runningman84/dnsrbl-exporter.svg)](https://hub.docker.com/r/runningman84/dnsrbl-exporter "Click to view the image on Docker Hub")
[![](https://img.shields.io/docker/pulls/runningman84/dnsrbl-exporter.svg)](https://hub.docker.com/r/runningman84/dnsrbl-exporter "Click to view the image on Docker Hub")

Introduction
----
This project is a dns realtime blacklist checker with a prometheus endpoint.

Install
----

```sh
docker pull runningman84/dnsrbl-exporter
```

Running
----

```sh
docker run -d -P -p 8000:8000 runningman84/dnsrbl-exporter
```

The container can be configured using these ENVIRONMENT variables:

Key | Description | Default
------------ | ------------- | -------------
DNSRBL_HTTP_BL_ACCESS_KEY | API Key for https://www.projecthoneypot.org | None
DNSRBL_DELAY_REQUESTS | Sleep time between two subsequent requests (single list check) | 1
DNSRBL_DELAY_RUNS | Sleep time between two subsequent runs (full list check) | 60
DNSRBL_LISTS | Space seperated list of rbls like "dnsbl.httpbl.org zen.spamhaus.org" | None
DNSRBL_LISTS_FILENAME | Filname containing list of rbls, one line per server | lists.txt
DNSRBL_CHECK_IP | IP address to be checked, auto discovery if not set | None
DNSRBL_PORT | Listener port for metric server | 8000

Finally
----
You can integrate dnsrbl-exporter with my helm chart and run it in a kubernetes cluster.

