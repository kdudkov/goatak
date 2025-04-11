#!/bin/bash

. "$(dirname "$(realpath "${BASH_SOURCE[0]}")")/params.sh"

SRV_NAME=${SRV_NAME:-$SRV_HOST}

if [[ ! -e ca-trusted.pem ]]; then
  echo "No ca cert found!"
  exit 1
fi

openssl pkcs12 -export -nokeys -name ${CA_NAME} -in ca-trusted.pem -out ca.p12 -passout pass:${PASS}

tmpdir=$(mktemp -d /tmp/cert-XXXXXX)

mkdir -p "${tmpdir}/MANIFEST"
mkdir -p "${tmpdir}/certs"

cp ca.p12 "${tmpdir}/certs/"

cat >"${tmpdir}/certs/${SRV_NAME}.pref" <<-EOF
<preferences>
  <preference version="1" name="cot_streams">
	<entry key="count" class="class java.lang.Integer">1</entry>
	<entry key="enabled0" class="class java.lang.Boolean">true</entry>
	<entry key="connectString0" class="class java.lang.String">${SRV_HOST}:8089:ssl</entry>
  <entry key="caLocation0" class="class java.lang.String">cert/ca.p12</entry>
  <entry key="caPassword0" class="class java.lang.String">${PASS}</entry>
  <entry key="enrollForCertificateWithTrust0" class="class java.lang.Boolean">true</entry>
	<entry key="useAuth0" class="class java.lang.Boolean">false</entry>
	<entry key="description0" class="class java.lang.String">SSL connection to ${SRV_NAME}</entry>
  </preference>
  <preference version="1" name="com.atakmap.app_preferences">
	<entry key="displayServerConnectionWidget" class="class java.lang.Boolean">true</entry>
  </preference>
</preferences>
EOF

cat >"${tmpdir}/MANIFEST/MANIFEST.xml" <<-EOF
<MissionPackageManifest version="2">
  <Configuration>
	<Parameter name="uid" value="${SRV_NAME}_config"/>
	<Parameter name="name" value="${SRV_NAME} config"/>
	<Parameter name="onReceiveDelete" value="true"/>
  </Configuration>
  <Contents>
	<Content ignore="false" zipEntry="certs/${SRV_NAME}.pref"/>
	<Content ignore="false" zipEntry="certs/ca.p12"/>
  </Contents>
</MissionPackageManifest>
EOF

cd "$tmpdir" || exit

zip -r "${SRV_NAME}_connect.zip" ./*
cd -
mv "${tmpdir}/${SRV_NAME}_connect.zip" ./
rm -rf "$tmpdir"
