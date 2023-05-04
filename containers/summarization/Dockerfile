FROM docker.io/huggingface/transformers-cpu:4.18.0

# bake model into container
RUN mkdir -p /models
RUN python3 -c 'from transformers import pipeline; pipeline("summarization", model="facebook/bart-large-cnn").save_pretrained("/models/bart-large-cnn")'

RUN apt-get update && apt-get install -y --no-install-recommends \
    wget \
    && rm -rf /var/lib/apt/lists/*

RUN wget https://launchpad.net/ubuntu/+archive/primary/+sourcefiles/file/1:5.41-3/file_5.41.orig.tar.gz && tar -xf file_5.41.orig.tar.gz
RUN cd file-5.41 && \
    FORCE_UNSAFE_CONFIGURE=1 ./configure --prefix=/root/filecmd/ && \
    make && make install && \
    cp /root/filecmd/bin/file /usr/local/bin/file5.41 && \
    chmod +x /usr/local/bin/file5.41

RUN mkdir -p /usr/local/bin
ENV PATH="/usr/local/bin:${PATH}"
COPY containers/scripts/run_program.sh /usr/local/bin/run_program.sh
ENTRYPOINT ["/usr/local/bin/run"] # This will get overwritten by Amplify
COPY containers/summarization/run /usr/local/bin/run
COPY containers/summarization/bart-summarize.py /usr/local/bin/bart-summarize.py