package holiday

const (
	GifPixel = "R0lGODlhAQABAJAAAP8AAAAAACH5BAUQAAAALAAAAAABAAEAAAICBAEAOw=="
	PngPixel = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABAQMAAAAl21bKAAAAA1BMVEUAAACnej3aAAAAAXRSTlMAQObYZgAAAApJREFUCNdjYAAAAAIAAeIhvDMAAAAASUVORK5CYII="
)

type Status struct {
	IsUser      bool
	IsDevice    bool
	IsIp        bool
	RequestType R_TYPE
	MimeType    M_TYPE
	IdSource    ID_SOURCE
	Source      T_SOURCE
}

type R_TYPE int

const (
	RequestUnknown R_TYPE = iota
	IMPR
	REQS
	CLIC
)

// M_TYPE 3 bits,
type M_TYPE int

const (
	MimeUnknown M_TYPE = iota
	JsonMime
	H5Mime
	JSMime
	ImageMime
	VideoMime
	AudioMime
)

// user id type, ID_SOURCE 3 bits

type ID_SOURCE int

const (
	IDSourceUnknown ID_SOURCE = iota
	SSPServer
	AndroidID
	IDFA
	OpenID
)

// traffic source, T_SOURCE 5 bits (32 values)

type T_SOURCE int

const (
	TrafficUnknown T_SOURCE = iota
	Browser
	MobileH5
	MobileNative
	PREBID
	DeviceSource
	EmailSource
	DSP
	AOFEI
)

func (self *Status) Pack() uint16 {
	d := uint16(0)

	if self.IsUser {
		d += (1 << 0)
	}
	if self.IsDevice {
		d += (1 << 1)
	}
	if self.IsIp {
		d += (1 << 2)
	}

	d += uint16((int(self.RequestType) & 3) << 14) // 2
	d += uint16((int(self.MimeType) & 7) << 11)    // 3
	d += uint16((int(self.IdSource) & 7) << 8)     // 3
	d += uint16((int(self.Source) & 31) << 3)      // 5

	return d
}

func UnpackStatus(status uint16) *Status {
	request := (status >> 14) & 3
	mime := (status >> 11) & 7
	idSource := (status >> 8) & 7
	source := (status >> 3) & 31

	isnew := false
	isspam := false
	ispsa := false
	if (status & 1) == 1 {
		isnew = true
	}
	if ((status >> 1) & 1) == 1 {
		isspam = true
	}
	if ((status >> 2) & 1) == 1 {
		ispsa = true
	}

	return &Status{isnew, isspam, ispsa, R_TYPE(request), M_TYPE(mime), ID_SOURCE(idSource), T_SOURCE(source)}
}

func (self *Status) MimeString() string {
	if self.MimeType == ImageMime {
		return "png"
	}
	if self.MimeType == VideoMime {
		return "mp4"
	}
	if self.MimeType == JSMime {
		return "js"
	}
	if self.MimeType == JsonMime {
		return "json"
	}
	if self.MimeType == H5Mime {
		return "html"
	}
	return ""
}

// ContentType generate correct content type according to extension
func (self *Status) ContentType() string {
	if self.MimeType == ImageMime {
		return "image/gif"
	}
	if self.MimeType == JSMime {
		return "application/javascript"
	}
	if self.MimeType == JsonMime {
		return "application/json"
	}
	if self.MimeType == H5Mime {
		return "text/html"
	}
	if self.MimeType == VideoMime {
		return "video/mp4"
	}
	return ""
}
