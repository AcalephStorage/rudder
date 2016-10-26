FROM alpine
MAINTAINER admin@acale.ph

ADD build/bin/rudder /usr/local/bin/rudder

EXPOSE 5000
ENTRYPOINT ["/usr/local/bin/rudder"]
