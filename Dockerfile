FROM golang:1.5.3-onbuild

ENV env docker-prod

RUN ulimit -n 5000
CMD ["app"]

