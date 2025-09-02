# create ssh
curl -X 'POST' \
  'http://128.199.227.169:37849/api/v1/vpn/ssh/create' \
  -H 'accept: application/json' \
  -H 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NTY4ODg5NTksImlhdCI6MTc1NjgwMjU1OSwidXNlcl9pZCI6ImFkbWluIiwidXNlcm5hbWUiOiJhZG1pbiJ9.ZAzdliOOkRuNyKvXY_08jdpPl-OUOpJ4Ct9-sRG0Uco' \
  -H 'Content-Type: application/json' \
  -d '{
  "days": 1,
  "email": "fdfdsf",
  "password": "dfdsfsdfdf",
  "protocol": "ssh",
  "username": "stffsdfdsffdsfring"
}'

{
  "success": true,
  "message": "SSH user created successfully",
  "data": {
    "protocol": "ssh",
    "server": "grn.mirrorfast.my.id",
    "port": 22,
    "username": "stffsdfdsffdsfring",
    "password": "dfdsfsdfdf",
    "config": {
      "ssh_port": "22",
      "ssl_port": "443",
      "stunnel_port": "444",
      "ws_port": "80"
    }
  }
}

# untuk renex atau extend ssh 

curl -X 'PUT' \
  'http://128.199.227.169:37849/api/v1/vpn/ssh/users/stffsdfdsffdsfring/extend' \
  -H 'accept: application/json' \
  -H 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NTY4ODg5NTksImlhdCI6MTc1NjgwMjU1OSwidXNlcl9pZCI6ImFkbWluIiwidXNlcm5hbWUiOiJhZG1pbiJ9.ZAzdliOOkRuNyKvXY_08jdpPl-OUOpJ4Ct9-sRG0Uco' \
  -H 'Content-Type: application/json' \
  -d '{
  "days": 30
}'

{
  "success": true,
  "message": "SSH user extended successfully"
}

# create trojan 
curl -X 'POST' \
  'http://128.199.227.169:37849/api/v1/vpn/trojan/create' \
  -H 'accept: application/json' \
  -H 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NTY4ODg5NTksImlhdCI6MTc1NjgwMjU1OSwidXNlcl9pZCI6ImFkbWluIiwidXNlcm5hbWUiOiJhZG1pbiJ9.ZAzdliOOkRuNyKvXY_08jdpPl-OUOpJ4Ct9-sRG0Uco' \
  -H 'Content-Type: application/json' \
  -d '{
  "days": 1,
  "email": "dfddf",
  "password": "bsdbffs",
  "protocol": "trojan",
  "username": "fddfdfasd"
}'

{
  "success": true,
  "message": "Trojan user created successfully",
  "data": {
    "protocol": "trojan",
    "server": "grn.mirrorfast.my.id",
    "port": 443,
    "username": "fddfdfasd",
    "password": "7291a7da-9cdc-4a69-9936-f5e03686107e",
    "config": {
      "config_url": "http://grn.mirrorfast.my.id:81/trojan-fddfdfasd.txt",
      "expired_on": "2025-09-03",
      "host": "grn.mirrorfast.my.id",
      "key": "7291a7da-9cdc-4a69-9936-f5e03686107e",
      "link_go": "trojan-go://7291a7da-9cdc-4a69-9936-f5e03686107e@grn.mirrorfast.my.id:443?path=%2Ftrojan-ws&security=tls&host=grn.mirrorfast.my.id&type=ws&sni=grn.mirrorfast.my.id#TROJANGO_fddfdfasd",
      "link_grpc": "trojan://7291a7da-9cdc-4a69-9936-f5e03686107e@grn.mirrorfast.my.id:443?mode=gun&security=tls&type=grpc&serviceName=trojan-grpc&sni=grn.mirrorfast.my.id#TROJAN_GRPC_fddfdfasd",
      "link_ws": "trojan://7291a7da-9cdc-4a69-9936-f5e03686107e@grn.mirrorfast.my.id:443?path=%2Ftrojan-ws&security=tls&host=grn.mirrorfast.my.id&type=ws&sni=grn.mirrorfast.my.id#TROJAN_WS_fddfdfasd",
      "network": "ws/grpc",
      "path": "/trojan-ws",
      "port": "443",
      "remarks": "fddfdfasd",
      "serviceName": "trojan-grpc"
    }
  }
}

# extend / renew trojan
curl -X 'PUT' \
  'http://128.199.227.169:37849/api/v1/vpn/trojan/users/fddfdfasd/extend' \
  -H 'accept: application/json' \
  -H 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NTY4ODg5NTksImlhdCI6MTc1NjgwMjU1OSwidXNlcl9pZCI6ImFkbWluIiwidXNlcm5hbWUiOiJhZG1pbiJ9.ZAzdliOOkRuNyKvXY_08jdpPl-OUOpJ4Ct9-sRG0Uco' \
  -H 'Content-Type: application/json' \
  -d '{
  "days": 30
}'

{
  "success": true,
  "message": "Trojan user extended successfully"
}

# create vless 
curl -X 'POST' \
  'http://128.199.227.169:37849/api/v1/vpn/vless/create' \
  -H 'accept: application/json' \
  -H 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NTY4ODg5NTksImlhdCI6MTc1NjgwMjU1OSwidXNlcl9pZCI6ImFkbWluIiwidXNlcm5hbWUiOiJhZG1pbiJ9.ZAzdliOOkRuNyKvXY_08jdpPl-OUOpJ4Ct9-sRG0Uco' \
  -H 'Content-Type: application/json' \
  -d '{
  "days": 1,
  "email": "fdsdsfdd",
  "password": "huhfddfdf",
  "protocol": "vless",
  "username": "fdfdfdfsa"
}'

