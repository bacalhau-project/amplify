#!/bin/bash
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
IMAGE=ghcr.io/bacalhau-project/amplify/detection:latest

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
    # Test single file
    rm -rf $SCRIPT_DIR/outputs
    mkdir -p $SCRIPT_DIR/outputs
    docker run -it --rm -v $SCRIPT_DIR/../test/testdata/images/picture.jpg:/inputs/picture.jpg:ro -v $SCRIPT_DIR/outputs:/outputs  --entrypoint "" $IMAGE run > $SCRIPT_DIR/outputs/capture.txt
    checkError
    checkFileExists $SCRIPT_DIR/outputs/crops/bicycle/picture.jpg
    checkFileExists $SCRIPT_DIR/outputs/capture.txt
    if ! grep -q content-classification "$SCRIPT_DIR/outputs/capture.txt"; then
        echo "No content-classification"
        exit 1
    fi
    filesize=$(cat $SCRIPT_DIR/outputs/capture.txt | wc -c)
    if [ $filesize -lt 20 ]; then
        echo "stdout too small"
        exit 1
    fi

    # Test input is a file
    rm -rf $SCRIPT_DIR/outputs
    docker run -it --rm -v $SCRIPT_DIR/../test/testdata/images/picture.jpg:/inputs:ro -v $SCRIPT_DIR/outputs:/outputs  --entrypoint "" $IMAGE run
    checkError
    checkFileExists $SCRIPT_DIR/outputs/crops/bicycle/file.jpg
    
    # Test input is a directory of images
    rm -rf $SCRIPT_DIR/outputs
    mkdir -p $SCRIPT_DIR/outputs
    docker run -it --rm -v $SCRIPT_DIR/../test/testdata/images:/inputs:ro -v $SCRIPT_DIR/outputs:/outputs  --entrypoint "" $IMAGE run > $SCRIPT_DIR/outputs/capture.txt
    checkError
    checkFileExists $SCRIPT_DIR/outputs/subdir/crops/bicycle/image1.jpg
    if ! grep -q '{"content-classification": "person"}' "$SCRIPT_DIR/outputs/capture.txt"; then
        echo "No '{"content-classification": "person"}'"
        exit 1
    fi

    # Test on videos
    rm -rf $SCRIPT_DIR/outputs
    docker run -it --rm -v $SCRIPT_DIR/../test/testdata/videos/bike.mp4:/inputs:ro -v $SCRIPT_DIR/outputs:/outputs --entrypoint "" $IMAGE run
    checkError
    checkFileExists $SCRIPT_DIR/outputs/crops/bicycle/file.jpg

    # Test on all files
    rm -rf $SCRIPT_DIR/outputs
    docker run -it --rm -v $SCRIPT_DIR/../test/testdata:/inputs:ro -v $SCRIPT_DIR/outputs:/outputs --entrypoint "" $IMAGE run
    checkError
    checkFileExists $SCRIPT_DIR/outputs/images/subdir/crops/bicycle/image1.jpg
}

main