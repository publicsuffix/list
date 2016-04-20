#!/bin/sh

rc=0
rm -rf log
mkdir -p log

# add CR if missing, it won't possibly survive git
sed -i -e 's/^e.example.com$/e.example.com\r/g' test_spaces.input

for file in `ls *.input|cut -d'.' -f1`; do
  echo -n "${file}: "
  ./pslint.py ${file}.input >log/${file}.log 2>&1
  diff -u ${file}.expected log/${file}.log >log/${file}.diff
  if [ $? -eq 0 ]; then
    echo OK
    rm log/${file}.diff log/${file}.log
  else
    echo FAILED
    cat log/${file}.diff
    rc=1
  fi
done

# remove CR, to not appear as changed to git
sed -i -e 's/^e.example.com\r$/e.example.com/g' test_spaces.input

if [ $rc -eq 0 ]; then
  rmdir log
fi

exit $rc
