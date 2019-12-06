#!/bin/bash

ROOT=$(dirname $0)

FROM=$1
NODENUM=$2
AMOUNT=$3
RESULT=$4

AMO1=1000000000000000000

fail() {
	echo "test failed"
	echo $1
	exit -1
}

$ROOT/qs.sh "$NODENUM"

. testaddr.sh

for ((i=FROM; i<=NODENUM; i++))
do
    printf "withdraw val$i: $(bc <<< "$AMOUNT / $AMO1") AMO - "

	out=$($CLI tx --broadcast=commit withdraw $CLIOPT --user tval$i "$AMOUNT")
	h=$(echo $out | python -c "import sys, json; print json.load(sys.stdin)['deliver_tx']['info']")
	if [ -z "$h" -o "$h" != "$RESULT" ]; then fail $out; fi

	printf "$h\n"
done

$ROOT/qs.sh "$NODENUM"

