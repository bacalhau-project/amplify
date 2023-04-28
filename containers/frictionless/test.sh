#!/bin/bash
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
IMAGE=ghcr.io/bacalhau-project/amplify/frictionless:latest

echo ${SCRIPT_DIR}

# checkError ensures previous command succeeded
checkError() {
    if [ $? -ne 0 ]; then
        echo "Failed to run test, there was an error"
        exit 1
    fi
}

# checkFileExists ensures that a file exists
checkFileExists() {
    if [ ! -f "$1" ]; then
        echo "File $1 does not exist"
        exit 1
    fi
}

main() {
    rm -rf $SCRIPT_DIR/outputs
    docker run -it --rm -v $SCRIPT_DIR/../test/testdata/csv:/inputs:ro -v $SCRIPT_DIR/outputs:/outputs  --entrypoint "" $IMAGE run
    checkError
    
    checkFileExists "$SCRIPT_DIR/outputs/countries-not-valid.csv.metadata.json"
    checkFileExists "$SCRIPT_DIR/outputs/countries-not-valid.csv.validate.json"
    checkFileExists "$SCRIPT_DIR/outputs/subdir/country-3-valid.csv.metadata.json"
    checkFileExists "$SCRIPT_DIR/outputs/subdir/country-3-valid.csv.validate.json"
}

main