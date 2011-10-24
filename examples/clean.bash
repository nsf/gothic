#!/bin/bash

for i in *.go; do
	rm ${i%.go}
done
