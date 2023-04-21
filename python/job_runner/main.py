#!/usr/bin/python3
# -*- coding: utf-8 -*-


"""
WIP - This script is meant to be run in an Amplify job and executes <any> binary such that it complies with Amplify's job requirements.

Examples:

- python job_runner/main.py /tmp/inputs /tmp/outputs 33,22 'magick mogrify -format jpg -resize ${param}% -quality 100 -path ${output} ${input}' --rm_suffix=true
- python job_runner/main.py /tmp/inputs_video /tmp/outputs_video 720,320 'ffmpeg -y -i ${input} -vcodec libx264 -vf "scale=${param}:-2,setsar=1" -f mp4 ${output}' --rm_suffix=false

Tested on Python 3.9.12

Features:
- check if file type is supported (? or just try to convert)
- manipulate paths as objects not strings
- replace /input/<path> with /outputs/<path>
- mkdir supporting levels
- replace file extension (if any)
- shell out to given command
- no dependencies other than python3

Why not using Bash:
    Escaping hell
    single file CID vs directory CID, file with or without extension
    Unsupported files (e.g. .DS_Store).
    Empty files.
    uppercase vs lowecase extension.
    Not testable.
    Manipulating paths as strings is too fragile
"""
import subprocess
import os
import sys
from pathlib import PosixPath, Path
import tempfile

global rm_suffix
rm_suffix = False

def recursive_search(path: str):
    """Search for *valid* files in a directory recursively.
    Invalid files are:
        - empty
        - directory
        - hidden files (e.g. .DS_Store)
        - symlink
    """
    path = Path(path)
    if not path.exists():
        raise Exception("Path does not exist: " + str(path))
    
    file_paths = []
    for file_path in path.rglob("*"):
        if file_path.is_dir():
            continue
        if file_path.name.startswith('.'): # e.g. .DS_Store
            continue
        if file_path.suffix == "": # no extension
            continue
        if file_path.is_symlink():
            continue
        if file_path.stat().st_size == 0:
            continue
        file_paths.append(file_path)
    if not file_paths:
        raise Exception("No valid files found in " + str(path))
    return file_paths

def parse_args(args: list):
    global rm_suffix

    input_path = None
    output_base_path = None
    cmd_params = None
    cmd = None

    print(len(args))

    # validate args
    if len(args) <= 1:
        raise Exception("No args provided")
    if len(args) <= 5:
        raise Exception("Not enough args provided (expected 5, got " + str(len(args) - 1) + ")")
    else: 
        input_path = Path(args[1])
        if not input_path.is_dir():
            raise Exception("Input path is not a directory")
        output_base_path = Path(args[2])
        if not output_base_path.is_dir():
            raise Exception("Output path is not a directory")
        cmd_params = args[3]
        if cmd_params == "":
            raise Exception("cmd_params is empty")
        else:
            for param in cmd_params.split(","):
                if param.strip() == "":
                    raise Exception("cmd_params contains an empty value")
        cmd = args[4]
        if cmd == "":
            raise Exception("cmd is empty")
        
        if len(args) == 6:
            if args[5] == "--rm_suffix=true":
                rm_suffix = True
    
    return input_path, output_base_path, cmd_params, cmd

def parse_cmd_params(cmd_params: str):
    """Parse cmd_params into a list"""
    params = []
    for cmd_par in cmd_params.split(","):
        if cmd_par != "":
            raw_param = cmd_par.strip()
            # TODO move this to tests
            # will file-system like it?
            try:
                with tempfile.TemporaryDirectory() as tmpdirname:
                    os.mkdir(os.path.join(tmpdirname, raw_param))
            except:
                raise Exception("Invalid cmd_params: {}, must be file-system compliant".format(raw_param))
            params.append(raw_param)
    if not params:
        raise Exception("No valid cmd_params provided")
    if len(set(params)) != len(params):
        raise Exception("Duplicate cmd_params provided")

    return params

def get_output_paths(input_path, output_base_path, input_files, parsed_params):
    """
    Transform input paths into output paths.
    Also, remove file extension, if any.
    """
    output_paths = []
    for file in input_files:
        for param in parsed_params:
            tmp_path=str(file)
            
            # os.path.join does not like leading slash
            if tmp_path.startswith("/"):
                tmp_path = tmp_path[1:]
            
            # TODO double check this - remove filename suffix
            if rm_suffix:
                tmp_path = str(Path(tmp_path).parent)
            else:
                tmp_path = str(Path(tmp_path))

            output = PosixPath(os.path.join(output_base_path, param, tmp_path))
            # print(output)
            output_paths.append((file, output))
    return output_paths

def main(args):
    input_path, output_base_path, cmd_params, cmd = parse_args(args)

    input_files = recursive_search(input_path)

    # parse params
    parsed_params = parse_cmd_params(cmd_params)

    # transform input paths into output paths
    output_paths = get_output_paths(input_path, output_base_path, input_files, parsed_params)

    # create output directories
    for foo, output_path in output_paths:
        Path(output_path).mkdir(parents=True, exist_ok=True)

    commands = []
    for param in parsed_params:
        for input, output in output_paths:
            new_cmd = (
                cmd
                .replace('${param}', param)
                .replace('${input}', str(input))
                .replace('${output}', str(output))
            )
            commands.append(new_cmd)

    for command in commands:
        pass
        print(command)
        result = subprocess.run(
            command.split(" "),
            shell=False, # must be False!
            check=True,
            capture_output=True,
            text=True
        )
        print("returncode:")
        print(result.returncode)
        print("stdout:")
        print(result.stdout)
        print("stderr:")
        print(result.stderr)

if __name__ == "__main__":
    main(sys.argv)
