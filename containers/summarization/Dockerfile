FROM docker.io/huggingface/transformers-cpu:4.18.0
RUN mkdir -p /usr/local/bin
ENV PATH="/usr/local/bin:${PATH}"
COPY containers/scripts/run_program.sh /usr/local/bin/run_program.sh
ENTRYPOINT ["/usr/local/bin/run"] # This will get overwritten by Amplify
COPY containers/summarization/run /usr/local/bin/run
COPY containers/summarization/bart-summarize.py /usr/local/bin/bart-summarize.py