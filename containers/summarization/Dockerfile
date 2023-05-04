FROM docker.io/huggingface/transformers-cpu:4.18.0
#FROM ubuntu:22.04

# bake model into container
RUN mkdir -p /models
RUN python3 -c 'from transformers import pipeline; pipeline("summarization", model="facebook/bart-large-cnn").save_pretrained("/models/bart-large-cnn")'

RUN mkdir -p /usr/local/bin
ENV PATH="/usr/local/bin:${PATH}"
COPY containers/scripts/run_program.sh /usr/local/bin/run_program.sh
ENTRYPOINT ["/usr/local/bin/run"] # This will get overwritten by Amplify
COPY containers/summarization/run /usr/local/bin/run
COPY containers/summarization/bart-summarize.py /usr/local/bin/bart-summarize.py