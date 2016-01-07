FROM ubuntu:14.04.3

RUN apt-get update && \
    apt-get install -y build-essential wget libncurses5-dev unzip bc curl python rsync ccache

RUN locale-gen en_US.UTF-8
ENV LANG en_US.UTF-8

ENV DAPPER_SOURCE /source
ENV DAPPER_OUTPUT ./dist
ENV SHELL /bin/bash
WORKDIR ${DAPPER_SOURCE}
