FROM golang:1.5.3-onbuild

RUN ulimit -n 5000
CMD ["app"]

