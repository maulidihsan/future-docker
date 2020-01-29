FROM centos:7

RUN yum update -y && \
    yum install -y epel-release

RUN yum install -y nginx

ADD nginx.conf /etc/nginx/nginx.conf
RUN yum clean all

CMD ["nginx"]
