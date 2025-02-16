package summer;

import (
    "testing"
)

func TestCampaign(t *testing.T) {
	c := CreateCampaign("", "", "", "", "", "")
	if c.ContentScore() != 0.0 ||
		c.VisualScore() != 0.0 ||
		c.ActScore()    != 0.0 ||
		c.DownloadScore()!=0.0 ||
		c.SpeedScore()  != 0.0 ||
		c.PostclickScore() != 0.0 {
		t.Errorf("%v", c.ToNames())
	}
	if c.Pack() != 149796 {
		t.Errorf("%v", c.Pack())
	}
}
