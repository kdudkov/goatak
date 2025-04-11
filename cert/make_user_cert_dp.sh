#!/bin/bash

. "$(dirname "$(realpath "${BASH_SOURCE[0]}")")/params.sh"

user=$1
user_p12=${user}.p12
SRV_NAME=${SRV_NAME:-$SRV_HOST}

if [[ -z "$user" ]]; then
  echo "usage: $0 username"
  exit 1
fi

if [[ ! -e ca.key ]]; then
	echo "No ca cert found!"
	exit 1
fi

openssl pkcs12 -export -nokeys -name ${CA_NAME} -in ca.pem -out truststore.p12 -passout pass:${PASS}

cat >ext.cfg <<-EOT
basicConstraints=critical,CA:FALSE
keyUsage=critical, digitalSignature, keyEncipherment
extendedKeyUsage = critical, clientAuth
EOT

# make client cert
openssl req -sha256 -nodes -newkey rsa:2048 -out ${user}.csr -keyout ${user}.key \
  -subj "/O=${user}/CN=${user}"
openssl x509 -req -in ${user}.csr -CA ca.pem -CAkey ca.key -CAcreateserial -out ${user}.pem -days 1024 \
  -extfile ext.cfg
rm ${user}.csr ext.cfg

# make client .p12
openssl pkcs12 -export -name client-cert -in ${user}.pem -inkey ${user}.key -out ${user}.p12 -CAfile ca.pem \
  -passin pass:${PASS} -passout pass:${PASS}
rm ${user}.key ${user}.pem

tmpdir=$(mktemp -d /tmp/cert-XXXXXX)

mkdir -p "${tmpdir}/MANIFEST"
mkdir -p "${tmpdir}/certs"

cp truststore.p12 "${tmpdir}/certs/"
cp "${user_p12}" "${tmpdir}/certs/"

cat > "${tmpdir}/certs/${SRV_NAME}.pref" <<-EOF
<preferences>
  <preference version="1" name="cot_streams">
	<entry key="count" class="class java.lang.Integer">1</entry>
	<entry key="enabled0" class="class java.lang.Boolean">true</entry>
	<entry key="connectString0" class="class java.lang.String">${SRV_HOST}:8089:ssl</entry>
	<entry key="useAuth0" class="class java.lang.Boolean">false</entry>
	<entry key="description0" class="class java.lang.String">SSL connection to ${SRV_NAME}</entry>
  </preference>
  <preference version="1" name="com.atakmap.app_preferences">
	<entry key="caLocation" class="class java.lang.String">cert/truststore.p12</entry>
	<entry key="caPassword" class="class java.lang.String">${PASS}</entry>
	<entry key="certificateLocation" class="class java.lang.String">cert/${user_p12}</entry>
	<entry key="clientPassword" class="class java.lang.String">${PASS}</entry>
	<entry key="displayServerConnectionWidget" class="class java.lang.Boolean">true</entry>
  </preference>
</preferences>
EOF

cat > "${tmpdir}/MANIFEST/MANIFEST.xml" <<-EOF
<MissionPackageManifest version="2">
  <Configuration>
	<Parameter name="uid" value="${SRV_NAME}_config"/>
	<Parameter name="name" value="${SRV_NAME} config"/>
	<Parameter name="onReceiveDelete" value="true"/>
  </Configuration>
  <Contents>
	<Content ignore="false" zipEntry="certs/${SRV_NAME}.pref"/>
	<Content ignore="false" zipEntry="certs/truststore.p12"/>
	<Content ignore="false" zipEntry="certs/${user_p12}"/>
  </Contents>
</MissionPackageManifest>
EOF

cd "$tmpdir" || exit

zip -r "${SRV_NAME}_${user}.zip" ./*
cd -
mv "${tmpdir}/${SRV_NAME}_${user}.zip" ./
rm -rf "$tmpdir"
