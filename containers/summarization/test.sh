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

main() {
    # Test subdirs
    rm -rf $SCRIPT_DIR/outputs
    docker run -it --rm -v $SCRIPT_DIR/../test/testdata/images:/inputs -v $SCRIPT_DIR/outputs:/outputs  --entrypoint "" $IMAGE run
    checkError
    # for res in '25' '50' '75'; do
    #     checkFileExists "$SCRIPT_DIR/outputs/$res/image1.jpg"
    #     checkFileExists "$SCRIPT_DIR/outputs/$res/subdir/an_image.jpg"
    # done

    # # Test input is a file
    # rm -rf $SCRIPT_DIR/outputs
    # docker run -it --rm -v $SCRIPT_DIR/../test/testdata/images/subdir/an_image.png:/inputs -v $SCRIPT_DIR/outputs:/outputs  --entrypoint "" $IMAGE run
    # checkError
    # for res in '25' '50' '75'; do
    #     checkFileExists "$SCRIPT_DIR/outputs/$res/file.jpg"
    # done
}

main