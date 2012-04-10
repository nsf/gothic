#!/bin/bash

for i in *.go; do
	go build -o ${i%.go} $i
done
