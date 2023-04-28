#!/bin/bash
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
IMAGE=ghcr.io/bacalhau-project/amplify/ffmpeg:latest

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
    docker run -it --rm -v $SCRIPT_DIR/../test/testdata/videos:/inputs:ro -v $SCRIPT_DIR/outputs:/outputs  --entrypoint "" $IMAGE run
    checkError
    for res in '1080' '720' '480' '360' '240' '144'; do
        checkFileExists "$SCRIPT_DIR/outputs/$res/video (1).mp4"
        checkFileExists "$SCRIPT_DIR/outputs/$res/videosubdir/video2.mp4"
    done

    # Test input is a file
    rm -rf $SCRIPT_DIR/outputs
    docker run -it --rm -v $SCRIPT_DIR/../test/testdata/videos/videosubdir/video2.mp4:/inputs:ro -v $SCRIPT_DIR/outputs:/outputs  --entrypoint "" $IMAGE run
    checkError
    for res in '1080' '720' '480' '360' '240' '144'; do
        checkFileExists $SCRIPT_DIR/outputs/$res/file.mp4
    done
}

main