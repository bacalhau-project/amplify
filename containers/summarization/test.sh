#!/bin/bash
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
IMAGE=ghcr.io/bacalhau-project/amplify/summarization:latest

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

checkFileDoesNotExists() {
    if [ -f "$1" ]; then
        echo "File $1 does exist"
        exit 1
    fi
}

main() {
    # Test blob
    rm -rf $SCRIPT_DIR/outputs
    docker run -it --rm -v $SCRIPT_DIR/../test/testdata/text_blob/somethoughts:/inputs -v $SCRIPT_DIR/outputs:/outputs  --entrypoint "" $IMAGE run
    checkError
    checkFileExists "$SCRIPT_DIR/outputs/file.plain.json"

    # Test subdir
    rm -rf $SCRIPT_DIR/outputs
    docker run -it --rm -v $SCRIPT_DIR/../test/testdata/text_dir:/inputs -v $SCRIPT_DIR/outputs:/outputs  --entrypoint "" $IMAGE run
    checkError
    checkFileExists "$SCRIPT_DIR/outputs/subdir/codfish.txt.json"
    checkFileExists "$SCRIPT_DIR/outputs/looneytunes.plain.json"
    # Test empty file
    checkFileDoesNotExists "$SCRIPT_DIR/outputs/empty.plain.json"

    # Test non textual files
    rm -rf $SCRIPT_DIR/outputs
    docker run -it --rm -v $SCRIPT_DIR/../test/testdata/images:/inputs -v $SCRIPT_DIR/outputs:/outputs  --entrypoint "" $IMAGE run
    checkError

    # Json file
    rm -rf $SCRIPT_DIR/outputs
    docker run -it --rm -v $SCRIPT_DIR/../test/testdata/json_blob:/inputs -v $SCRIPT_DIR/outputs:/outputs  --entrypoint "" $IMAGE run
    checkError
    checkFileDoesNotExists "$SCRIPT_DIR/outputs/bafkreibd4mqgydtbi5vuygtti2eiugyxiqjzwsaexvs7ofmqyrsmsvmosi.plain.json"
}

main