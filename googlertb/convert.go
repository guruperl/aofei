package googlertb

import (
	openrtb2 "github.com/prebid/openrtb/v20/openrtb2"
)

func ConvertBanner(banner *openrtb2.Banner) *BidRequest_Imp_Banner {
	w := int32(*banner.W)
	h := int32(*banner.H)
	pos := AdPosition(*banner.Pos)
	api := make([]APIFramework, 0)
	for _, item := range banner.API {
		api = append(api, APIFramework(item))
	}
	btype := make([]BannerAdType, 0)
	for _, item := range banner.BType {
		btype = append(btype, BannerAdType(item))
	}
	battr := make([]CreativeAttribute, 0)
	for _, item := range banner.BAttr {
		battr = append(battr, CreativeAttribute(item))
	}

	return &BidRequest_Imp_Banner{W: &w, H: &h, Pos: &pos, Api: api, Btype: btype, Battr: battr}
}

func ConvertImp(imp *openrtb2.Imp) *BidRequest_Imp {
	id := imp.ID
	tagid := imp.TagID
	instl := true
	if imp.Instl == 0 {
		instl = false
	}
	bidfloor := imp.BidFloor

	return &BidRequest_Imp{Id: &id, Tagid: &tagid, Instl: &instl, Bidfloor: &bidfloor, Banner: ConvertBanner(imp.Banner)}
}

func ConvertGeo(geo *openrtb2.Geo) *BidRequest_Geo {
	lat := geo.Lat
	lon := geo.Lon
	country := geo.Country
	region := geo.Region
	city := geo.City
	metro := geo.Metro
	zip := geo.ZIP

	return &BidRequest_Geo{Lat: lat, Lon: lon, Country: &country, Region: &region, City: &city, Metro: &metro, Zip: &zip}
}

func ConvertDevice(device *openrtb2.Device) *BidRequest_Device {
	dnt := true
	if device.DNT == nil || *device.DNT == 0 {
		dnt = false
	}
	js := true
	if device.JS == nil || *device.JS == 0 {
		js = false
	}
	ua := device.UA
	ip := device.IP
	dpidsha1 := device.DIDSHA1
	dpidmd5 := device.DPIDMD5
	carrier := device.Carrier
	language := device.Language
	maker := device.Make
	model := device.Model
	os := device.OS
	osv := device.OSV
	connectiontype := ConnectionType(*device.ConnectionType)
	devicetype := DeviceType(device.DeviceType)

	return &BidRequest_Device{Geo: ConvertGeo(device.Geo), Dnt: &dnt, Ua: &ua, Ip: &ip, Dpidsha1: &dpidsha1, Dpidmd5: &dpidmd5, Carrier: &carrier, Language: &language, Make: &maker, Model: &model, Os: &os, Osv: &osv, Js: &js, Connectiontype: &connectiontype, Devicetype: &devicetype}
}

func ConvertUser(user *openrtb2.User) *BidRequest_User {
	id := user.ID
	yob := int32(user.Yob)
	gender := user.Gender

	return &BidRequest_User{Id: &id, Yob: &yob, Gender: &gender}
}

func ConvertPublisher(publisher *openrtb2.Publisher) *BidRequest_Publisher {
	pub_id := publisher.ID
	pub_name := publisher.Name
	pub_domain := publisher.Domain

	return &BidRequest_Publisher{Id: &pub_id, Name: &pub_name, Domain: &pub_domain}
}

func ConverApp(app *openrtb2.App) *BidRequest_App {
	app_id := app.ID
	app_name := app.Name
	app_ver := app.Ver
	app_bundle := app.Bundle
	app_storeurl := app.StoreURL
	cat := make([]string, 0)
	for _, item := range app.Cat {
		cat = append(cat, item)
	}

	return &BidRequest_App{Id: &app_id, Name: &app_name, Ver: &app_ver, Bundle: &app_bundle, Storeurl: &app_storeurl, Cat: cat, Publisher: ConvertPublisher(app.Publisher)}
}

func ConvertBidRequest(req *openrtb2.BidRequest) *BidRequest {
	id := req.ID
	AT := AuctionType(req.AT)
	bcat := make([]string, 0)
	for _, item := range req.BCat {
		bcat = append(bcat, item)
	}
	badv := make([]string, 0)
	for _, item := range req.BAdv {
		badv = append(badv, item)
	}

	imp := req.Imp[0]
	return &BidRequest{Id: &id, At: &AT, Bcat: bcat, Badv: badv, Imp: []*BidRequest_Imp{ConvertImp(&imp)}, Device: ConvertDevice(req.Device), User: ConvertUser(req.User)}
}
