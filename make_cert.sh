#!/bin/bash

server_name=test_server
server_ip=192.168.0.10
server_host=host.com
server_port=58088
user=user

if [[ ! -e ca.key ]]; then
openssl req -x509 -sha256 -nodes -newkey rsa:2048 -days 1825 -out ca.pem -keyout ca.key -subj "/CN=${server_ip}/C=US/ST=CA/O=${server_name}"
fi

rm -f *.p12
openssl req -x509 -sha256 -nodes -newkey rsa:2048 -days 1825 -out client.pem -keyout client.key -subj "/CN=${user}"

openssl pkcs12 -export -name client-cert -in client.pem -inkey client.key -out ${server_name}.p12 -passout pass:${storepass}

keytool -import -alias server-cert -file ca.pem -keystore truststore.p12 -storepass ${storepass} -trustcacerts -noprompt -storetype pkcs12
keytool -import -alias client-cert -file client.pem -keystore truststore.p12 -storepass ${storepass} -trustcacerts -noprompt -storetype pkcs12

uuid=$(uuidgen)

mkdir -p /tmp/cert/$uuid
mkdir -p /tmp/cert/MANIFEST

cp truststore.p12 /tmp/cert/$uuid/
cp ${server_name}.p12 /tmp/cert/$uuid/

cat > /tmp/cert/${uuid}/${server_name}.pref <<-EOF
<preferences>
  <preference version="1" name="cot_streams">
    <entry key="count" class="class java.lang.Integer">1</entry>
    <entry key="description0" class="class java.lang.String">SSL connection to ${server_host}</entry>
    <entry key="enabled0" class="class java.lang.Boolean">true</entry>
    <entry key="connectString0" class="class java.lang.String">${server_host}:${server_port}:ssl</entry>
  </preference>
  <preference version="1" name="com.atakmap.app_preferences">
    <entry key="displayServerConnectionWidget" class="class java.lang.Boolean">true</entry>
    <entry key="caLocation" class="class java.lang.String">/storage/emulated/0/atak/cert/truststore.p12</entry>
    <entry key="caPassword" class="class java.lang.String">${storepass}</entry>
    <entry key="certificateLocation" class="class java.lang.String">/storage/emulated/0/atak/cert/${server_name}.p12</entry>
    <entry key="clientPassword" class="class java.lang.String">${storepass}</entry>
  </preference>
</preferences>
EOF

cat > /tmp/cert/MANIFEST/manifest.xml <<-EOF
<MissionPackageManifest version="2">
  <Configuration>
    <Parameter name="uid" value="$(uuidgen)"/>
    <Parameter name="name" value="${server_name} config"/>
    <Parameter name="onReceiveDelete" value="true"/>
  </Configuration>
  <Contents>
    <Content ignore="false" zipEntry="${uuid}/${server_name}.pref"/>
    <Content ignore="false" zipEntry="${uuid}/truststore.p12"/>
    <Content ignore="false" zipEntry="${uuid}/${server_name}.p12"/>
  </Contents>
</MissionPackageManifest>
EOF

cd /tmp/cert

zip -r "${server_name}.zip" ./*
cd -
mv "/tmp/cert/${server_name}.zip" ./
rm -rf /tmp/cert