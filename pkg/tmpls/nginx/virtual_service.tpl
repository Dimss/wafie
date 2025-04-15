upstream wp-1 {
  server {{.InternalSvc}}:{{.UpstreamPort}};
}

server {
  listen {{.ListPort}};
  server_name {{.Hostname}};

  modsecurity {{.ModSecOn}};
  modsecurity_rules_file /opt/app/nginx/conf/modsec/main.conf;

  location / {
    proxy_pass {{.Upstream}};
    proxy_http_version 1.1;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection '';
    proxy_cache_bypass $http_upgrade;
  }
}