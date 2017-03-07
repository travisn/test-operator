FROM ubuntu

COPY test-operator /opt/test/bin/
ENTRYPOINT ["/opt/test/bin/test-operator"]