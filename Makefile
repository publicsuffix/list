Dir	= $(PWD)
Options	= --with-psl-file=$(Dir)/public_suffix_list.dat --with-psl-testfile=$(Dir)/tests/tests.txt

all: test

test: test-syntax test-rules

test-rules: libpsl-libicu

test-syntax:
	@
	  cd linter;                                \
	  ./pslint_selftest.sh;                     \
	  ./pslint.py ../public_suffix_list.dat;

libpsl-config:
	@
	  test -d libpsl || git clone --depth=1 https://github.com/rockdaboot/libpsl;   \
	  cd libpsl;                                                                    \
	  git pull;                                                                     \
	  echo "EXTRA_DIST =" >  gtk-doc.make;                                          \
	  echo "CLEANFILES =" >> gtk-doc.make;                                          \
	  autoreconf --install --force --symlink;

# Test PSL data with libicu (IDNA2008 UTS#46)
libpsl-libicu: libpsl-config
	cd libpsl && ./configure -q -C --enable-runtime=libicu --enable-builtin=libicu $(Options) && make -s clean && make -s check -j4

# TEST PSL data with libidn2 (IDNA2008)
libpsl-libidn2: libpsl-config
	cd libpsl && ./configure -q -C --enable-runtime=libidn2 --enable-builtin=libidn2 $(Options) && make -s clean && make -s check -j4

# TEST PSL data with libidn (IDNA2003)
libpsl-libidn: libpsl-config
	cd libpsl && ./configure -q -C --enable-runtime=libidn --enable-builtin=libidn $(Options) && make -s clean && make -s check -j4