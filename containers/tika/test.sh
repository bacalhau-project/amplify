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
    if [ ! -f "$1" ]; then
        echo "File $1 does not exist"
        exit 1
    fi
}

# checkFileDoesntExists ensures that a file does not exist
checkFileDoesntExists() {
    if [ -f "$1" ]; then
        echo "File $1 exists when it shouldn't"
        exit 1
    fi
}

main() {
    rm -rf $SCRIPT_DIR/outputs
    mkdir -p $SCRIPT_DIR/outputs
    
    # Test input is a file
    docker run -it --rm -v $SCRIPT_DIR/../test/testdata/file/dummy.pdf:/inputs -v $SCRIPT_DIR/outputs:/outputs  --entrypoint "" $IMAGE run > $SCRIPT_DIR/outputs/capture.txt
    checkError
    checkFileExists $SCRIPT_DIR/outputs/file.metadata.json
    checkFileExists $SCRIPT_DIR/outputs/capture.txt
    if ! grep -q Content-Type "$SCRIPT_DIR/outputs/capture.txt"; then
        echo "No Content-Type"
        exit 1
    fi
    filesize=$(cat $SCRIPT_DIR/outputs/capture.txt | wc -c)
    echo "filesize is $filesize"
    if [ $filesize -lt 20 ]; then
        echo "stdout too small"
        exit 1
    fi

    # Test multiple files in a directory
    rm -rf $SCRIPT_DIR/outputs
    docker run -it --rm -v $SCRIPT_DIR/../test/testdata:/inputs -v $SCRIPT_DIR/outputs:/outputs  --entrypoint "" $IMAGE run
    checkError
    checkFileExists $SCRIPT_DIR/outputs/file/dummy.pdf.metadata.json
    checkFileExists $SCRIPT_DIR/outputs/files/dummy.csv.metadata.json
    checkFileExists $SCRIPT_DIR/outputs/subdir/dir/dummy.csv.metadata.json
    checkFileExists "$SCRIPT_DIR/outputs/videos/video (1).mp4.metadata.json"

    # Test that a file that doesn't have an extension hasn't been copied to the
    # same directory
    rm -rf $SCRIPT_DIR/outputs
    docker run -it --rm -v $SCRIPT_DIR/../test/testdata/bad_names:/inputs -v $SCRIPT_DIR/outputs:/outputs  --entrypoint "" $IMAGE run
    checkError
    checkFileDoesntExists "$SCRIPT_DIR/../test/testdata/bad_names/0.metadata.json"
    checkFileExists "$SCRIPT_DIR/outputs/.json.metadata.json"
    checkFileExists "$SCRIPT_DIR/outputs/0.metadata.json"

    # Test CSV File
    rm -rf $SCRIPT_DIR/outputs
    mkdir -p $SCRIPT_DIR/outputs
    docker run -it --rm -v $SCRIPT_DIR/../test/testdata/csv:/inputs -v $SCRIPT_DIR/outputs:/outputs  --entrypoint "" $IMAGE run > $SCRIPT_DIR/outputs/capture.txt
    checkError
    if ! grep -q "text/csv" "$SCRIPT_DIR/outputs/capture.txt"; then
        echo "No text/csv"
        exit 1
    fi

}

main