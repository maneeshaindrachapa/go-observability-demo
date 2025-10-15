FROM otel/opentelemetry-collector-contrib:0.91.0

COPY config/otel-collector-config.yaml /etc/otel-collector-config.yaml

ENTRYPOINT ["/otelcol-contrib"]
CMD ["--config=/etc/otel-collector-config.yaml"]
