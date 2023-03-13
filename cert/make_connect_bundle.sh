#!/bin/bash

server_name=test_server
connect_string=192.168.0.1:8089:ssl
user=$1
storepass=atakatak
user_p12=${user}.p12

if [[ -z "$user" ]]; then
  echo "usage: make_connect_bundle.sh username"
  exit 1
fi

if [[ ! -e cacert.key ]]; then
	echo "No ca cert found!"
	exit 1
fi

# make client cert
openssl req -sha256 -nodes -newkey rsa:2048 -out client.csr -keyout client.key \
 -subj "/CN=${user}/O=${user}"
openssl x509 -req -in client.csr -CA cacert.pem -CAkey cacert.key -CAcreateserial -out client.pem -days 1024 -extfile <(echo "extendedKeyUsage = clientAuth")
rm client.csr

# make client .p12
openssl pkcs12 -export -name client-cert -in client.pem -inkey client.key -out "${user_p12}" -passout pass:${storepass}
rm client.key client.pem

dir=$(mktemp -d /tmp/cert-XXXXXX)

mkdir -p "${dir}/MANIFEST"

cp truststore.p12 "${dir}/"
cp "${user_p12}" "${dir}/"

cat > "${dir}/${server_name}.pref" <<-EOF
<preferences>
  <preference version="1" name="cot_streams">
	<entry key="count" class="class java.lang.Integer">1</entry>
	<entry key="enabled0" class="class java.lang.Boolean">true</entry>
	<entry key="connectString0" class="class java.lang.String">${connect_string}</entry>
	<entry key="useAuth0" class="class java.lang.Boolean">false</entry>
	<entry key="description0" class="class java.lang.String">SSL connection to ${server_host}</entry>
  </preference>
  <preference version="1" name="com.atakmap.app_preferences">
	<entry key="clientPassword" class="class java.lang.String">${storepass}</entry>
	<entry key="caPassword" class="class java.lang.String">${storepass}</entry>
	<entry key="caLocation" class="class java.lang.String">/storage/emulated/0/atak/cert/truststore.p12</entry>
	<entry key="certificateLocation" class="class java.lang.String">/storage/emulated/0/atak/cert/${user_p12}</entry>
	<entry key="displayServerConnectionWidget" class="class java.lang.Boolean">true</entry>
  </preference>
</preferences>
EOF

cat > "${dir}/MANIFEST/manifest.xml" <<-EOF
<MissionPackageManifest version="2">
  <Configuration>
	<Parameter name="uid" value="${server_name}_config"/>
	<Parameter name="name" value="${server_name} config"/>
	<Parameter name="onReceiveDelete" value="true"/>
  </Configuration>
  <Contents>
	<Content ignore="false" zipEntry="${server_name}.pref"/>
	<Content ignore="false" zipEntry="truststore.p12"/>
	<Content ignore="false" zipEntry="${user_p12}"/>
  </Contents>
</MissionPackageManifest>
EOF

cd "$dir" || exit

zip -r "${server_name}_${user}.zip" ./*
cd -
mv "${dir}/${server_name}_${user}.zip" ./
rm -rf "$dir"
