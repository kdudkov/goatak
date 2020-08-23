package main

import (
	"encoding/xml"
	"testing"

	"gotac/cot"
)

func TestParseEvent(t *testing.T) {
	evt := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
	<event version="2.0" uid="ANDROID-aabbcc5577" type="a-log-G-U-C" time="2020-08-19T08:01:32.157Z" start="2020-08-19T08:01:32.157Z" stale="2020-08-19T08:07:47.157Z" how="h-e">
<point lat="50.123" lon="30.123" hae="77.8140859592704" ce="9.9" le="9999999.0"/>
<detail>
<takv os="29" version="4.0.0.7 (7939f102).1592931989-CIV" device="XIAOMI MI 9T" platform="ATAK-CIV"/>
<contact endpoint="*:-1:stcp" callsign="kott"/>
<uid Droid="kott"/>
<precisionlocation altsrc="GPS" geopointsrc="GPS"/>
<__group role="Team Member" name="Dark Green"/>
<status battery="94"/>
<track course="225.41936519723822" speed="0.0"/>
</detail>
</event>
`

	e := &cot.Event{}

	_ = xml.Unmarshal([]byte(evt), e)

}
