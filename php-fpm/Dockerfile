FROM centos:7
RUN yum update -y && \
    yum install -y epel-release

RUN yum install -y http://rpms.remirepo.net/enterprise/remi-release-7.rpm && \
    yum install -y yum-utils && \
    yum-config-manager --enable remi-php72 && \
    yum -y update

RUN yum install -y php-fpm php-xml php-cli php-bcmath php-dba php-gd php-intl php-mbstring php-mysql php-pdo php-soap php-pecl-redis php-pecl-imagick

#RUN useradd -M -d /opt/app -s /bin/false nginx
RUN useradd -M -s /bin/false -u 1000 nginx

RUN mkdir -p /run/php-fpm && \
    chown nginx:nginx /run/php-fpm

RUN mkdir -p /var/lib/php/session && \
    chown nginx:nginx /var/lib/php/session

COPY php-fpm.conf /etc/php-fpm.conf
COPY www.conf /etc/php-fpm.d/www.conf
COPY php.ini /etc/php.ini

RUN yum clean all

CMD ["php-fpm"]