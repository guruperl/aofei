package tencent

import (
	"errors"
	"encoding/json"
)

var RetCodeMsgs = map[int]string {
0:"执行成功",
1:"全部执行失败",
2:"部分执行成功，部分执行失败",
3:"系统认证失败或者内部错误",
}

var ErrorCodeMsgs = map[int]string{
100:"系统错误，请稍后再试",
101:"错误的 DSP ID 或 token;DSP ID,Token 验证失败",
102:"业务处理的过程中发生错误，可能是参数错误，或者是传入的信息不合法之类的，请具体参考每个 API 的 ret_msg 说明",
300:"必须的参数没有传入",
301:"至少必须传入此组参数中的一个",
302:"参数必须为整数",
303:"参数格式错误",
304:"参数不在允许的值的范围内",
305:"参数不是正数",
306:"参数不是合法的 URL",
307:"参数太长，超出了允许的长度范围",
308:"参数不是合法的 YYYY-MM-DD 的日期",
309:"不是合法的 JSON 数据，无法被解析",
310:"可以被解析但是是空的 JSON 数据",
311:"参数太短，小于允许的长度范围",
312:"可以被解析的 JSON 数据，但不是 API 要求的 JSON 格式",
313:"参数不是 0 或正数",
314:"参数的值太大，超过了允许的最大值",
315:"参数的值不允许被修改",
400:"客户 ID 不存在/没有匹配的客户信息",
500:"客户名称重复",
503:"客户不能被修改",
504:"客户行业不合法",
505:"客户 URL 为空或者是 URL 不合法",
506:"客户 vocation 为空或者是 vocation 不合法",
507:"客户 area 为空",
508:"客户 qualification_class 不合法",
509:"内部错误，更新 DB 的过程中发生了错误",
510:"客户 qualification_files 不合法",
511:"客户 file_name 不合法",
512:"客户 file_url 不合法",
513:"不支持的客户资质文件的格式，目前支持的文件格式：jpg,jpeg,gif,png",
514:"内部错误，文件移动失败，可能是文件过大",
515:"客户 name 为空",
516:"客户 memo 为空",
601:"文件加载失败",
602:"未知的文件格式，文件的格式无法识别",
603:"不支持的文件格式,目前支持的文件格式：jpg,gif,png,swf,flv",
604:"Flv 素材获取不到时长信息",
605:"URL 对应的素材发生了变化，请换一个 URL",
606:"执行插入过程中发生了错误，请关注是否是同时上传",
607:"文件过大",
609:"素材 URL 为空或者是地址不合法",
610:"目标地址为空或者是地址不合法",
611:"客户名称为空",
612:"第三方曝光监测地址错误",
613:"素材过大，超过素材的大小限制",
614:"传入的 file_info 格式错误",
615:"URL 对应的客户发生变化，不能上传",
616:"同一次请求中，一个素材 URL 出现了多次",
801:"传入的结束时间大于开始时间",
802:"超过开始和结束时间限制，目前只支持一次查询少于 7 天的数据，如果时间段过长，请分开查询",
}

type Top struct {
	Ret_code int
	Ret_msg interface{}
	Error_code int
}

func GetTop(body []byte) *Top {
	parsed := new(Top)
	if err := json.Unmarshal(body, parsed); err != nil {
		panic(err)
	}
	return parsed
}

func CheckTop(body []byte) error {
	parsed := GetTop(body)
	if parsed.Error_code != 0 || parsed.Ret_code != 0 {
		e1 :=   RetCodeMsgs[parsed.Ret_code]
		e2 := ErrorCodeMsgs[parsed.Error_code]
		return errors.New(string(body) + "\n" + e1 + "\n" + e2)
	}
	return nil
}
