package main

import (
	"encoding/xml"
	"testing"

	"github.com/kdudkov/goatak/pkg/cot"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseEvent(t *testing.T) {
	data := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
	<event version="2.0" uid="ANDROID-aabbcc5577" type="a-f-G-U-C" time="2020-08-19T08:01:32.157Z" start="2020-08-19T08:01:32.157Z" stale="2020-08-19T08:07:47.157Z" how="h-e">
<point lat="50.123" lon="30.123" hae="77.8140859592704" ce="9.9" le="9999999.0"/>
<detail>
<takv os="29" version="4.0.0.7 (7939f102).1592931989-CIV" device="XIAOMI MI 9T" platform="ATAK-CIV"/>
<contact endpoint="*:-1:stcp" callsign="cs"/>
<uid Droid="cs"/>
<precisionlocation altsrc="GPS" geopointsrc="GPS"/>
<__group role="Team Member" name="Dark Green"/>
<status battery="94"/>
<track course="225.41936519723822" speed="0.0"/>
</detail>
</event>
`

	e := &cot.Event{}

	require.NoError(t, xml.Unmarshal([]byte(data), e))

	assert.Equal(t, "a-f-G-U-C", e.Type)

	c, err := cot.EventToProto(e)
	require.NoError(t, err)

	assert.Equal(t, "ANDROID-aabbcc5577", c.GetUid())
	assert.Equal(t, "a-f-G-U-C", c.GetType())
	assert.Equal(t, "cs", c.GetCallsign())
	assert.Equal(t, "Team Member", c.GetRole())
}
