FROM debian:bookworm-slim

COPY ./kube-scheduler /usr/local/bin/kube-scheduler
COPY ./kdaptScheduler.yaml /config/scheduler-config.yaml

CMD ["tail", "-f", "/dev/null"]
