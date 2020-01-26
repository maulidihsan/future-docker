FROM centos:7
RUN yum update -y && \
    yum install -y epel-release && \
    yum install -y rabbitmq-server && \
    rabbitmq-plugins enable rabbitmq_management && \

RUN chown -R rabbitmq:rabbitmq /var/lib/rabbitmq/
ADD entrypoint.sh /
ADD rabbitmq.config /etc/rabbitmq/

EXPOSE 5672 15672

RUN yum clean all
ENTRYPOINT ["/entrypoint.sh"]