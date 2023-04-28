#!/usr/bin/env python3

import sys
import os
import pandas as pd
import pathlib
from ydata_profiling import ProfileReport

df = pd.read_csv(sys.argv[1], sep=None, encoding_errors='replace')
profile = ProfileReport(df, minimal=False, plot={"dpi": 300, "image_format": "png"},)

# print to stdout
print(profile.to_json())

# print to file
output_html = os.path.join(sys.argv[2], pathlib.Path(sys.argv[1]).name + ".html")
print(output_html, file=sys.stderr)
profile.to_file(output_html)
