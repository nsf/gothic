#!/bin/bash

for i in *.go; do
	rm -f ${i%.go}
done
rm -f *.[568]
