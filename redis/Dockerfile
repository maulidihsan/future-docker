FROM centos:7
RUN yum install epel-release -y && \
    yum update -y && \
    yum install redis -y && \
    yum clean all

ADD redis.conf /etc/redis.conf

EXPOSE 6379

CMD ["/usr/bin/redis-server", "/etc/redis.conf"]
