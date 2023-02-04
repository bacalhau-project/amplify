# Containers

This directory contains the source code for the pre-built containers that are used for parsing metadata from data.

## Container Contract

### Extract Metadata

* Containers must execute using the following pattern: `extract-metadata <input> <output_dir>`
* Containers must be able to batch-process all files within a directory. [See tika/test.sh](tika/test.sh) for an example.
* Containers must be able to process when `<input>` is a file or a directory.
