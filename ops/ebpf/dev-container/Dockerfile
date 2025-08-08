FROM fedora:42
ENV GOPATH=$HOME/.go GOBIN=$HOME/.go/bin
ENV PATH=$PATH:$GOBIN
RUN dnf update -y \
    && dnf -y install go net-tools clang llvm-strip \
    && mkdir -p $HOME/.go/bin \
    && mkdir -p $HOME/.go/src/github.com \
    && cd $HOME/.go/src/github.com \
    && git clone https://github.com/cilium/ebpf.git

