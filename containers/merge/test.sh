#!/bin/bash
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
IMAGE=ghcr.io/bacalhau-project/amplify/merge:latest

# checkError ensures previous command succeeded
checkError() {
    if [ $? -ne 0 ]; then
        echo "Failed to run"
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

    # Test remove bacalhau structure
    rm -rf $SCRIPT_DIR/outputs
    docker run -it --rm \
    -v $SCRIPT_DIR/../test/testdata/bacalhau:/inputs/images:ro \
    -v $SCRIPT_DIR/../test/testdata/file:/inputs/src:ro \
    -v $SCRIPT_DIR/outputs:/outputs --entrypoint "" $IMAGE run
    checkError
    checkFileExists "$SCRIPT_DIR/outputs/src/dummy.pdf"
    checkFileExists "$SCRIPT_DIR/outputs/images/image1.png"
    checkFileExists "$SCRIPT_DIR/outputs/images/subdir/an_image.png"
    checkFileExists "$SCRIPT_DIR/outputs/stdout"
    # Test that stdout and stderr are merged
    if ! (grep -q "hello stdout" "$SCRIPT_DIR/outputs/stdout") && (grep -q "sub stdout" "$SCRIPT_DIR/outputs/stdout") ; then
        exit 1
    fi
    if ! (grep -q "hello stderr" "$SCRIPT_DIR/outputs/stderr") && (grep -q "sub stderr" "$SCRIPT_DIR/outputs/stderr") ; then
        exit 1
    fi

    # Test muliple volumes
    rm -rf $SCRIPT_DIR/outputs
    docker run -it --rm \
    -v $SCRIPT_DIR/../test/testdata/images:/inputs/images:ro \
    -v $SCRIPT_DIR/../test/testdata/file:/inputs/src:ro \
    -v $SCRIPT_DIR/../test/testdata/csv:/inputs/csvs:ro \
    -v $SCRIPT_DIR/outputs:/outputs --entrypoint "" $IMAGE run
    checkError
    checkFileExists "$SCRIPT_DIR/outputs/src/dummy.pdf"
    checkFileExists "$SCRIPT_DIR/outputs/images/picture.jpg"
    checkFileExists "$SCRIPT_DIR/outputs/csvs/subdir/country-3-valid.csv"

    # Test on all files
    rm -rf $SCRIPT_DIR/outputs
    docker run -it --rm -v $SCRIPT_DIR/../test/testdata:/inputs:ro -v $SCRIPT_DIR/outputs:/outputs --entrypoint "" $IMAGE run
    checkError
    checkFileExists "$SCRIPT_DIR/outputs/videos/video (1).mp4" # Spaces
    checkFileExists "$SCRIPT_DIR/outputs/bad_names/0" # Does not fix extensions, just copies
    checkFileDoesntExists "$SCRIPT_DIR/outputs/images/subdir/empty_image.png" # Ignores empty files

    # Test single file
    rm -rf $SCRIPT_DIR/outputs
    docker run -it --rm -v $SCRIPT_DIR/../test/testdata/images/picture.jpg:/inputs/picture.jpg:ro -v $SCRIPT_DIR/outputs:/outputs  --entrypoint "" $IMAGE run 
    checkError
    checkFileExists $SCRIPT_DIR/outputs/picture.jpg

    # Test input is a file
    rm -rf $SCRIPT_DIR/outputs
    docker run -it --rm -v $SCRIPT_DIR/../test/testdata/images/picture.jpg:/inputs:ro -v $SCRIPT_DIR/outputs:/outputs  --entrypoint "" $IMAGE run
    checkError
    checkFileExists $SCRIPT_DIR/outputs/file
}

main