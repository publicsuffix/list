#!/bin/sh

set -e

# 1. lint the PSL
(
  cd linter
  ./pslint_selftest.sh
  ./pslint.py ../public_suffix_list.dat
)

# 2. run the libpsl self test
test -d libpsl || git clone --depth=1 https://github.com/rockdaboot/libpsl
(
  DIR=`pwd`
  cd libpsl
  git pull
  echo "EXTRA_DIST =" >gtk-doc.make
  echo "CLEANFILES =" >>gtk-doc.make
  autoreconf --install --force --symlink
  OPTIONS="--with-psl-file=$DIR/public_suffix_list.dat --with-psl-testfile=$DIR/tests/tests.txt"

  # Test PSL data with libicu (IDNA2008 UTS#46)
  ./configure -q -C --enable-runtime=libicu --enable-builtin=libicu $OPTIONS && make -s clean && make -s check -j4

  # TEST PSL data with libidn2 (IDNA2008)
  # ./configure -q -C --enable-runtime=libidn2 --enable-builtin=libidn2 $OPTIONS && make -s clean && make -s check -j4

  # TEST PSL data with libidn (IDNA2003)
  # ./configure -q -C --enable-runtime=libidn --enable-builtin=libidn $OPTIONS && make -s clean && make -s check -j4
)
