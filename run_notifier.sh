#!/bin/bash

# This script is used to run the notifier service

go run cmd/notifier/main.go \
    -p ${POSTCODE} \
    -a ${ADDRESS_CODE} \
    -f ${FROM_NUMBER} \
    -n ${TO_NUMBER} \
    -r ${COLLECTION_DAY}
