package pzutil

const (
	SSPHandler = "/pz"
	BIDHandler = "/bid"

	CLK = "/cz"
	WIN = "/win"

	GifPixel = "R0lGODlhAQABAJAAAP8AAAAAACH5BAUQAAAALAAAAAABAAEAAAICBAEAOw=="
	PngPixel = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABAQMAAAAl21bKAAAAA1BMVEUAAACnej3aAAAAAXRSTlMAQObYZgAAAApJREFUCNdjYAAAAAIAAeIhvDMAAAAASUVORK5CYII="
)

// TRequest 2 bits, 4 values
type TRequest int

const (
	UnknownRequest TRequest = iota
	IMPR
	REQS
	CLIC
)

// TStyle 2 bits, 4 values
type TStyle int

const (
	UnknownStyle TStyle = iota
	BANNER
	VIDEO
	NATIVE
)

// TIDSource 3 bits
type TIDSource int

const (
	UnknownIDSource TIDSource = iota
	SSPServer
	IMEI
	IDFA
	EXCHANGE
)

// TSource 3 bits (8 values)
type TSource int

const (
	UnknownSource TSource = iota
	BROWSER
	MOBILE
	SDK
	PREBID
	DSP
	AOFEI
)

func StringToSource(s string) TSource {
	switch s {
	case "browser":
		return BROWSER
	case "mobile":
		return MOBILE
	case "sdk":
		return SDK
	case "prebid":
		return PREBID
	case "dsp":
		return DSP
	case "aofei":
		return AOFEI
	default:
		return UnknownSource
	}
}

type Status struct {
	IsNew      bool
	IsDailyNew bool
	IsPSA      bool
	Request    TRequest
	Mime       TMime
	IDSource   TIDSource
	Source     TSource
	Style      TStyle
}

func (self *Status) Pack() uint16 {
	d := uint16(0)
	if self.IsNew {
		d += (1 << 0)
	}
	if self.IsDailyNew {
		d += (1 << 1)
	}
	if self.IsPSA {
		d += (1 << 2)
	}

	d += uint16((int(self.Request) & 3) << 14) //2
	d += uint16((int(self.Mime) & 7) << 11)    //3
	d += uint16((int(self.IDSource) & 7) << 8) //3
	d += uint16((int(self.Source) & 7) << 5)   //3
	d += uint16((int(self.Style) & 3) << 3)    //2

	return d
}

func UnpackStatus(status uint16) *Status {
	request := (status >> 14) & 3
	mime := (status >> 11) & 7
	idSource := (status >> 8) & 7
	source := (status >> 5) & 7
	style := (status >> 3) & 3

	isnew := false
	isdailynew := false
	ispsa := false
	if (status & 1) == 1 {
		isnew = true
	}
	if ((status >> 1) & 1) == 1 {
		isdailynew = true
	}
	if ((status >> 2) & 1) == 1 {
		ispsa = true
	}

	return &Status{
		IsNew:      isnew,
		IsDailyNew: isdailynew,
		IsPSA:      ispsa,
		Request:    TRequest(request),
		Mime:       TMime(mime),
		IDSource:   TIDSource(idSource),
		Source:     TSource(source),
		Style:      TStyle(style),
	}
}
