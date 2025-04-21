upstream {{.UpstreamName}} {
  server {{.UpstreamHost}}:{{.UpstreamPort}};
}

server {
  listen {{.IngressPort}};
  server_name {{.IngressHost}};

  {{- if .ModSecEnabled }}
  modsecurity on;
  modsecurity_rules_file /opt/app/nginx/conf/modsec/main.conf;
  {{- end }}

  location / {
    proxy_pass http://{{.UpstreamName}};
    proxy_http_version 1.1;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection '';
    proxy_cache_bypass $http_upgrade;
  }
}