#!/bin/bash

arch=$(uname -m | sed 's/^..86$/386/; s/^.86$/386/; s/x86_64/amd64/; s/arm.*/arm/')
case $arch in
	"386")
		O=8 ;;
	"amd64")
		O=6 ;;
	"arm")
		O=5 ;;
esac

compile=${O}g
link=${O}l

for i in *.go; do
	src=$i
	obj=${i%.go}.$O
	bin=${i%.go}
	$compile $src && $link -o $bin $obj && rm $obj
done
