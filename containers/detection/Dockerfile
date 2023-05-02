# mac users wanting to test locally must change the tag to: latest-arm64
# it will run, but will be really slow
FROM ultralytics/ultralytics@sha256:ae02bb4835db50cd355e29e0c5564b0825b934038b20fcd6441d999831e1f669
RUN apt-get install -y file rsync wget
RUN wget https://github.com/ultralytics/assets/releases/download/v0.0.0/yolov8n-pose.pt -O /usr/src/ultralytics/yolov8n-pose.pt
RUN wget https://github.com/ultralytics/assets/releases/download/v0.0.0/yolov8n-seg.pt -O /usr/src/ultralytics/yolov8n-seg.pt
RUN mkdir -p /usr/local/bin
ENV PATH="/usr/local/bin:${PATH}"
COPY containers/scripts/run_program.sh /usr/local/bin/run_program.sh
COPY containers/detection/run /usr/local/bin/run
COPY containers/detection/yolo_config.yaml /workspace/config.yaml
ENTRYPOINT ["run"]
