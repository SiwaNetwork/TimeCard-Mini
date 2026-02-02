# Welcome to Timebeat 2.2.20

PTP / NTP clock synchronisation system

## Getting Started

To get started with Timebeat, you need to set up Elasticsearch on
your localhost first. After that, start Timebeat with:

     ./timebeat -c timebeat.yml -e

This will start Timebeat and send the data to your Elasticsearch
instance. To load the dashboards for Timebeat into Kibana, run:

    ./timebeat setup -e

For further steps visit the
[Quick start](https://www.elastic.co/guide/en/beats/timebeat/main/timebeat-installation-configuration.html) guide.

## Documentation

Visit [Elastic.co Docs](https://www.elastic.co/guide/en/beats/timebeat/main/index.html)
for the full Timebeat documentation.

## Release notes

https://www.elastic.co/guide/en/beats/libbeat/main/release-notes-2.2.20.html
