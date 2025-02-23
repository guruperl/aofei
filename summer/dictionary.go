package summer

//	"github.com/golang/glog"

func Dictionary(word string) string {
	var ref = map[string]string{
		"IAB1":                                 "艺术娱乐",
		"IAB2":                                 "汽车",
		"IAB3":                                 "商务",
		"IAB4":                                 "求职",
		"IAB5":                                 "教育",
		"IAB6":                                 "家庭和育儿",
		"IAB7":                                 "健康健身",
		"IAB8":                                 "食品饮料",
		"IAB9":                                 "兴趣爱好",
		"IAB10":                                "家庭和庭院",
		"IAB11":                                "法律政治",
		"IAB12":                                "新闻",
		"IAB13":                                "个人理财",
		"IAB14":                                "社区",
		"IAB15":                                "科学",
		"IAB16":                                "宠物",
		"IAB17":                                "体育",
		"IAB18":                                "时尚流行",
		"IAB19":                                "技术和数码",
		"IAB20":                                "旅行",
		"IAB21":                                "房地产",
		"IAB22":                                "购物",
		"IAB23":                                "宗教和灵性",
		"IAB24":                                "其它类",
		"IAB25":                                "非主流类",
		"IAB26":                                "非法类",
		"Black":                                "黑名单",
		"White":                                "白名单",
		"Inherit":                              "默认",
		"Content":                              "内容",
		"Visual":                               "视觉",
		"Act":                                  "动作",
		"Download":                             "下载",
		"Speed":                                "速度",
		"Postclick":                            "点击后",
		"Internet":                             "互联网",
		"World":                                "全国",
		"Local":                                "当地",
		"Domain":                               "域名",
		"Age":                                  "年龄",
		"Popup":                                "跳出",
		"Crowd":                                "稠密度",
		"Traffic":                              "流量",
		"Source":                               "来源",
		"Control":                              "控制度",
		"Image":                                "图片",
		"Javascript":                           "JS",
		"Html":                                 "H5页面",
		"Video":                                "视频",
		"Web":                                  "网站",
		"Mobile":                               "应用App",
		"Email":                                "邮件",
		"Device":                               "设备终端",
		"Homepage":                             "首页",
		"Section":                              "二级",
		"Sub Section":                          "三级",
		"Rest":                                 "其它",
		"Unkown":                               "未知",
		"Married":                              "已婚",
		"Single":                               "未婚",
		"Scroll Up":                            "顶端",
		"Scroll Down":                          "底部",
		"Scroll Middle":                        "中段",
		"Sticky":                               "跟随",
		"Pop Under":                            "鼠标下",
		"Jump Screen":                          "满屏",
		"Normal":                               "正常",
		"Excellent":                            "优秀",
		"Good":                                 "良好",
		"Poor":                                 "差",
		"Poor/Negative etc.":                   "较差",
		"Ugly/Blank/Body etc.":                 "丑/空白/人体等",
		"Slow":                                 "慢",
		"Normal, No act":                       "正常",
		"Expand/Popup etc.":                    "展开或跳出",
		"Audio/Download etc.":                  "声音或下载",
		"Normal, No download":                  "正常无下载",
		"Paper/Document etc.":                  "文本文件",
		"Wallpaper/Software etc.":              "壁纸或软件",
		"Executable":                           "执行文件",
		"Top Brand":                            "名牌",
		"Brand":                                "品牌",
		"Dating etc.":                          "交友",
		"Provocative/Puzzle/Casino etc.":       "挑衅/猜谜/博彩",
		"Deceptive":                            "欺骗",
		"Good Looking Site":                    "美观网站",
		"Poor/Wrong Site":                      "不美观或错误网站",
		"Broken/Hangup":                        "链接失效或挂起",
		"Famous":                               "知名",
		"Sometimes":                            "时而知道",
		"Unknow/New":                           "未知或者新",
		"Normal Domain Name":                   "普通域名",
		"Top/Short Name":                       "顶级或简洁域名",
		"Poor Name":                            "差名名称",
		"Sub of Poor Domain Name":              "二级",
		"Normal, 1-10 Years":                   "正常，1-10年",
		"10-20 Years":                          "10-20年",
		"20 or more Years":                     "20年以上",
		"Less 1 Year":                          "少于一年",
		"1 Popups":                             "一个跳屏",
		"2-4 Popups":                           "2-4个跳屏",
		"5 or more popups":                     "5个或更多跳屏",
		"Normal 1-5 per page":                  "正常，每页1-5个广告",
		"1 or no ad":                           "1个或没有广告",
		"5-10 ads":                             "5-10广告",
		"10 or more ads":                       "10个或更多广告",
		"Normal Traffic":                       "正常流量",
		"Normal Site/App":                      "正常应用或网站",
		"Hijack/Plugin":                        "截获流量",
		"Proxy":                                "代理服务器流量",
		"Spiderware":                           "俘获流量",
		"Normal Managed":                       "管理正常",
		"Copied Site":                          "拷贝内容",
		"No or User Uploaded":                  "无管理或用户自我上传",
		"Year of Birth":                        "出生年",
		"Household":                            "家庭人数",
		"Gender":                               "性别",
		"Married Status":                       "婚否",
		"Income":                               "年收入",
		"Having Children":                      "有无子女",
		"Ethnicity":                            "人种",
		"Browser":                              "浏览器",
		"Operation System":                     "操作系统",
		"Platform":                             "硬件平台",
		"Animals":                              "动物",
		"Arts & Humanities":                    "艺术和人文",
		"Automotive":                           "汽车",
		"Beauty & Personal Care":               "美容和个人护理",
		"Business":                             "商务",
		"Computers & Electronics":              "电脑和电器",
		"Entertainment":                        "娱乐",
		"Finance & Insurance":                  "金融和保险",
		"Food & Drink":                         "食品饮料",
		"Games":                                "游戏",
		"Health":                               "健康",
		"Home & Garden":                        "居家和花园",
		"Industries":                           "工业界",
		"Lifestyles":                           "日常生活",
		"News & Current Events":                "新闻和热点",
		"Photo & Video":                        "图片视频",
		"Real Estate":                          "房地产",
		"Recreation":                           "娱乐",
		"Reference":                            "参考资料",
		"Science":                              "科学",
		"Shopping":                             "购物",
		"Social Networks & Online Communities": "在线社区",
		"Society":                              "社会",
		"Sports":                               "体育",
		"Telecommunications":                   "通讯",
		"Travel":                               "旅游",
		"UNDEFINED":                            "不定",
		"Occupation":                           "职业",
		"Yes":                                  "是",
		"No":                                   "否",
		"African":                              "非裔",
		"Asian":                                "亚裔",
		"Caucasian":                            "白人",
		"Hispanic":                             "西裔",
		"Native":                               "印第安裔",
		"M":                                    "男",
		"F":                                    "女",
		"O":                                    "其它",
		"BrowserUnknown":                       "未知浏览器",
		"BrowserChrome":                        "谷歌浏览器",
		"BrowserIE":                            "微软浏览器",
		"BrowserSafari":                        "苹果浏览器",
		"BrowserFirefox":                       "火狐浏览器",
		"BrowserAndroid":                       "安卓浏览器",
		"BrowserOpera":                         "Opera浏览器",
		"BrowserBlackberry":                    "黑莓浏览器",
		"BrowserUCBrowser":                     "UCBrowser",
		"BrowserSilk":                          "Silk",
		"BrowserNokia":                         "Nokia",
		"BrowserNetFront":                      "NetFront",
		"BrowserQQ":                            "QQ",
		"BrowserMaxthon":                       "Maxthon",
		"BrowserSogouExplorer":                 "搜狗",
		"BrowserSpotify":                       "Spotify",
		"BrowserBot":                           "Bot",
		"BrowserNintendo":                      "Nintendo",
		"BrowserSamsung":                       "Samsung",
		"BrowserAppleBot":                      "AppleBot",
		"BrowserBaiduBot":                      "BaiduBot",
		"BrowserBingBot":                       "BingBot",
		"BrowserDuckDuckGoBot":                 "DuckDuckGoBot",
		"BrowserFacebookBot":                   "FacebookBot",
		"BrowserGoogleBot":                     "GoogleBot",
		"BrowserLinkedInBot":                   "LinkedInBot",
		"BrowserMsnBot":                        "MsnBot",
		"BrowserPingdomBot":                    "PingdomBot",
		"BrowserTwitterBot":                    "TwitterBot",
		"BrowserYandexBot":                     "YandexBot",
		"BrowserYahooBot":                      "YahooBot",

		"DeviceUnknown":   "未知终端",
		"DeviceMobile":    "移动终端",
		"DevicePC":        "电脑",
		"DeviceConnected": "智能终端",
		"DeviceSetTopBox": "机顶盒",
		"DeviceComputer":  "电脑",
		"DeviceTablet":    "平板",
		"DevicePhone":     "手机",
		"DeviceConsole":   "智能终端",
		"DeviceWearable":  "智能穿戴",
		"DeviceTV":        "智能电视",

		"PositionUnknown":    "未定位置",
		"PositionAboveFold":  "当前出现",
		"PositionLocked":     "跟随",
		"PositionBelowFold":  "下滑后出现",
		"PositionHeader":     "抬头框",
		"PositionFooter":     "结尾框",
		"PositionSideBar":    "左右导读框",
		"PositionFullScreen": "全屏显示",

		"ContentVideo":   "视频内容",
		"ContentGame":    "游戏内容",
		"ContentMusic":   "音乐内容",
		"ContentApp":     "应用App内容",
		"ContentText":    "文字内容",
		"ContentOther":   "其它内容",
		"ContentUnknown": "未定内容",

		"AttrUnknown":                "普通类",
		"AttrAudioAuto":              "自动播放声音",
		"AttrAudioUser":              "用户触发声音",
		"AttrExpandableAuto":         "自动扩展",
		"AttrExpandableUserClick":    "用户点击后扩展",
		"AttrExpandableUserRollover": "用户接触后扩展",
		"AttrVideoAuto":              "自动播放视频",
		"AttrVideoUser":              "用户触发播放视频",
		"AttrPop":                    "跳出",
		"AttrProvocative":            "攻击性图片",
		"AttrExtremeAnimation":       "抖动闪耀或类似动画",
		"AttrSurvey":                 "普查",
		"AttrTextOnly":               "只有文字",
		"AttrInteractive":            "内置游戏类",
		"AttrWindowsDialog":          "视窗对话",
		"AttrHasAudioToggleButton":   "带声音按钮",
		"AttrHasSkipButton":          "带下页按钮",
		"AttrFlash":                  "FLASH内容",
		"AttrResponsive":             "自动调节大小类",

		"MimeUnknown": "未知MIME",
		"XHTMLText":   "Mobile文字",
		"XHTMLBanner": "MobileH5",
		"JSMime":      "JS脚本类",
		"Iframe":      "Iframe类",

		"OSUnknown":            "未知系统",
		"OSWindowsPhone":       "微软手机",
		"OSWindows":            "微软视窗",
		"OSMacOSX":             "MacOS",
		"OSiOS":                "iOS",
		"OSAndroid":            "安卓",
		"OSBlackberry":         "黑莓系统",
		"OSChromeOS":           "Chrome系统",
		"OSKindle":             "Kindle",
		"OSWebOS":              "WebOS系统",
		"OSLinux":              "Linux操作系统",
		"OSPlaystation":        "Playstation系统",
		"OSXbox":               "Xbox系统",
		"OSNintendo":           "Nintendo系统",
		"OSBot":                "机器人系统",
		"PlatformUnknown":      "未知平台",
		"PlatformWindows":      "Windows",
		"PlatformMac":          "Mac",
		"PlatformLinux":        "Linux",
		"PlatformiPad":         "iPad",
		"PlatformiPhone":       "iPhone",
		"PlatformiPod":         "iPod",
		"PlatformBlackberry":   "黑莓",
		"PlatformWindowsPhone": "微软手机平台",
		"PlatformPlaystation":  "Playstation平台",
		"PlatformXbox":         "Xbox平台",
		"PlatformNintendo":     "Nintendo平台",
		"PlatformBot":          "机器人平台",
		"Sun":                  "星期日",
		"Mon":                  "周一",
		"Tue":                  "周二",
		"Wed":                  "周三",
		"Thu":                  "周四",
		"Fri":                  "周五",
		"Sat":                  "星期六",
	}
	if chinese, ok := ref[word]; ok {
		return chinese
	}
	return word
}

