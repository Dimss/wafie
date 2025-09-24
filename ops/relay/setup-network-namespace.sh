export network_name=ns1
ip netns add ns1
ip netns list
ip netns exec ns1 ip addr show
nsenter --net=/var/run/netns/ns1 bash -c "socat -dddd TCP-LISTEN:9090,fork TCP:1.1.1.1:8888 &"