#!/bin/bash
shopt -s globstar
set -e

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

function debug {
    if [ ! -z "$DEBUG" ]
    then
        echo "$*"
    fi
}


# Ensure there are three arguments
if [ $# -lt 3 ]; then
    echo "Usage: run_program <command> <input_dir> <output_dir> <optional_sub_dir>"
    exit 1
fi

COMMAND=$1
INPUT_DIR=$2
BASE_OUTPUT_DIR=${3%/}
if [ $# -eq 4 ]; then
    SUB_DIR="/"${4%/}
    debug "using subdir: $SUB_DIR"
else
    SUB_DIR=""
    debug "not using subdir"
fi
MODE="batch" # Operation Mode: batch, single
DEFAULT_FILENAME="${DEFAULT_FILENAME:-file}"
DEFAULT_EXTENSION="${DEFAULT_EXTENSION:-}" # If blank, will use the `file` binary to determine the extension
APPEND_EXTENSION="${APPEND_EXTENSION:-}" # If set, append this extension to the output file
VALID_EXTENSIONS="${VALID_EXTENSIONS:-}" # If set, only process files with these extensions
PARALLEL="${PARALLEL:-}" # If set, run in parallel mode

# Check to see if input directory is actually a file. This happens when the
# input CID is a blob.
if [ -f "$INPUT_DIR" ]; then
    # Copy the file to a new temp location with an extension
    RANDOM_DIR=$(echo $RANDOM | md5sum | head -c 20)
    TMP_DIR="/tmp/${RANDOM_DIR}"
    mkdir -p ${TMP_DIR}
    TMP_FILE="${TMP_DIR}/$DEFAULT_FILENAME"
    debug "input is a file, writing to $TMP_FILE"
    cp ${INPUT_DIR} ${TMP_FILE}

    # Set the new input directory, because we can't overwrite the original
    INPUT_DIR=${TMP_DIR}
fi

# If mode is batch, then walk over all files in the input directory and run the
# program
if [ $MODE = "batch" ]; then
    # Find all files in the input directory
    for input_file in ${INPUT_DIR}/**/{*,.[^.],.??*}; do # Whitespace-safe and recursive
        debug "processing input_file: $input_file"

        # if it is a directory, continue
        if [ -d "$input_file" ]; then
            continue
        fi

        # if file is empty, continue
        if [ ! -s "$input_file" ]; then
            continue
        fi
        
        # Parse the subpath
        debug "input_file: $input_file"
        subpath=${input_file#"${INPUT_DIR}"}
        debug "subpath: ...$subpath"

        # if extension is empty, set it to mp4
        filename="${input_file##*/}"                      # Strip longest match of */ from start
        dir="${fullpath:0:${#fullpath} - ${#filename}}" # Substring from 0 thru pos of filename
        base="${filename%.[^.]*}"                       # Strip shortest match of . plus at least one non-dot char from end
        ext="${filename:${#base} + 1}"                  # Substring from len of base thru end
        if [[ -z "$base" && -n "$ext" ]]; then          # If we have an extension and no base, it's really the base
            base=".${ext}"
            ext=""
        fi
        debug "base: $base, ext: $ext"
        if [ -z "$ext" ]; then
            if [ -z "$DEFAULT_EXTENSION" ] ; then
                if [ $(command -v file) ] ; then # Ensure file is installed
                    debug "ext is empty, using file to get extension"
                    # if extension is empty, use `file` to get the extension
                    extensions=$(file $input_file --extension --brief)
                    debug "extensions: $extensions"
                    ext=${extensions%%/*}

                    # Sometimes the extension is reported as ???, default to mime-type
                    if [ "$ext" = "???" ]; then
                        mime=$(file $input_file --mime-type --brief)
                        ext=${mime#*/}
                    fi
                else
                    debug "file is not installed"
                fi
            else
                debug "ext is empty, setting to $DEFAULT_EXTENSION"
                ext=$DEFAULT_EXTENSION
            fi
            debug "new ext: $ext"

            # Copy the file to a new temp location with an extension
            RANDOM_DIR=$(echo $RANDOM | md5sum | head -c 20)
            TMP_DIR="/tmp/${RANDOM_DIR}"
            mkdir -p ${TMP_DIR}
            if [ -z $ext ] ; then
                TMP_FILE="${TMP_DIR}/$filename"
            else
                TMP_FILE="${TMP_DIR}/$filename.$ext"
            fi
            debug "${input_file} is a file, copying to $TMP_FILE"
            cp ${input_file} ${TMP_FILE}

            # Set the new input directory, because we can't overwrite the original
            input_file=${TMP_FILE}
        fi

        # If the ext is not in the valid extensions, skip
        if [ ! -z "$VALID_EXTENSIONS" ]; then
            if [[ ",$VALID_EXTENSIONS," != *",$ext,"* ]]; then
                debug "ext $ext is not in valid extensions $VALID_EXTENSIONS, skipping"
                continue
            fi
        fi
        
        output_dir=$(dirname "${BASE_OUTPUT_DIR}${SUB_DIR}${subpath}")

        # Sometimes we want to remove the bacalhau directories
        if [ "$FLATTEN_BACALHAU_DIR" ]; then
            debug "FLATTEN_BACALHAU_DIR is set, removing bacalhau directories from $output_dir"
            # Remove all directories named `default` from the output dir
            output_dir=${output_dir//\/default/}
            # If this file is stdout then append to the base stdout and continue
            if [ "$base" = "stdout" ]; then
                debug "base is stdout, appending ${input_file} to base stdout"
                cat ${input_file} >> ${BASE_OUTPUT_DIR}/stdout
                continue
            fi
            # If this file is stderr then append to the base stderr and continue
            if [ "$base" = "stderr" ]; then
                debug "base is stderr, appending ${input_file} to base stderr"
                cat ${input_file} >> ${BASE_OUTPUT_DIR}/stderr
                continue
            fi
        fi

        # Create the output directory
        debug "output_dir: $output_dir"
        mkdir -p $output_dir

        # Set the output path
        output_file="${output_dir}/$(basename "$input_file")"

        # Escape the input/output paths
        debug "input_file: $input_file"
        input_file=$(printf '%q' "$input_file")
        debug "input_file: $input_file"
        output_file=$(printf '%q' "$output_file")

        # Add the appened extension if required
        if [ ! -z "$APPEND_EXTENSION" ]; then
            output_file="${output_file}.${APPEND_EXTENSION}"
        fi
        output_dir=$(dirname "${output_file}")

        # Template the run command
        debug "COMMAND: $COMMAND"
        rendered_command=$(eval "echo $COMMAND") # Danger! Can expose things like $USER.
        debug "rendered_command: $rendered_command"

        # If running in parallel, run the command in the background and store the pid
        if [ $PARALLEL ]; then
            debug "running in parallel mode"
            nohup bash -c "${rendered_command}" &
        else
            debug "running in serial mode"
            # Run the command
            bash -c "${rendered_command}"
        fi
    done

    if [ $PARALLEL ]; then
        debug "waiting for all pids to finish: $(jobs -p)"
        for job in $(jobs -p); do 
            debug "waiting for $job"
            wait ${job};
        done
    fi
fi
