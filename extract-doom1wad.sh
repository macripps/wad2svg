#!/usr/bin/env bash
mkdir -p output/$2
for e in {1..4}
do
  for m in {1..9}
  do
    go run main.go "$1" "E${e}M${m}" > "output/$2/E${e}M${m}.svg"
  done

done
