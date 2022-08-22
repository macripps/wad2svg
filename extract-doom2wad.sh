#!/usr/bin/env bash
mkdir -p output/$2
for m in {1..32}
do
  map=$(printf "%02d" $m)
  go run main.go "$1" "MAP${map}" > "output/$2/MAP${map}.svg"
done
