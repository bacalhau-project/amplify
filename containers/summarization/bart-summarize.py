import os
import sys
import json
import pathlib

from transformers import pipeline


# read input file
with open(sys.argv[1], mode='r', errors='replace') as f:
    text = f.read()
if len(text) <= 0:
    raise Exception("File {} appears to be empty".format(sys.argv[1]))

# facebook/bart-large-cnn can take input up to 1024 char
if len(text) > 1024:
    text = text[:1024]    

# run inference
summarizer = pipeline("summarization", model="/models/bart-large-cnn")
summary_list = summarizer(text, max_length=130, min_length=30, do_sample=False)

# save JSON summary
if summary_list:
    if isinstance(summary_list[0], dict) and ('summary_text' in summary_list[0].keys()):
        json_object = json.dumps(summary_list[0], indent = None)

        print(json_object)
        
        output_file = os.path.join(sys.argv[2], pathlib.Path(sys.argv[1]).name + ".json")
        print(output_file, file=sys.stderr)

        with open(output_file, "w") as outfile:
            outfile.write(json_object)
else:
    raise Exception("Generated summary is empty.")
