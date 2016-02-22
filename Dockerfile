FROM golang:1.6-onbuild

ENV env docker-prod

RUN mkdir /data
VOLUME /data

CMD ["app"]

