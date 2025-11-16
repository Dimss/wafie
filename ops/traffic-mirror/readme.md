## Wafie Traffic Mirror Sample Service

### Installation 

```bash
helm repo add wafie https://charts.wafie.io
helm repo update 
helm install wtm wafie/wafie-traffic-mirror \
 --set baseHost=$(ipconfig getifaddr en0).nip.io
```