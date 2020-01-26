FROM centos:7

RUN yum update -y && \
    yum install -y epel-release

RUN yum install -y nginx gettext

ADD nginx.conf.tmpl /
ADD entrypoint.sh /usr/bin/entrypoint
RUN yum clean all

EXPOSE 80

ENTRYPOINT ["entrypoint"]
CMD ["nginx"]