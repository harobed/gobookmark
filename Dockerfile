FROM debian:jessie
RUN mkdir /data/
WORKDIR /
ADD ./releases/linux_amd64/gobookmark /gobookmark
ENV GOBOOKMARK_DATABASE=/data/gobookmark
ENV GOBOOKMARK_HOST=0.0.0.0
EXPOSE 8000
CMD /gobookmark web