func Translate(obj interface{}) interface{} {
	switch t := obj.(type) {
	case string:
		return Dictionary(t)
	case map[string]map[uint32][]interface{}:
		out := make(map[string]map[uint32][]interface{})
		for key, val := range t {
			out[key] = make(map[uint32][]interface{})
			for i, v := range val {
				out[key][i] = make([]interface{}, 0)
				for j, item := range v {
					if j == 0 {
						out[key][i] = append(out[key][i], Dictionary(item.(string)))
					} else {
						out[key][i] = append(out[key][i], item)
					}
				}
			}
		}
		return out
	case map[uint32][]interface{}:
		out := make(map[uint32][]interface{})
		for i, v := range t {
			out[i] = make([]interface{}, 0)
			for j, item := range v {
				if j == 0 {
					out[i] = append(out[i], Dictionary(item.(string)))
				} else {
					out[i] = append(out[i], item)
				}
			}
		}
		return out
	case map[string]string:
		out := make(map[string]string)
		for k, v := range t {
			out[k] = Dictionary(v)
		}
		return out
	case []string:
		out := make([]string, 0)
		for _, v := range t {
			out = append(out, Dictionary(v))
		}
		return out
	case map[string]map[uint32]string:
		out := make(map[string]map[uint32]string)
		for k, v := range t {
			item := make(map[uint32]string)
			for key, val := range v {
				item[key] = Dictionary(val)
			}
			out[k] = item
		}
		return out
	default:
	}
	return obj
}

func TranslateOne(obj interface{}, nameIn, nameOut string) {
	switch t := obj.(type) {
	case map[string][]map[string]interface{}:
		for k, items := range t {
			for i, item := range items {
				t[k][i][nameOut] = Dictionary(item[nameIn].(string))
			}
		}
	case []map[string]interface{}:
		for _, v := range t {
			v[nameOut] = Dictionary(v[nameIn].(string))
		}
	case map[string]interface{}:
		t[nameOut] = Dictionary(t[nameIn].(string))
	default:
	}
}
