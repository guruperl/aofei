<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html xmlns="http://www.w3.org/1999/xhtml">
<head>
<meta http-equiv="Content-Type" content="text/html; charset=utf-8" />
<title>登录</title>
</head>

<body>
<table width=100% bgcolor="#ece9d8">
<tr><td>
<A HREF="/">主页</A>
</td><td align=right>
&nbsp;
</td></tr>
</table>

<h3>{{ .Other.Errorstr }}</h3>

<FORM METHOD="POST" ACTION="/goto/adv/e/login">
<INPUT TYPE="HIDDEN" NAME="{{ .Other.Go_uri_name }}" VALUE="{{ .Other.go_uri }}">
<pre>
账号 : <INPUT TYPE="TEXT"     NAME="{{ .Other.Login }}" size=20>
密码 : <INPUT TYPE="PASSWORD" NAME="{{ .Other.Password }}" size=20>
<INPUT TYPE="SUBMIT" VALUE=" 登录 ">
</pre>
</FORM>

<p>忘记密码？
<a href="/goto/web/e/adv?action=startretrieve">找回密码</a>.
<a href="/goto/web/e/adv?action=startnew">申请新账号</a>.
</p>

</body>
</html>