{
  "success": true,
  "message": "VLESS user created successfully",
  "data": {
    "protocol": "vless",
    "server": "grn.mirrorfast.my.id",
    "port": 443,
    "username": "fdfdfdfsa",
    "uuid": "9df1306b-0ffa-4488-a787-c20f1d22eb66",
    "config": {
      "config_url": "http://grn.mirrorfast.my.id:81/vless-fdfdfdfsa.txt",
      "encryption": "none",
      "expired_on": "2025-09-03",
      "host": "grn.mirrorfast.my.id",
      "link_grpc": "vless://9df1306b-0ffa-4488-a787-c20f1d22eb66@grn.mirrorfast.my.id:443?mode=gun&security=tls&encryption=none&type=grpc&serviceName=vless-grpc&sni=grn.mirrorfast.my.id#VLESS_GRPC_fdfdfdfsa",
      "link_ntls": "vless://9df1306b-0ffa-4488-a787-c20f1d22eb66@grn.mirrorfast.my.id:80?type=ws&encryption=none&security=none&host=grn.mirrorfast.my.id&path=/vless#XRAY_VLESS_NTLS_fdfdfdfsa",
      "link_tls": "vless://9df1306b-0ffa-4488-a787-c20f1d22eb66@grn.mirrorfast.my.id:443?type=ws&encryption=none&security=tls&host=grn.mirrorfast.my.id&path=/vless&allowInsecure=1&sni=grn.mirrorfast.my.id#XRAY_VLESS_TLS_fdfdfdfsa",
      "network": "ws/grpc",
      "path": "/vless",
      "port_ntls": "80",
      "port_tls": "443",
      "remarks": "fdfdfdfsa",
      "serviceName": "vless-grpc",
      "uuid": "9df1306b-0ffa-4488-a787-c20f1d22eb66"
    }
  }
}



# extend vless 

curl -X 'PUT' \
  'http://128.199.227.169:37849/api/v1/vpn/vless/users/fdfdfdfsa/extend' \
  -H 'accept: application/json' \
  -H 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NTY4ODg5NTksImlhdCI6MTc1NjgwMjU1OSwidXNlcl9pZCI6ImFkbWluIiwidXNlcm5hbWUiOiJhZG1pbiJ9.ZAzdliOOkRuNyKvXY_08jdpPl-OUOpJ4Ct9-sRG0Uco' \
  -H 'Content-Type: application/json' \
  -d '{
  "days": 1
}'


{
  "success": true,
  "message": "VLESS user extended successfully"
}

# create vmess 

curl -X 'POST' \
  'http://128.199.227.169:37849/api/v1/vpn/vmess/create' \
  -H 'accept: application/json' \
  -H 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NTY4ODg5NTksImlhdCI6MTc1NjgwMjU1OSwidXNlcl9pZCI6ImFkbWluIiwidXNlcm5hbWUiOiJhZG1pbiJ9.ZAzdliOOkRuNyKvXY_08jdpPl-OUOpJ4Ct9-sRG0Uco' \
  -H 'Content-Type: application/json' \
  -d '{
  "days": 1,
  "email": "dfsfdssce",
  "password": "strgdsfving",
  "protocol": "vmess",
  "username": "strvsdfvvmessing"
}'


{
  "success": true,
  "message": "VMESS user created successfully",
  "data": {
    "protocol": "vmess",
    "server": "grn.mirrorfast.my.id",
    "port": 443,
    "username": "strvsdfvvmessing",
    "uuid": "1e87bfd9-4e29-45f7-9b88-0c88004ab5ef",
    "config": {
      "alterId": "0",
      "config_url": "http://grn.mirrorfast.my.id:81/vmess-strvsdfvvmessing.txt",
      "expired_on": "2025-09-03",
      "host": "grn.mirrorfast.my.id",
      "link_grpc": "vmess://1e87bfd9-4e29-45f7-9b88-0c88004ab5ef@grn.mirrorfast.my.id:443?mode=gun&security=tls&encryption=none&type=grpc&serviceName=vmess-grpc&sni=grn.mirrorfast.my.id#VMESS_GRPC_strvsdfvvmessing",
      "link_ws": "vmess://1e87bfd9-4e29-45f7-9b88-0c88004ab5ef@grn.mirrorfast.my.id:443?type=ws&encryption=none&security=tls&host=grn.mirrorfast.my.id&path=/vmess&allowInsecure=1&sni=grn.mirrorfast.my.id#VMESS_WS_strvsdfvvmessing",
      "network": "ws/grpc",
      "path": "/vmess",
      "port": "443",
      "remarks": "strvsdfvvmessing",
      "security": "auto",
      "serviceName": "vmess-grpc",
      "uuid": "1e87bfd9-4e29-45f7-9b88-0c88004ab5ef"
    }
  }
}


# extend vmess 

curl -X 'PUT' \
  'http://128.199.227.169:37849/api/v1/vpn/vmess/users/strvsdfvvmessing/extend' \
  -H 'accept: application/json' \
  -H 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NTY4ODg5NTksImlhdCI6MTc1NjgwMjU1OSwidXNlcl9pZCI6ImFkbWluIiwidXNlcm5hbWUiOiJhZG1pbiJ9.ZAzdliOOkRuNyKvXY_08jdpPl-OUOpJ4Ct9-sRG0Uco' \
  -H 'Content-Type: application/json' \
  -d '{
  "days": 30
}'

{
  "success": true,
  "message": "VMESS user extended successfully"
}
