FROM ubuntu:24.04
WORKDIR /opt/app
ARG MOD_SECURITY_VERIONS=v3.0.13
ARG NGINX_VERSION=1.26.2
ENV PATH=$PATH:/opt/app/nginx/sbin
RUN apt -y update \
    && apt -y install \
     build-essential \
     git \
     libpcre3 \
     libpcre3-dev \
     zlib1g \
     zlib1g-dev \
     libssl-dev \
     libgd-dev \
     libxml2 \
     libxml2-dev \
     uuid-dev \
     libgeoip-dev \
     libtool \
     automake \
     vim \
     libyajl-dev
ADD nginx-${NGINX_VERSION}.tar.gz ./
RUN cd nginx-${NGINX_VERSION} \
    && ./configure \
        --prefix=/opt/app/nginx \
        --with-http_realip_module \
        --with-http_geoip_module \
        --with-http_ssl_module \
        --with-http_sub_module \
        --with-http_gunzip_module \
        --with-http_auth_request_module \
        --with-http_stub_status_module \
        --without-mail_pop3_module \
        --without-mail_imap_module \
        --without-mail_smtp_module \
        --with-stream \
        --with-stream_ssl_module \
        --with-stream_realip_module \
        --with-stream_geoip_module \
        --with-compat \
    && make \
    && make install \
    && rm -fr ../nginx-${NGINX_VERSION}.tar.gz
RUN git clone \
     --depth 1 \
     --branch ${MOD_SECURITY_VERIONS} \
     https://github.com/owasp-modsecurity/ModSecurity.git \
    && cd ModSecurity \
    && git submodule init \
    && git submodule update \
    && ./build.sh \
    && ./configure \
    && make \
    && make install
RUN git clone \
      --depth 1 \
      https://github.com/owasp-modsecurity/ModSecurity-nginx.git \
    && cd nginx-${NGINX_VERSION} \
    && ./configure \
         --with-compat \
         --prefix=/opt/app/nginx \
         --add-dynamic-module=../ModSecurity-nginx \
    && make modules \
    && mkdir -p /opt/app/nginx/modules \
    && mkdir -p /opt/app/nginx/conf/modsec \
    && cp objs/ngx_http_modsecurity_module.so /opt/app/nginx/modules
# configure modsecutiy and CRS
ADD modsecurity.conf /opt/app/nginx/conf/modsec
ADD main.conf /opt/app/nginx/conf/modsec
ADD unicode.mapping /opt/app/nginx/conf/modsec
ADD v4.12.0.tar.gz /opt/app/nginx/conf/modsec
RUN mv \
     /opt/app/nginx/conf/modsec/coreruleset-4.12.0/crs-setup.conf.example \
     /opt/app/nginx/conf/modsec/coreruleset-4.12.0/crs-setup.conf
# configure nginx
ADD nginx.conf /opt/app/nginx/conf



