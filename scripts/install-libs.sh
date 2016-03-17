#!/bin/bash
set -ex

toolchains="arm-linux-gnueabihf aarch64-linux-gnu"
libs="libselinux libsepol pcre3"

cd /usr/local/src
for i in $libs; do
    apt-get build-dep -y $i
    apt-get source -y $i
done
for TOOLCHAIN in $toolchains; do
    apt-get install -y gcc-${TOOLCHAIN} g++-${TOOLCHAIN}
    cd /usr/local/src/pcre3-*
    autoreconf
    CC=${TOOLCHAIN}-gcc CXX=${TOOLCHAIN}-g++ ./configure --host=${TOOLCHAIN} --prefix=/usr/${TOOLCHAIN}
    make -j$(nproc)
    make install && make distclean
    cd /usr/local/src/libselinux-*
    CC=${TOOLCHAIN}-gcc CXX=${TOOLCHAIN}-g++ make CFLAGS=-Wall
    make PREFIX=/usr/${TOOLCHAIN} DESTDIR=/usr/${TOOLCHAIN} install && make clean
    cd /usr/local/src/libsepol-*
    CC=${TOOLCHAIN}-gcc CXX=${TOOLCHAIN}-g++ make CFLAGS=-Wall
    make PREFIX=/usr/${TOOLCHAIN} DESTDIR=/usr/${TOOLCHAIN} install && make clean
done
