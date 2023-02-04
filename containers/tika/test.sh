#!/bin/bash
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
IMAGE=ghcr.io/bacalhau-project/amplify/tika:latest

# checkError ensures previous command succeeded
checkError() {
    if [ $? -ne 0 ]; then
        echo "Failed to run tika"
        exit 1
    fi
}

# checkFileExists ensures that a file exists
checkFileExists() {
    if [ ! -f $1 ]; then
        echo "File $1 does not exist"
        exit 1
    fi
}

main() {

    # Test input is a file
    rm -rf $SCRIPT_DIR/outputs
    docker run -it --rm -v $SCRIPT_DIR/..:/containers -v $SCRIPT_DIR/outputs:/outputs  --entrypoint "" $IMAGE extract-metadata /containers/test/testdata/file/dummy.pdf /outputs
    checkError
    checkFileExists $SCRIPT_DIR/outputs/metadata.json

    # Test directory path
    rm -rf $SCRIPT_DIR/outputs
    docker run -it --rm -v $SCRIPT_DIR/..:/containers -v $SCRIPT_DIR/outputs:/outputs  --entrypoint "" $IMAGE extract-metadata /containers/test/testdata/file /outputs
    checkError
    checkFileExists $SCRIPT_DIR/outputs/dummy.pdf.json

    # Test multiple files in a directory
    rm -rf $SCRIPT_DIR/outputs
    docker run -it --rm -v $SCRIPT_DIR/..:/containers -v $SCRIPT_DIR/outputs:/outputs  --entrypoint "" $IMAGE extract-metadata /containers/test/testdata/files /outputs
    checkError
    checkFileExists $SCRIPT_DIR/outputs/dummy.pdf.json
    checkFileExists $SCRIPT_DIR/outputs/dummy.csv.json

    # Test subdirs
    rm -rf $SCRIPT_DIR/outputs
    docker run -it --rm -v $SCRIPT_DIR/..:/containers -v $SCRIPT_DIR/outputs:/outputs  --entrypoint "" $IMAGE extract-metadata /containers/test/testdata/subdir /outputs
    checkError
    checkFileExists $SCRIPT_DIR/outputs/dummy.pdf.json
    checkFileExists $SCRIPT_DIR/outputs/dir/dummy.csv.json
}

main