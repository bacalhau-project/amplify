#!/bin/bash
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
IMAGE=ghcr.io/bacalhau-project/amplify/frictionless-extract:latest

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
    docker run -it --rm -v $SCRIPT_DIR/../test/testdata/various_tables:/inputs -v $SCRIPT_DIR/outputs:/outputs  --entrypoint "" $IMAGE run
    checkError
    
    checkFileExists "$SCRIPT_DIR/outputs/table1.csv.csv"
    checkFileExists "$SCRIPT_DIR/outputs/table1.csv.zip.csv"
    checkFileExists "$SCRIPT_DIR/outputs/table1.parquet.csv"
    checkFileExists "$SCRIPT_DIR/outputs/table2.xlsx.csv"
}

main