FROM alpine:3.20
COPY cligram /usr/bin/cligram
ENTRYPOINT ["/usr/bin/cligram"]