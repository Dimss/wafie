FROM fedora:42
ENV GOPATH=$HOME/.go GOBIN=$HOME/.go/bin
ENV PATH=$PATH:$GOBIN
RUN dnf update -y \
    && dnf -y install go \
                      net-tools\
                      iputils \
                      openssh-server \
                      vim \
    && mkdir -p $HOME/.go/bin \
    && mkdir -p $HOME/.go/src/github.com \
    && ssh-keygen -A \
    && echo "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIH70dUHZIm2HemGywbFAdERdjm4Jh8gonsUv4xw5wqk2 dimssss@gmail.com" > /root/.ssh/authorized_keys

