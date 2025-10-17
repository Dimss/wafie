FROM fedora:42
ENV GOPATH=/root/.go
ENV GOBIN=$GOPATH/bin
ENV PATH=$PATH:$GOBIN
RUN dnf update -y \
    && dnf -y install go \
                      net-tools\
                      iputils \
                      openssh-server \
                      vim \
                      nft \
                      socat \
                      procps \
                      bash-completion \
                      iproute \
                      wget \
    && echo 'source /usr/share/git-core/contrib/completion/git-prompt.sh' >> /root/.bashrc \
    && echo "export PS1='\[\033[1;32m\]\W\[\033[1;96m\]\$(__git_ps1 \" (%s)\") #\[\033[m\] '" >> /root/.bashrc \
    && mkdir -p $HOME/.go/bin \
    && mkdir -p $HOME/.go/src/github.com \
    && go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest \
    && ssh-keygen -A \
    && echo "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIH70dUHZIm2HemGywbFAdERdjm4Jh8gonsUv4xw5wqk2 dimssss@gmail.com" > /root/.ssh/authorized_keys \
    && BIN="/usr/local/bin" \
    && VERSION="1.57.2" \
    && curl -sSL \
       "https://github.com/bufbuild/buf/releases/download/v${VERSION}/buf-$(uname -s)-$(uname -m)" \
       -o "${BIN}/buf" \
    && chmod +x "${BIN}/buf" \
    && git config --global user.email "dimssss@gmail.com" \
    && git config --global user.name "Dmitry Kartsev"

