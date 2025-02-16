{{ template "header" }}

<form action=adv method=post>
<input type=hidden name=action value="insert" />
<table style="text-align:left">
<tr><td>邮箱地址：</td><td><input type=text name=email size=30 /></td></tr>
<tr><td>密码：</td><td><input type=password name=passwd size=10 /></td></tr>
<tr><td>确认密码：</td><td><input type=password name=confirm size=10 /></td></tr>
<tr><td>公司名：</td><td><input type=text name=company size=30 /></td></tr>
<tr><td colspan=2><input type=submit value="注册" /></td></tr>
</table>
</form>

{{ template "footer" }}
